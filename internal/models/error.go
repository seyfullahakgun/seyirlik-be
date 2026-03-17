package models

// API hata yanıtı için standart yapı
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Hata kodları
const (
	ErrCodeBadRequest     = "BAD_REQUEST"
	ErrCodeNotFound       = "NOT_FOUND"
	ErrCodeInternalError  = "INTERNAL_ERROR"
	ErrCodeValidation     = "VALIDATION_ERROR"
	ErrCodeExternalAPI    = "EXTERNAL_API_ERROR"
)
