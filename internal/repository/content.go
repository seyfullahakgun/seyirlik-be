package repository

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"seyirlik.net/api/internal/database"
	"seyirlik.net/api/internal/models"
)

// ContentRepository — contents tablosu ile konuşur
type ContentRepository struct {
	db *database.DB
}

func NewContentRepository(db *database.DB) *ContentRepository {
	return &ContentRepository{db: db}
}

// CacheEntry — DB'den okunan içerik + cache bilgisi
type CacheEntry struct {
	Content   models.MultiSearchItem
	UpdatedAt time.Time
}

// IsStale — 7 günden eskiyse true döner, yenilenmesi gerekir
func (c *CacheEntry) IsStale() bool {
	return time.Since(c.UpdatedAt) > 7*24*time.Hour
}

// GetByTMDBID — tmdb_id ve media_type ile içerik arar
// Bulunamazsa sql.ErrNoRows hatası döner
func (r *ContentRepository) GetByTMDBID(tmdbID int, mediaType string) (*CacheEntry, error) {
	query := `
		SELECT tmdb_id, media_type, title, original_title, overview,
		       poster_path, backdrop_path, release_date,
		       vote_average, vote_count, genres, updated_at
		FROM contents
		WHERE tmdb_id = $1 AND media_type = $2
	`

	var entry CacheEntry
	var genresJSON []byte
	var posterPath, backdropPath, originalTitle, overview, releaseDate sql.NullString

	err := r.db.QueryRow(query, tmdbID, mediaType).Scan(
		&entry.Content.ID,
		&entry.Content.MediaType,
		&entry.Content.Title,
		&originalTitle,
		&overview,
		&posterPath,
		&backdropPath,
		&releaseDate,
		&entry.Content.VoteAverage,
		&entry.Content.VoteCount,
		&genresJSON,
		&entry.UpdatedAt,
	)

	if err != nil {
		return nil, err // sql.ErrNoRows dahil
	}

	// NULL olabilecek alanları güvenle ata
	if originalTitle.Valid {
		entry.Content.OriginalTitle = originalTitle.String
	}
	if overview.Valid {
		entry.Content.Overview = overview.String
	}
	if posterPath.Valid {
		entry.Content.PosterPath = posterPath.String
	}
	if backdropPath.Valid {
		entry.Content.BackdropPath = backdropPath.String
	}
	if releaseDate.Valid {
		entry.Content.ReleaseDate = releaseDate.String
	}

	// JSONB olarak saklanan genre'leri parse et
	if len(genresJSON) > 0 {
		json.Unmarshal(genresJSON, &entry.Content.GenreIDs)
	}

	return &entry, nil
}

// Upsert — içerik varsa güncelle, yoksa ekle
// "INSERT ... ON CONFLICT DO UPDATE" — PostgreSQL'e özgü güçlü bir özellik
func (r *ContentRepository) Upsert(item models.MultiSearchItem) error {
	genresJSON, err := json.Marshal(item.GenreIDs)
	if err != nil {
		genresJSON = []byte("[]")
	}

	// Başlık: film için Title, dizi için Name
	title := item.Title
	if title == "" {
		title = item.Name
	}

	originalTitle := item.OriginalTitle
	if originalTitle == "" {
		originalTitle = item.OriginalName
	}

	releaseDate := item.ReleaseDate
	if releaseDate == "" {
		releaseDate = item.FirstAirDate
	}

	query := `
		INSERT INTO contents (
			tmdb_id, media_type, title, original_title, overview,
			poster_path, backdrop_path, release_date,
			vote_average, vote_count, genres, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,NOW())
		ON CONFLICT (tmdb_id, media_type) DO UPDATE SET
			title          = EXCLUDED.title,
			original_title = EXCLUDED.original_title,
			overview       = EXCLUDED.overview,
			poster_path    = EXCLUDED.poster_path,
			backdrop_path  = EXCLUDED.backdrop_path,
			release_date   = EXCLUDED.release_date,
			vote_average   = EXCLUDED.vote_average,
			vote_count     = EXCLUDED.vote_count,
			genres         = EXCLUDED.genres,
			updated_at     = NOW()
	`

	_, err = r.db.Exec(query,
		item.ID, item.MediaType, title, originalTitle, item.Overview,
		item.PosterPath, item.BackdropPath, releaseDate,
		item.VoteAverage, item.VoteCount, genresJSON,
	)

	return err
}

