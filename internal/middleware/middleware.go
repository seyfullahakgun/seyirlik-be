package middleware

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// Context key'leri
type contextKey string

const (
	RequestIDKey contextKey = "request_id"
)

// RequestID her request'e unique ID atar
func RequestID() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Header'da varsa kullan, yoksa yeni oluştur
			requestID := c.Request().Header.Get("X-Request-ID")
			if requestID == "" {
				requestID = uuid.New().String()
			}

			// Response header'a ekle
			c.Response().Header().Set("X-Request-ID", requestID)

			// Context'e ekle
			ctx := context.WithValue(c.Request().Context(), RequestIDKey, requestID)
			c.SetRequest(c.Request().WithContext(ctx))

			return next(c)
		}
	}
}

// GetRequestID context'ten request ID'yi alır
func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(RequestIDKey).(string); ok {
		return id
	}
	return ""
}

// SlogLogger slog tabanlı request logger
func SlogLogger(logger *slog.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			// Request'i işle
			err := next(c)

			// Log bilgilerini topla
			req := c.Request()
			res := c.Response()
			latency := time.Since(start)

			// Request ID'yi al
			requestID := GetRequestID(req.Context())

			// Log attributes
			attrs := []any{
				"request_id", requestID,
				"method", req.Method,
				"path", req.URL.Path,
				"status", res.Status,
				"latency_ms", latency.Milliseconds(),
				"ip", c.RealIP(),
				"user_agent", req.UserAgent(),
			}

			// Query params varsa ekle
			if req.URL.RawQuery != "" {
				attrs = append(attrs, "query", req.URL.RawQuery)
			}

			// Hata varsa ekle
			if err != nil {
				attrs = append(attrs, "error", err.Error())
				logger.Error("HTTP request failed", attrs...)
				return err
			}

			// Status'a göre log level
			if res.Status >= 500 {
				logger.Error("HTTP request", attrs...)
			} else if res.Status >= 400 {
				logger.Warn("HTTP request", attrs...)
			} else {
				logger.Info("HTTP request", attrs...)
			}

			return nil
		}
	}
}

// Timeout request'lere timeout ekler
func Timeout(timeout time.Duration) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ctx, cancel := context.WithTimeout(c.Request().Context(), timeout)
			defer cancel()

			c.SetRequest(c.Request().WithContext(ctx))

			// Channel ile timeout kontrolü
			done := make(chan error, 1)
			go func() {
				done <- next(c)
			}()

			select {
			case err := <-done:
				return err
			case <-ctx.Done():
				return echo.ErrRequestTimeout
			}
		}
	}
}

// SecurityHeaders güvenlik header'larını ekler
func SecurityHeaders() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Güvenlik header'ları
			c.Response().Header().Set("X-Content-Type-Options", "nosniff")
			c.Response().Header().Set("X-Frame-Options", "DENY")
			c.Response().Header().Set("X-XSS-Protection", "1; mode=block")
			c.Response().Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

			return next(c)
		}
	}
}
