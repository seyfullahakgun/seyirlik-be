package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"seyirlik.net/api/internal/models"
	"seyirlik.net/api/internal/services"

	"github.com/labstack/echo/v4"
)

// Uygulama başlangıç zamanı (uptime için)
var startTime = time.Now()

// Sayfa limitleri
const (
	minPage = 1
	maxPage = 500 // TMDB limiti
)

type Handler struct {
	tmdb services.TMDBClient
}

// Handler'ı oluşturur, servisi içine enjekte eder
func NewHandler(tmdb services.TMDBClient) *Handler {
	return &Handler{tmdb: tmdb}
}

// Standart hata yanıtı döner
func errorResponse(c echo.Context, status int, code, message string) error {
	return c.JSON(status, models.APIError{
		Code:    code,
		Message: message,
	})
}

// handleTMDBError TMDB servis hatalarını HTTP yanıtına çevirir
func handleTMDBError(c echo.Context, err error, notFoundMsg string) error {
	switch {
	case errors.Is(err, services.ErrNotFound):
		return errorResponse(c, http.StatusNotFound, models.ErrCodeNotFound, notFoundMsg)
	case errors.Is(err, services.ErrRateLimit):
		return errorResponse(c, http.StatusTooManyRequests, models.ErrCodeRateLimit, "Çok fazla istek yapıldı, lütfen bekleyin")
	case errors.Is(err, services.ErrServerError):
		return errorResponse(c, http.StatusBadGateway, models.ErrCodeExternalAPI, "TMDB servisi şu anda kullanılamıyor")
	case errors.Is(err, services.ErrBadRequest):
		return errorResponse(c, http.StatusBadRequest, models.ErrCodeBadRequest, "Geçersiz istek")
	default:
		return errorResponse(c, http.StatusInternalServerError, models.ErrCodeExternalAPI, "Beklenmeyen bir hata oluştu")
	}
}

// ID parametresini parse eder ve validate eder
func parseID(c echo.Context) (int, error) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id < 1 {
		return 0, errors.New("geçersiz ID")
	}
	return id, nil
}

// Page parametresini parse eder ve validate eder
func parsePage(c echo.Context) int {
	page := 1
	if p := c.QueryParam("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil {
			page = parsed
		}
	}
	// Sınırları kontrol et
	if page < minPage {
		page = minPage
	}
	if page > maxPage {
		page = maxPage
	}
	return page
}

// GET /api/search?q=inception&page=1
// Multi search: Hem film hem dizi döner, media_type ile ayırt edilir
func (h *Handler) Search(c echo.Context) error {
	query := c.QueryParam("q")
	if query == "" {
		return errorResponse(c, http.StatusBadRequest, models.ErrCodeValidation, "q parametresi zorunlu")
	}

	page := parsePage(c)

	result, err := h.tmdb.SearchMulti(c.Request().Context(), query, page)
	if err != nil {
		return handleTMDBError(c, err, "Arama sonucu bulunamadı")
	}

	return c.JSON(http.StatusOK, result)
}

// GET /api/movie/:id
func (h *Handler) GetDetail(c echo.Context) error {
	id, err := parseID(c)
	if err != nil {
		return errorResponse(c, http.StatusBadRequest, models.ErrCodeValidation, "Geçersiz film ID")
	}

	movie, err := h.tmdb.GetMovieDetail(c.Request().Context(), id)
	if err != nil {
		return handleTMDBError(c, err, "Film bulunamadı")
	}

	return c.JSON(http.StatusOK, movie)
}

// GET /api/movie/:id/watch-providers
func (h *Handler) GetWatchProviders(c echo.Context) error {
	id, err := parseID(c)
	if err != nil {
		return errorResponse(c, http.StatusBadRequest, models.ErrCodeValidation, "Geçersiz film ID")
	}

	providers, err := h.tmdb.GetWatchProviders(c.Request().Context(), id)
	if err != nil {
		return handleTMDBError(c, err, "Film bulunamadı")
	}

	return c.JSON(http.StatusOK, providers)
}

// GET /api/movie/:id/credits
func (h *Handler) GetCredits(c echo.Context) error {
	id, err := parseID(c)
	if err != nil {
		return errorResponse(c, http.StatusBadRequest, models.ErrCodeValidation, "Geçersiz film ID")
	}

	credits, err := h.tmdb.GetCredits(c.Request().Context(), id)
	if err != nil {
		return handleTMDBError(c, err, "Film bulunamadı")
	}

	return c.JSON(http.StatusOK, credits)
}

// ==================== TV SHOW (DİZİ) HANDLER'LARI ====================

// GET /api/tv/:id
func (h *Handler) GetTVDetail(c echo.Context) error {
	id, err := parseID(c)
	if err != nil {
		return errorResponse(c, http.StatusBadRequest, models.ErrCodeValidation, "Geçersiz dizi ID")
	}

	tv, err := h.tmdb.GetTVDetail(c.Request().Context(), id)
	if err != nil {
		return handleTMDBError(c, err, "Dizi bulunamadı")
	}

	return c.JSON(http.StatusOK, tv)
}

// GET /api/tv/:id/watch-providers
func (h *Handler) GetTVWatchProviders(c echo.Context) error {
	id, err := parseID(c)
	if err != nil {
		return errorResponse(c, http.StatusBadRequest, models.ErrCodeValidation, "Geçersiz dizi ID")
	}

	providers, err := h.tmdb.GetTVWatchProviders(c.Request().Context(), id)
	if err != nil {
		return handleTMDBError(c, err, "Dizi bulunamadı")
	}

	return c.JSON(http.StatusOK, providers)
}

// GET /api/tv/:id/credits
func (h *Handler) GetTVCredits(c echo.Context) error {
	id, err := parseID(c)
	if err != nil {
		return errorResponse(c, http.StatusBadRequest, models.ErrCodeValidation, "Geçersiz dizi ID")
	}

	credits, err := h.tmdb.GetTVCredits(c.Request().Context(), id)
	if err != nil {
		return handleTMDBError(c, err, "Dizi bulunamadı")
	}

	return c.JSON(http.StatusOK, credits)
}

// ==================== HEALTH CHECK ====================

// HealthResponse health check yanıtı
type HealthResponse struct {
	Status    string              `json:"status"`
	Uptime    string              `json:"uptime"`
	TMDB      string              `json:"tmdb"`
	Cache     services.CacheStats `json:"cache"`
	Timestamp string              `json:"timestamp"`
}

// HealthCheck detaylı sağlık kontrolü
func (h *Handler) HealthCheck(c echo.Context) error {
	// TMDB bağlantısını kontrol et
	tmdbStatus := "ok"
	if err := h.tmdb.Ping(c.Request().Context()); err != nil {
		tmdbStatus = "error"
	}

	// Cache istatistikleri
	cacheStats := h.tmdb.GetCacheStats()

	// Uptime hesapla
	uptime := time.Since(startTime).Round(time.Second).String()

	response := HealthResponse{
		Status:    "ok",
		Uptime:    uptime,
		TMDB:      tmdbStatus,
		Cache:     cacheStats,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	// TMDB down ise status'u degraded yap
	if tmdbStatus != "ok" {
		response.Status = "degraded"
	}

	return c.JSON(http.StatusOK, response)
}
