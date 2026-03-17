package models

// TMDB'nin /search/movie endpoint'inden dönen yapı
type SearchResponse struct {
	Page         int     `json:"page"`
	Results      []Movie `json:"results"`
	TotalPages   int     `json:"total_pages"`
	TotalResults int     `json:"total_results"`
}

// Tek bir filmi temsil eder
type Movie struct {
	ID               int     `json:"id"`
	Title            string  `json:"title"`
	Overview         string  `json:"overview"`    // Film özeti
	PosterPath       string  `json:"poster_path"` // Afiş URL'inin son kısmı
	BackdropPath     string  `json:"backdrop_path"`
	ReleaseDate      string  `json:"release_date"`
	VoteAverage      float64 `json:"vote_average"` // IMDB benzeri puan
	VoteCount        int     `json:"vote_count"`
	GenreIDs         []int   `json:"genre_ids"` // Arama sonuçlarında gelir
	Genres           []Genre `json:"genres"`    // Detay sayfasında gelir
	Runtime          int     `json:"runtime"`   // Dakika cinsinden
	OriginalLanguage string  `json:"original_language"`
	OriginalTitle    string  `json:"original_title"`
	MediaType        string  `json:"media_type,omitempty"` // Multi search için: "movie"
}

// Tek bir diziyi temsil eder
type TVShow struct {
	ID               int     `json:"id"`
	Name             string  `json:"name"`
	Overview         string  `json:"overview"`
	PosterPath       string  `json:"poster_path"`
	BackdropPath     string  `json:"backdrop_path"`
	FirstAirDate     string  `json:"first_air_date"`
	VoteAverage      float64 `json:"vote_average"`
	VoteCount        int     `json:"vote_count"`
	GenreIDs         []int   `json:"genre_ids"`
	Genres           []Genre `json:"genres"`
	OriginalLanguage string  `json:"original_language"`
	OriginalName     string  `json:"original_name"`
	MediaType        string  `json:"media_type,omitempty"` // Multi search için: "tv"
}

// Multi search sonucu - film veya dizi olabilir
type MultiSearchItem struct {
	// Ortak alanlar
	ID           int     `json:"id"`
	MediaType    string  `json:"media_type"` // "movie" veya "tv"
	Overview     string  `json:"overview"`
	PosterPath   string  `json:"poster_path"`
	BackdropPath string  `json:"backdrop_path"`
	VoteAverage  float64 `json:"vote_average"`
	VoteCount    int     `json:"vote_count"`
	GenreIDs     []int   `json:"genre_ids"`

	// Film alanları (media_type="movie" ise dolu)
	Title         string `json:"title,omitempty"`
	OriginalTitle string `json:"original_title,omitempty"`
	ReleaseDate   string `json:"release_date,omitempty"`

	// Dizi alanları (media_type="tv" ise dolu)
	Name         string `json:"name,omitempty"`
	OriginalName string `json:"original_name,omitempty"`
	FirstAirDate string `json:"first_air_date,omitempty"`
}

// Multi search API response
type MultiSearchResponse struct {
	Page         int               `json:"page"`
	Results      []MultiSearchItem `json:"results"`
	TotalPages   int               `json:"total_pages"`
	TotalResults int               `json:"total_results"`
}

type Genre struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// Platform (Netflix, Prime vb.) bilgisi
type WatchProvider struct {
	LogoPath        string `json:"logo_path"`
	ProviderID      int    `json:"provider_id"`
	ProviderName    string `json:"provider_name"`
	DisplayPriority int    `json:"display_priority"`
}

// Türkiye için platform sonucu
type WatchProviderResult struct {
	Link     string          `json:"link"`     // Platforma direkt link
	Flatrate []WatchProvider `json:"flatrate"` // Abonelikle izlenebilir
	Rent     []WatchProvider `json:"rent"`     // Kiralık
	Buy      []WatchProvider `json:"buy"`      // Satın alma
}

// Oyuncu/yönetmen bilgisi
type Credits struct {
	Cast []CastMember `json:"cast"`
	Crew []CrewMember `json:"crew"`
}

type CastMember struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Character   string `json:"character"` // Oynadığı karakter
	ProfilePath string `json:"profile_path"`
	Order       int    `json:"order"` // Kaçıncı sırada gösterilsin
}

type CrewMember struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Job         string `json:"job"` // "Director", "Producer" vb.
	Department  string `json:"department"`
	ProfilePath string `json:"profile_path"`
}
