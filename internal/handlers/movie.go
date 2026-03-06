package handlers

import (
	"net/http"
	"strconv"

	"seyirlik.net/api/internal/services"

	"github.com/labstack/echo/v4"
)

type MovieHandler struct {
	tmdb *services.TMDBService
}

// Handler'ı oluşturur, servisi içine enjekte eder
func NewMovieHandler(tmdb *services.TMDBService) *MovieHandler {
	return &MovieHandler{tmdb: tmdb}
}

// GET /api/search?q=inception&page=1
func (h *MovieHandler) Search(c echo.Context) error {
	query := c.QueryParam("q")
	if query == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "q parametresi zorunlu",
		})
	}

	// page parametresi yoksa 1 kullan
	page := 1
	if p := c.QueryParam("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil {
			page = parsed
		}
	}

	result, err := h.tmdb.SearchMovies(query, page)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, result)
}

// GET /api/movie/:id
func (h *MovieHandler) GetDetail(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Geçersiz film ID",
		})
	}

	movie, err := h.tmdb.GetMovieDetail(id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, movie)
}

// GET /api/movie/:id/watch-providers
func (h *MovieHandler) GetWatchProviders(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Geçersiz film ID",
		})
	}

	providers, err := h.tmdb.GetWatchProviders(id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, providers)
}

// GET /api/movie/:id/credits
func (h *MovieHandler) GetCredits(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Geçersiz film ID",
		})
	}

	credits, err := h.tmdb.GetCredits(id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, credits)
}
