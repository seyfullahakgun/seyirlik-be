# Aşama 1: Uygulamayı Derle
FROM golang:1.24-alpine AS builder

# Build için gerekli araçlar
RUN apk add --no-cache ca-certificates

WORKDIR /app

# Önce dependency'leri indir (cache için)
COPY go.mod go.sum ./
RUN go mod download

# Kaynak kodu kopyala ve derle
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /seyirlik_be ./cmd/main.go

# Aşama 2: Minimal Çalıştırma Ortamı
FROM alpine:3.19

# HTTPS istekleri için CA sertifikaları
RUN apk --no-cache add ca-certificates

# Güvenlik: non-root user
RUN adduser -D -g '' appuser

WORKDIR /app

# Binary'yi kopyala
COPY --from=builder /seyirlik_be /app/seyirlik_be

# Non-root user olarak çalıştır
USER appuser

# Port tanımla
EXPOSE 8080

# Uygulamayı başlat
CMD ["/app/seyirlik_be"]
