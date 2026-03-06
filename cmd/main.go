package main

import (
	"log"

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
	movieHandler := handlers.NewMovieHandler(tmdbService)

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

	// 5. Route'ları tanımla
	api := e.Group("/api")
	api.GET("/search", movieHandler.Search)
	api.GET("/movie/:id", movieHandler.GetDetail)
	api.GET("/movie/:id/watch-providers", movieHandler.GetWatchProviders)
	api.GET("/movie/:id/credits", movieHandler.GetCredits)

	// 6. Sunucuyu başlat
	log.Printf("Sunucu :%s portunda çalışıyor...", cfg.Port)
	if err := e.Start(":" + cfg.Port); err != nil {
		log.Fatal(err)
	}
}
