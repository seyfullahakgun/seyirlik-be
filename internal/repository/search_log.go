package repository

import (
	"seyirlik.net/api/internal/database"
)

type SearchLogRepository struct {
	db *database.DB
}

func NewSearchLogRepository(db *database.DB) *SearchLogRepository {
	return &SearchLogRepository{db: db}
}

// Log — arama kaydı oluşturur, hata olsa bile sessizce geçer
// Loglama kritik değil, ana akışı engellememelidir
func (r *SearchLogRepository) Log(query, mediaType string, resultsCount int) {
	sql := `
		INSERT INTO search_logs (query, media_type, results_count)
		VALUES ($1, $2, $3)
	`
	// Hata olsa bile görmezden gel — loglama başarısız olursa kullanıcı etkilenmemeli
	r.db.Exec(sql, query, mediaType, resultsCount)
}
