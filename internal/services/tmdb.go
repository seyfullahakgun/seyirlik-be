package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"seyirlik.net/api/internal/models"
)

type TMDBService struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
	imageBase  string
}

// Yeni bir TMDBService oluşturur
func NewTMDBService(apiKey string) *TMDBService {
	return &TMDBService{
		apiKey:     apiKey,
		baseURL:    "https://api.themoviedb.org/3",
		httpClient: &http.Client{},
		imageBase:  "https://image.tmdb.org/t/p/w500",
	}
}

// TMDB'ye GET isteği atan yardımcı fonksiyon
// Her endpoint için tekrar tekrar aynı kodu yazmamak için
func (s *TMDBService) get(endpoint string, params url.Values) ([]byte, error) {
	// API key'i her isteğe otomatik ekle
	params.Set("api_key", s.apiKey)
	params.Set("language", "tr-TR") // Türkçe içerik

	fullURL := fmt.Sprintf("%s%s?%s", s.baseURL, endpoint, params.Encode())

	resp, err := s.httpClient.Get(fullURL)
	if err != nil {
		return nil, fmt.Errorf("istek hatası: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("TMDB hata kodu: %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

// Film arama — /search?q=inception
func (s *TMDBService) SearchMovies(query string, page int) (*models.SearchResponse, error) {
	params := url.Values{}
	params.Set("query", query)
	params.Set("page", fmt.Sprintf("%d", page))

	body, err := s.get("/search/movie", params)
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

// Film detayı — /movie/123
func (s *TMDBService) GetMovieDetail(id int) (*models.Movie, error) {
	body, err := s.get(fmt.Sprintf("/movie/%d", id), url.Values{})
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

	return &movie, nil
}

// Hangi platformda var — /movie/123/watch-providers
func (s *TMDBService) GetWatchProviders(id int) (*models.WatchProviderResult, error) {
	body, err := s.get(fmt.Sprintf("/movie/%d/watch/providers", id), url.Values{})
	if err != nil {
		return nil, err
	}

	// TMDB ülke bazlı döner: { results: { TR: { flatrate: [...] } } }
	var raw struct {
		Results map[string]*models.WatchProviderResult `json:"results"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("JSON parse hatası: %w", err)
	}

	// Sadece Türkiye verisini döndür
	tr := raw.Results["TR"]
	if tr == nil {
		return &models.WatchProviderResult{}, nil // TR'de yoksa boş döner
	}

	// Platform logolarını tam URL yap
	for i := range tr.Flatrate {
		tr.Flatrate[i].LogoPath = s.imageBase + tr.Flatrate[i].LogoPath
	}

	return tr, nil
}

// Oyuncular ve yönetmen — /movie/123/credits
func (s *TMDBService) GetCredits(id int) (*models.Credits, error) {
	body, err := s.get(fmt.Sprintf("/movie/%d/credits", id), url.Values{})
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
