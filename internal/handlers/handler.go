package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"seyirlik.net/api/internal/models"
	"seyirlik.net/api/internal/services"

	"github.com/labstack/echo/v4"
)

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
		return errorResponse(c, http.StatusInternalServerError, models.ErrCodeExternalAPI, err.Error())
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
		if errors.Is(err, services.ErrNotFound) {
			return errorResponse(c, http.StatusNotFound, models.ErrCodeNotFound, "Film bulunamadı")
		}
		return errorResponse(c, http.StatusInternalServerError, models.ErrCodeExternalAPI, err.Error())
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
		if errors.Is(err, services.ErrNotFound) {
			return errorResponse(c, http.StatusNotFound, models.ErrCodeNotFound, "Film bulunamadı")
		}
		return errorResponse(c, http.StatusInternalServerError, models.ErrCodeExternalAPI, err.Error())
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
		if errors.Is(err, services.ErrNotFound) {
			return errorResponse(c, http.StatusNotFound, models.ErrCodeNotFound, "Film bulunamadı")
		}
		return errorResponse(c, http.StatusInternalServerError, models.ErrCodeExternalAPI, err.Error())
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
		if errors.Is(err, services.ErrNotFound) {
			return errorResponse(c, http.StatusNotFound, models.ErrCodeNotFound, "Dizi bulunamadı")
		}
		return errorResponse(c, http.StatusInternalServerError, models.ErrCodeExternalAPI, err.Error())
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
		if errors.Is(err, services.ErrNotFound) {
			return errorResponse(c, http.StatusNotFound, models.ErrCodeNotFound, "Dizi bulunamadı")
		}
		return errorResponse(c, http.StatusInternalServerError, models.ErrCodeExternalAPI, err.Error())
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
		if errors.Is(err, services.ErrNotFound) {
			return errorResponse(c, http.StatusNotFound, models.ErrCodeNotFound, "Dizi bulunamadı")
		}
		return errorResponse(c, http.StatusInternalServerError, models.ErrCodeExternalAPI, err.Error())
	}

	return c.JSON(http.StatusOK, credits)
}
