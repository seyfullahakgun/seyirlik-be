package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"seyirlik.net/api/internal/config"
	"seyirlik.net/api/internal/handlers"
	"seyirlik.net/api/internal/services"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	// 1. Ayarları yükle (.env'den TMDB_API_KEY ve PORT okur)
	cfg := config.Load()

	if cfg.TMDBApiKey == "" {
		log.Fatal("TMDB_API_KEY env variable eksik!")
	}

	// 2. TMDB servisini oluştur
	tmdbService := services.NewTMDBService(cfg.TMDBApiKey)

	// 3. Handler'ı oluştur, servisi içine ver
	h := handlers.NewHandler(tmdbService)

	// 4. Echo sunucusunu başlat
	e := echo.New()

	// Loglama — her isteği terminale yazar
	e.Use(middleware.Logger())
	// Panic olursa sunucu çökmez, hata döner
	e.Use(middleware.Recover())
	// Frontend'den gelen isteklere izin ver (CORS)
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"http://localhost:3000", "http://localhost:5173", "https://seyirlik.net", "https://www.seyirlik.net"},
		AllowMethods: []string{"GET"},
	}))

	// 5. Health check endpoint
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{
			"status": "ok",
		})
	})

	// 6. Route'ları tanımla
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

	// 7. Sunucuyu başlat (goroutine içinde)
	log.Printf("Sunucu :%s portunda başlatılıyor...", cfg.Port)
	go func() {
		if err := e.Start(":" + cfg.Port); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Sunucu hatası: %v", err)
		}
	}()

	// Shutdown sinyallerini dinle
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Sunucu kapatılıyor...")

	// 10 saniye içinde mevcut istekleri tamamla
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := e.Shutdown(ctx); err != nil {
		log.Fatalf("Sunucu kapatma hatası: %v", err)
	}

	log.Println("Sunucu başarıyla kapatıldı")
}
