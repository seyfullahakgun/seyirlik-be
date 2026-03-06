package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	TMDBApiKey string
	Port       string
}

func Load() *Config {
	// .env dosyasını yükle
	// Hata verirse zaten env variable set edilmiş demektir (production'da normal)
	if err := godotenv.Load(); err != nil {
		log.Println(".env dosyası bulunamadı, sistem env variable'ları kullanılıyor")
	}

	return &Config{
		TMDBApiKey: os.Getenv("TMDB_API_KEY"),
		Port:       getEnv("PORT", "8080"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