// BulkUpsert — arama sonuçlarını toplu kaydet
func (r *ContentRepository) BulkUpsert(items []models.MultiSearchItem) error {
	for _, item := range items {
		// person gibi media_type'ları atla
		if item.MediaType != "movie" && item.MediaType != "tv" {
			continue
		}
		if err := r.Upsert(item); err != nil {
			return fmt.Errorf("upsert hatası (id:%d): %w", item.ID, err)
		}
	}
	return nil
}

// SearchInDB — DB'de query'e göre içerik arar
// Önce title'da arar, bulamazsa original_title'da
func (r *ContentRepository) SearchInDB(query string) ([]models.MultiSearchItem, error) {
	sqlQuery := `
		SELECT tmdb_id, media_type, title, original_title, overview,
		       poster_path, backdrop_path, release_date,
		       vote_average, vote_count, genres, updated_at
		FROM contents
		WHERE title ILIKE $1 OR original_title ILIKE $1
		ORDER BY vote_count DESC
		LIMIT 20
	`
	// %query% → "inception" → "%inception%" şeklinde arar
	rows, err := r.db.Query(sqlQuery, "%"+query+"%")
	if err != nil {
		return nil, fmt.Errorf("DB arama hatası: %w", err)
	}
	defer rows.Close()

	var items []models.MultiSearchItem
	var oldestUpdatedAt time.Time

	for rows.Next() {
		var item models.MultiSearchItem
		var genresJSON []byte
		var posterPath, backdropPath, originalTitle, overview, releaseDate sql.NullString
		var updatedAt time.Time

		err := rows.Scan(
			&item.ID,
			&item.MediaType,
			&item.Title,
			&originalTitle,
			&overview,
			&posterPath,
			&backdropPath,
			&releaseDate,
			&item.VoteAverage,
			&item.VoteCount,
			&genresJSON,
			&updatedAt,
		)
		if err != nil {
			continue
		}

		if originalTitle.Valid {
			item.OriginalTitle = originalTitle.String
		}
		if overview.Valid {
			item.Overview = overview.String
		}
		if posterPath.Valid {
			item.PosterPath = posterPath.String
		}
		if backdropPath.Valid {
			item.BackdropPath = backdropPath.String
		}
		if releaseDate.Valid {
			item.ReleaseDate = releaseDate.String
		}
		if len(genresJSON) > 0 {
			json.Unmarshal(genresJSON, &item.GenreIDs)
		}

		// En eski updated_at'ı takip et
		if oldestUpdatedAt.IsZero() || updatedAt.Before(oldestUpdatedAt) {
			oldestUpdatedAt = updatedAt
		}

		items = append(items, item)
	}

	return items, nil
}

// IsStaleSearch — DB'deki arama sonuçları 7 günden eski mi?
func (r *ContentRepository) IsStaleSearch(query string) (bool, error) {
	sqlQuery := `
		SELECT MIN(updated_at)
		FROM contents
		WHERE title ILIKE $1 OR original_title ILIKE $1
	`
	var oldestUpdatedAt sql.NullTime
	err := r.db.QueryRow(sqlQuery, "%"+query+"%").Scan(&oldestUpdatedAt)
	if err != nil {
		return true, err
	}

	// Hiç kayıt yoksa stale sayılır
	if !oldestUpdatedAt.Valid {
		return true, nil
	}

	// 7 günden eskiyse stale
	return time.Since(oldestUpdatedAt.Time) > 7*24*time.Hour, nil
}
