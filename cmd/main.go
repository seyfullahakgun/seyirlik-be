package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"seyirlik.net/api/internal/config"
	"seyirlik.net/api/internal/handlers"
	"seyirlik.net/api/internal/middleware"
	"seyirlik.net/api/internal/services"

	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"
)

func main() {
	// 1. Structured logger oluştur (JSON formatında)
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// 2. Ayarları yükle (.env'den TMDB_API_KEY ve PORT okur)
	cfg := config.Load()

	if cfg.TMDBApiKey == "" {
		logger.Error("TMDB_API_KEY env variable eksik!")
		os.Exit(1)
	}

	// 3. TMDB servisini oluştur
	tmdbService := services.NewTMDBService(cfg.TMDBApiKey)

	// 4. Handler'ı oluştur, servisi içine ver
	h := handlers.NewHandler(tmdbService)

	// 5. Echo sunucusunu başlat
	e := echo.New()
	e.HideBanner = true

	// ==================== MIDDLEWARE STACK ====================
	// Sıralama önemli: önce panic recovery, sonra request ID, sonra diğerleri

	// Panic recovery — sunucu çökmez, hata loglanır
	e.Use(echomw.RecoverWithConfig(echomw.RecoverConfig{
		LogErrorFunc: func(c echo.Context, err error, stack []byte) error {
			requestID := middleware.GetRequestID(c.Request().Context())
			logger.Error("Panic recovered",
				"request_id", requestID,
				"error", err.Error(),
				"stack", string(stack),
			)
			return nil
		},
	}))

	// Request ID — her isteğe unique ID atar
	e.Use(middleware.RequestID())

	// Security headers — X-Content-Type-Options, X-Frame-Options vb.
	e.Use(middleware.SecurityHeaders())

	// Gzip compression — response'ları sıkıştırır
	e.Use(echomw.GzipWithConfig(echomw.GzipConfig{
		Level: 5, // Dengeli sıkıştırma seviyesi
		Skipper: func(c echo.Context) bool {
			// Health check endpoint'ini sıkıştırma
			return c.Path() == "/health"
		},
	}))

	// Request timeout — 30 saniye sonra timeout
	e.Use(middleware.Timeout(30 * time.Second))

	// Rate limiting — dakikada 100 istek (IP başına)
	e.Use(echomw.RateLimiterWithConfig(echomw.RateLimiterConfig{
		Skipper: func(c echo.Context) bool {
			// Health check endpoint'ini rate limit'ten muaf tut
			return c.Path() == "/health"
		},
		Store: echomw.NewRateLimiterMemoryStoreWithConfig(
			echomw.RateLimiterMemoryStoreConfig{
				Rate:      100,             // Dakikada 100 istek
				Burst:     20,              // Ani 20 istek burst'üne izin ver
				ExpiresIn: 1 * time.Minute, // Rate limit window
			},
		),
		IdentifierExtractor: func(c echo.Context) (string, error) {
			return c.RealIP(), nil
		},
		ErrorHandler: func(c echo.Context, err error) error {
			return c.JSON(http.StatusTooManyRequests, map[string]string{
				"code":    "RATE_LIMIT_EXCEEDED",
				"message": "Çok fazla istek yapıldı, lütfen bekleyin",
			})
		},
		DenyHandler: func(c echo.Context, identifier string, err error) error {
			return c.JSON(http.StatusTooManyRequests, map[string]string{
				"code":    "RATE_LIMIT_EXCEEDED",
				"message": "Çok fazla istek yapıldı, lütfen bekleyin",
			})
		},
	}))

	// CORS — Frontend'den gelen isteklere izin ver
	e.Use(echomw.CORSWithConfig(echomw.CORSConfig{
		AllowOrigins: []string{
			"http://localhost:3000",
			"http://localhost:5173",
			"https://seyirlik.net",
			"https://www.seyirlik.net",
		},
		AllowMethods: []string{"GET"},
	}))

	// Request logger — her isteği loglar (en son, böylece response bilgisi de var)
	e.Use(middleware.SlogLogger(logger))

	// ==================== ROUTES ====================

	// Health check endpoint (detaylı)
	e.GET("/health", h.HealthCheck)

	// API routes
	api := e.Group("/api")
	api.GET("/search", h.Search) // Multi search: film + dizi

	// Film endpoint'leri
	api.GET("/movie/:id", h.GetDetail)
	api.GET("/movie/:id/watch-providers", h.GetWatchProviders)
	api.GET("/movie/:id/credits", h.GetCredits)

	// Dizi endpoint'leri
	api.GET("/tv/:id", h.GetTVDetail)
	api.GET("/tv/:id/watch-providers", h.GetTVWatchProviders)
	api.GET("/tv/:id/credits", h.GetTVCredits)

	// ==================== SERVER START ====================

	logger.Info("Sunucu başlatılıyor", "port", cfg.Port)
	go func() {
		if err := e.Start(":" + cfg.Port); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("Sunucu hatası", "error", err.Error())
			os.Exit(1)
		}
	}()

	// Shutdown sinyallerini dinle
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Sunucu kapatılıyor...")

	// 10 saniye içinde mevcut istekleri tamamla
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := e.Shutdown(ctx); err != nil {
		logger.Error("Sunucu kapatma hatası", "error", err.Error())
		os.Exit(1)
	}

	logger.Info("Sunucu başarıyla kapatıldı")
}
