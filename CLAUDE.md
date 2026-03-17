# Seyirlik Backend

Film ve dizi izleme platformu Seyirlik'in Go backend API'si. TMDB API'sini sarmalayarak Turkce icerik sunar.

## Tech Stack

- **Dil:** Go 1.24
- **Framework:** Echo v4
- **Harici API:** TMDB (The Movie Database)
- **Cache:** In-memory (TTL bazli)
- **Deploy:** Docker + GitHub Actions -> Hetzner

## Proje Yapisi

```
seyirlik-be/
├── cmd/
│   └── main.go                 # Uygulama giris noktasi, graceful shutdown
├── internal/
│   ├── cache/
│   │   └── cache.go            # Thread-safe in-memory cache
│   ├── config/
│   │   └── config.go           # Environment degiskenleri
│   ├── handlers/
│   │   └── handler.go          # HTTP handler'lari
│   ├── models/
│   │   ├── movie.go            # Film, Dizi, Credits veri yapilari
│   │   └── error.go            # Standart API hata yapisi
│   └── services/
│       └── tmdb.go             # TMDB API istemcisi (interface + impl)
├── .github/workflows/
│   └── deploy.yml              # CI/CD pipeline
├── .gitignore                  # Git ignore kurallari
├── Dockerfile                  # Multi-stage, non-root user
└── go.mod
```

## API Endpoint'leri

Tum endpoint'ler `/api` prefix'i altinda:

### Arama
| Endpoint | Aciklama |
|----------|----------|
| `GET /api/search?q={query}&page={n}` | Film ve dizi arama (media_type ile ayirt edilir) |

### Film
| Endpoint | Aciklama |
|----------|----------|
| `GET /api/movie/:id` | Film detayi |
| `GET /api/movie/:id/watch-providers` | Turkiye'deki izleme platformlari |
| `GET /api/movie/:id/credits` | Oyuncu ve ekip bilgileri |

### Dizi
| Endpoint | Aciklama |
|----------|----------|
| `GET /api/tv/:id` | Dizi detayi |
| `GET /api/tv/:id/watch-providers` | Turkiye'deki izleme platformlari |
| `GET /api/tv/:id/credits` | Oyuncu ve ekip bilgileri |

### Health Check
| Endpoint | Aciklama |
|----------|----------|
| `GET /health` | Sunucu saglik kontrolu |

## Hata Yanit Formati

```json
{
  "code": "NOT_FOUND",
  "message": "Film bulunamadi"
}
```

Hata kodlari: `BAD_REQUEST`, `NOT_FOUND`, `VALIDATION_ERROR`, `EXTERNAL_API_ERROR`, `INTERNAL_ERROR`

## Environment Variables

```bash
TMDB_API_KEY=xxx    # Zorunlu - TMDB API anahtari
PORT=8080           # Opsiyonel - Varsayilan 8080
```

## Gelistirme Komutlari

```bash
# Calistir
go run cmd/main.go

# Build
go build -o seyirlik_be ./cmd/main.go

# Docker
docker build -t seyirlik-be .
docker run -p 8080:8080 -e TMDB_API_KEY=xxx seyirlik-be
```

## Mimari Notlar

### Onemli Ozellikler
- **Interface-based Design:** `TMDBClient` interface'i ile test edilebilirlik
- **In-memory Cache:** Arama 5dk, detay 30dk TTL
- **Context Propagation:** Tum TMDB istekleri request context'i tasir
- **Graceful Shutdown:** SIGINT/SIGTERM ile duzgun kapanma
- **HTTP Timeout:** 10 saniye TMDB istek limiti

### Guvenlik
- Non-root Docker container
- Input validation (ID, page parametreleri)
- CORS: localhost:3000/5173, seyirlik.net izinli

### Kod Organizasyonu
- DRY: `getWatchProviders()`, `getCredits()` ortak fonksiyonlar
- Standart hata yaniti: `models.APIError`
- Page limiti: 1-500 (TMDB limiti)

## CI/CD

- `main` -> prod (seyirlik.net)
- `preprod` / `feature/**` -> test (test.seyirlik.net)
- Docker image: `{DOCKER_USERNAME}/seyirlik-be:{env}-{sha}`

## Iliskili Projeler

- **Frontend:** seyirlik-fe (ayri repo)
- **Infra:** seyirlik_infra (sunucuda docker-compose)
