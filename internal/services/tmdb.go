package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"seyirlik.net/api/internal/cache"
	"seyirlik.net/api/internal/models"
	"seyirlik.net/api/internal/repository"
)

// Hata tanımlamaları
var (
	ErrNotFound    = errors.New("içerik bulunamadı")
	ErrRateLimit   = errors.New("TMDB istek limiti aşıldı")
	ErrServerError = errors.New("TMDB sunucu hatası")
	ErrBadRequest  = errors.New("geçersiz istek")
)

// Cache TTL süreleri
const (
	searchCacheTTL = 5 * time.Minute  // Arama sonuçları 5 dakika
	detailCacheTTL = 30 * time.Minute // Detay bilgileri 30 dakika
)

// CacheStats cache istatistiklerini tutar
type CacheStats struct {
	Search cache.Stats `json:"search"`
	Detail cache.Stats `json:"detail"`
}

// TMDBClient TMDB servisinin interface'i - test edilebilirlik için
type TMDBClient interface {
	// Arama
	SearchMulti(ctx context.Context, query string, page int) (*models.MultiSearchResponse, error)

	// Film
	GetMovieDetail(ctx context.Context, id int) (*models.Movie, error)
	GetWatchProviders(ctx context.Context, id int) (*models.WatchProviderResult, error)
	GetCredits(ctx context.Context, id int) (*models.Credits, error)

	// Dizi
	GetTVDetail(ctx context.Context, id int) (*models.TVShow, error)
	GetTVWatchProviders(ctx context.Context, id int) (*models.WatchProviderResult, error)
	GetTVCredits(ctx context.Context, id int) (*models.Credits, error)

	// Health check
	Ping(ctx context.Context) error
	GetCacheStats() CacheStats
}

// TMDBService interface'i implemente eder
var _ TMDBClient = (*TMDBService)(nil)

type TMDBService struct {
	apiKey        string
	baseURL       string
	httpClient    *http.Client
	imageBase     string
	searchCache   *cache.Cache
	detailCache   *cache.Cache
	contentRepo   *repository.ContentRepository
	searchLogRepo *repository.SearchLogRepository
	logger        *slog.Logger
}

// Yeni bir TMDBService oluşturur
func NewTMDBService(apiKey string, contentRepo *repository.ContentRepository, searchLogRepo *repository.SearchLogRepository) *TMDBService {
	return &TMDBService{
		apiKey:  apiKey,
		baseURL: "https://api.themoviedb.org/3",
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		imageBase:     "https://image.tmdb.org/t/p/w500",
		searchCache:   cache.New(searchCacheTTL),
		detailCache:   cache.New(detailCacheTTL),
		contentRepo:   contentRepo,
		searchLogRepo: searchLogRepo,
		logger:        slog.Default(),
	}
}

