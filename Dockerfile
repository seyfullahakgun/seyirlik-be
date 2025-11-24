# Aşama 1: Uygulamayı Derle
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod .
COPY go.sum .
RUN go mod tidy
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /seyirlik_be ./cmd/main.go 

# Aşama 2: Çalıştırma Ortamı
FROM alpine:latest
COPY --from=builder /seyirlik_be /usr/local/bin/
CMD ["/usr/local/bin/seyirlik_be"]
