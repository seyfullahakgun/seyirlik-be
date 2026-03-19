package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	TMDBApiKey string
	Port       string
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
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
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "postgres"),
		DBPassword: os.Getenv("DB_PASSWORD"),
		DBName:     getEnv("DB_NAME", "seyirlik"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