// TMDB'ye GET isteği atan yardımcı fonksiyon
// Her endpoint için tekrar tekrar aynı kodu yazmamak için
func (s *TMDBService) get(ctx context.Context, endpoint string, params url.Values) ([]byte, error) {
	// API key'i her isteğe otomatik ekle
	params.Set("api_key", s.apiKey)
	params.Set("language", "tr-TR") // Türkçe içerik

	fullURL := fmt.Sprintf("%s%s?%s", s.baseURL, endpoint, params.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("istek oluşturma hatası: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("istek hatası: %w", err)
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	// HTTP status code'a göre hata döndür
	switch resp.StatusCode {
	case http.StatusOK:
		// OK, devam et
	case http.StatusNotFound:
		return nil, ErrNotFound
	case http.StatusTooManyRequests:
		return nil, ErrRateLimit
	case http.StatusBadRequest:
		return nil, ErrBadRequest
	default:
		if resp.StatusCode >= 500 {
			return nil, ErrServerError
		}
		return nil, fmt.Errorf("TMDB hata kodu: %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

// Film arama — /search?q=inception
func (s *TMDBService) SearchMovies(ctx context.Context, query string, page int) (*models.SearchResponse, error) {
	params := url.Values{}
	params.Set("query", query)
	params.Set("page", fmt.Sprintf("%d", page))

	body, err := s.get(ctx, "/search/movie", params)
	if err != nil {
		return nil, err
	}

	var result models.SearchResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("JSON parse hatası: %w", err)
	}

	// Poster URL'lerini tam hale getir
	for i := range result.Results {
		if result.Results[i].PosterPath != "" {
			result.Results[i].PosterPath = s.imageBase + result.Results[i].PosterPath
		}
	}

	return &result, nil
}

// Multi search — Film ve dizi birlikte arama
// 3 katmanlı cache: 1. Memory (5dk) → 2. DB (7 gün) → 3. TMDB API
func (s *TMDBService) SearchMulti(ctx context.Context, query string, page int) (*models.MultiSearchResponse, error) {
	cacheKey := fmt.Sprintf("search:%s:%d", query, page)

	// 1. In-memory cache kontrol
	if cached, found := s.searchCache.Get(cacheKey); found {
		s.logger.Info("Arama sonucu memory cache'den geldi", "query", query)
		return cached.(*models.MultiSearchResponse), nil
	}

	// 2. DB'de taze veri var mı? (7 günden yeni)
	if s.contentRepo != nil {
		stale, err := s.contentRepo.IsStaleSearch(query)
		if err == nil && !stale {
			items, err := s.contentRepo.SearchInDB(query)
			if err == nil && len(items) > 0 {
				s.logger.Info("Arama sonucu DB'den geldi", "query", query, "count", len(items))
				result := &models.MultiSearchResponse{
					Page:         page,
					Results:      items,
					TotalResults: len(items),
					TotalPages:   1,
				}
				s.searchCache.Set(cacheKey, result)
				return result, nil
			}
		}
	}

	// 3. TMDB'den çek
	s.logger.Info("Arama sonucu TMDB'den çekiliyor", "query", query)
	params := url.Values{}
	params.Set("query", query)
	params.Set("page", fmt.Sprintf("%d", page))

	body, err := s.get(ctx, "/search/multi", params)
	if err != nil {
		return nil, err
	}

	var result models.MultiSearchResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("JSON parse hatası: %w", err)
	}

	// Sadece film ve dizi sonuçlarını filtrele (person'ları çıkar)
	filtered := make([]models.MultiSearchItem, 0, len(result.Results))
	for i := range result.Results {
		item := &result.Results[i]

		// Sadece movie ve tv tiplerini al
		if item.MediaType != "movie" && item.MediaType != "tv" {
			continue
		}

		// Poster URL'lerini tam hale getir
		if item.PosterPath != "" {
			item.PosterPath = s.imageBase + item.PosterPath
		}
		if item.BackdropPath != "" {
			item.BackdropPath = "https://image.tmdb.org/t/p/w1280" + item.BackdropPath
		}

		filtered = append(filtered, *item)
	}

	result.Results = filtered

	// Arka planda DB'ye kaydet ve logla
	if s.contentRepo != nil {
		go func() {
			if err := s.contentRepo.BulkUpsert(result.Results); err != nil {
				s.logger.Error("DB upsert hatası", "error", err)
			}
			if s.searchLogRepo != nil {
				s.searchLogRepo.Log(query, "multi", len(result.Results))
			}
		}()
	}

	// Memory cache'e kaydet
	s.searchCache.Set(cacheKey, &result)

	return &result, nil
}

// Film detayı — /movie/123
func (s *TMDBService) GetMovieDetail(ctx context.Context, id int) (*models.Movie, error) {
	cacheKey := fmt.Sprintf("movie:%d", id)

	if cached, found := s.detailCache.Get(cacheKey); found {
		return cached.(*models.Movie), nil
	}

	body, err := s.get(ctx, fmt.Sprintf("/movie/%d", id), url.Values{})
	if err != nil {
		return nil, err
	}

	var movie models.Movie
	if err := json.Unmarshal(body, &movie); err != nil {
		return nil, fmt.Errorf("JSON parse hatası: %w", err)
	}

	if movie.PosterPath != "" {
		movie.PosterPath = s.imageBase + movie.PosterPath
	}
	if movie.BackdropPath != "" {
		movie.BackdropPath = "https://image.tmdb.org/t/p/w1280" + movie.BackdropPath
	}

	s.detailCache.Set(cacheKey, &movie)

	return &movie, nil
}

// ==================== ORTAK YARDIMCI FONKSİYONLAR ====================

// getWatchProviders - Film veya dizi için izleme platformlarını getirir
// mediaType: "movie" veya "tv"
func (s *TMDBService) getWatchProviders(ctx context.Context, mediaType string, id int) (*models.WatchProviderResult, error) {
	body, err := s.get(ctx, fmt.Sprintf("/%s/%d/watch/providers", mediaType, id), url.Values{})
	if err != nil {
		return nil, err
	}

	var raw struct {
		Results map[string]*models.WatchProviderResult `json:"results"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("JSON parse hatası: %w", err)
	}

	// Sadece Türkiye verisini döndür
	tr := raw.Results["TR"]
	if tr == nil {
		return &models.WatchProviderResult{}, nil
	}

	// Platform logolarını tam URL yap
	for i := range tr.Flatrate {
		tr.Flatrate[i].LogoPath = s.imageBase + tr.Flatrate[i].LogoPath
	}

	return tr, nil
}

// getCredits - Film veya dizi için oyuncu/ekip bilgilerini getirir
// mediaType: "movie" veya "tv"
func (s *TMDBService) getCredits(ctx context.Context, mediaType string, id int) (*models.Credits, error) {
	body, err := s.get(ctx, fmt.Sprintf("/%s/%d/credits", mediaType, id), url.Values{})
	if err != nil {
		return nil, err
	}

	var credits models.Credits
	if err := json.Unmarshal(body, &credits); err != nil {
		return nil, fmt.Errorf("JSON parse hatası: %w", err)
	}

	// Profil fotoğraflarını tam URL yap
	for i := range credits.Cast {
		if credits.Cast[i].ProfilePath != "" {
			credits.Cast[i].ProfilePath = s.imageBase + credits.Cast[i].ProfilePath
		}
	}

	return &credits, nil
}

// ==================== FİLM FONKSİYONLARI ====================

// Film için izleme platformları — /movie/123/watch-providers
func (s *TMDBService) GetWatchProviders(ctx context.Context, id int) (*models.WatchProviderResult, error) {
	return s.getWatchProviders(ctx, "movie", id)
}

// Film oyuncuları — /movie/123/credits
func (s *TMDBService) GetCredits(ctx context.Context, id int) (*models.Credits, error) {
	return s.getCredits(ctx, "movie", id)
}

// ==================== TV SHOW (DİZİ) FONKSİYONLARI ====================

// Dizi detayı — /tv/123
func (s *TMDBService) GetTVDetail(ctx context.Context, id int) (*models.TVShow, error) {
	cacheKey := fmt.Sprintf("tv:%d", id)

	if cached, found := s.detailCache.Get(cacheKey); found {
		return cached.(*models.TVShow), nil
	}

	body, err := s.get(ctx, fmt.Sprintf("/tv/%d", id), url.Values{})
	if err != nil {
		return nil, err
	}

	var tv models.TVShow
	if err := json.Unmarshal(body, &tv); err != nil {
		return nil, fmt.Errorf("JSON parse hatası: %w", err)
	}

	if tv.PosterPath != "" {
		tv.PosterPath = s.imageBase + tv.PosterPath
	}
	if tv.BackdropPath != "" {
		tv.BackdropPath = "https://image.tmdb.org/t/p/w1280" + tv.BackdropPath
	}

	s.detailCache.Set(cacheKey, &tv)

	return &tv, nil
}

// Dizi için izleme platformları — /tv/123/watch-providers
func (s *TMDBService) GetTVWatchProviders(ctx context.Context, id int) (*models.WatchProviderResult, error) {
	return s.getWatchProviders(ctx, "tv", id)
}

// Dizi oyuncuları — /tv/123/credits
func (s *TMDBService) GetTVCredits(ctx context.Context, id int) (*models.Credits, error) {
	return s.getCredits(ctx, "tv", id)
}

// ==================== HEALTH CHECK ====================

// Ping TMDB API'sine bağlantıyı test eder
func (s *TMDBService) Ping(ctx context.Context) error {
	// Basit bir endpoint'e istek at
	_, err := s.get(ctx, "/configuration", url.Values{})
	return err
}

// GetCacheStats cache istatistiklerini döner
func (s *TMDBService) GetCacheStats() CacheStats {
	return CacheStats{
		Search: s.searchCache.GetStats(),
		Detail: s.detailCache.GetStats(),
	}
}
