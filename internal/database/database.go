package database

import (
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver'ı register eder
)

type DB struct {
	*sql.DB // sql.DB'yi embed ediyoruz, tüm metodlarına direkt erişebiliriz
}

type Config struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
}

// Yeni DB bağlantısı oluşturur
func New(cfg Config) (*DB, error) {
	// PostgreSQL bağlantı string'i
	// örnek: "host=localhost port=5432 user=postgres password=123 dbname=seyirlik sslmode=disable"
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Name,
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("DB bağlantısı açılamadı: %w", err)
	}

	// Bağlantı havuzu ayarları
	// Aynı anda max 25 bağlantı açık olabilir
	db.SetMaxOpenConns(25)
	// Bunların max 25'i boşta bekleyebilir
	db.SetMaxIdleConns(25)
	// Bağlantılar max 5 dakika açık kalabilir
	db.SetConnMaxLifetime(5 * time.Minute)

	// Gerçekten bağlanabildi mi test et
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("DB'ye ping atılamadı: %w", err)
	}

	slog.Info("PostgreSQL bağlantısı kuruldu", "host", cfg.Host, "db", cfg.Name)

	return &DB{db}, nil
}

// Tabloları oluşturur — uygulama her başladığında çalışır
// Tablo zaten varsa "IF NOT EXISTS" sayesinde hata vermez
func (db *DB) Migrate() error {
	queries := []string{
		// İçerikler tablosu — film ve diziler burada tutulur
		`CREATE TABLE IF NOT EXISTS contents (
			id            SERIAL PRIMARY KEY,
			tmdb_id       INTEGER NOT NULL,
			media_type    VARCHAR(10) NOT NULL CHECK (media_type IN ('movie', 'tv')),
			title         VARCHAR(500) NOT NULL,
			original_title VARCHAR(500),
			overview      TEXT,
			poster_path   VARCHAR(500),
			backdrop_path VARCHAR(500),
			release_date  VARCHAR(20),
			vote_average  DECIMAL(4,2) DEFAULT 0,
			vote_count    INTEGER DEFAULT 0,
			popularity    DECIMAL(10,4) DEFAULT 0,
			genres        JSONB DEFAULT '[]',
			runtime       INTEGER DEFAULT 0,
			updated_at    TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			created_at    TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			UNIQUE(tmdb_id, media_type)
		)`,

		// Arama logları tablosu — kim ne aradı
		`CREATE TABLE IF NOT EXISTS search_logs (
			id            SERIAL PRIMARY KEY,
			query         VARCHAR(500) NOT NULL,
			media_type    VARCHAR(10),
			results_count INTEGER DEFAULT 0,
			created_at    TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,

		// Hızlı arama için index
		`CREATE INDEX IF NOT EXISTS idx_contents_tmdb_id ON contents(tmdb_id)`,
		`CREATE INDEX IF NOT EXISTS idx_contents_media_type ON contents(media_type)`,
		`CREATE INDEX IF NOT EXISTS idx_contents_updated_at ON contents(updated_at)`,
		`CREATE INDEX IF NOT EXISTS idx_search_logs_created_at ON search_logs(created_at)`,
	}

	for _, q := range queries {
		if _, err := db.Exec(q); err != nil {
			return fmt.Errorf("migration hatası: %w", err)
		}
	}

	slog.Info("DB migration tamamlandı")
	return nil
}
