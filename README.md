# bug-free-umbrella

[![Tests](https://github.com/scaryPonens/bug-free-umbrella/actions/workflows/tests.yml/badge.svg?branch=main)](https://github.com/scaryPonens/bug-free-umbrella/actions/workflows/tests.yml)
![Coverage](https://img.shields.io/badge/Coverage-77%25-brightgreen.svg)


Go-based crypto trading advisor bot. Tracks live crypto prices, stores OHLCV candles, and serves data via HTTP and Telegram.

- [Gin](https://github.com/gin-gonic/gin) web API with Swagger docs
- CoinGecko integration — live prices for 10 assets (BTC, ETH, SOL, XRP, ADA, DOGE, DOT, AVAX, LINK, MATIC)
- OHLCV candle storage in Postgres (5m, 15m, 1h, 4h, 1d intervals)
- Redis cache-aside for latest prices
- Background polling with rate-limited CoinGecko API calls
- Telegram bot (`/ping`, `/price`, `/volume`)
- OpenTelemetry tracing with Jaeger
- Configurable via `.env` file

## Stack

- **Go / Gin** - HTTP framework
- **OpenTelemetry** - Distributed tracing (exported via gRPC to an OTel Collector)
- **Jaeger** - Trace visualization
- **Swag** - Auto-generated OpenAPI/Swagger docs from annotations

## Project Structure

```
cmd/server/            Entrypoint and dependency wiring
internal/bot/          Telegram bot commands
internal/cache/        Redis client initialization
internal/config/       Environment variable loading
internal/db/           Postgres connection pool
internal/domain/       Domain types (Candle, PriceSnapshot, Asset, Signal)
internal/handler/      HTTP handlers with Swagger annotations
internal/job/          Background polling jobs (price poller)
internal/provider/     External API clients (CoinGecko) and rate limiter
internal/repository/   Postgres persistence (candle repository, migrations)
internal/service/      Business logic (price service, work service)
pkg/tracing/           OpenTelemetry initialization
docs/                  Generated Swagger spec (do not edit manually)
```


## Setup & Running

1. Copy `.env.example` to `.env` and fill in required secrets (see below).
2. Start all services (API, Postgres, Redis, OTel, Jaeger) with:

```sh
docker compose --env-file .env up --build
```

The app will be available at http://localhost:8080

### .env file

Example:

```env
TRACING_ENABLED=false
OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317
GIN_MODE=debug

# Telegram Bot
TELEGRAM_BOT_TOKEN=your-telegram-bot-token

# Postgres Database
DATABASE_URL=postgres://postgres:postgres@db:5432/postgres?sslmode=disable

# Redis
REDIS_URL=redis:6379

# CoinGecko polling interval in seconds (default 60)
COINGECKO_POLL_SECS=60
```

> **Note:** The default Docker Compose setup will run Postgres and Redis containers for you. The app will auto-connect using the above variables.

| Service     | URL                                    |
|-------------|----------------------------------------|
| API         | http://localhost:8080                  |
| Swagger UI  | http://localhost:8080/swagger/index.html |
| Jaeger UI   | http://localhost:16686                 |
| Postgres    | localhost:5432 (inside Docker: db:5432) |
| Redis       | localhost:6379 (inside Docker: redis:6379) |


## API Endpoints

| Method | Path                  | Description                                    |
|--------|-----------------------|------------------------------------------------|
| GET    | /health               | Health check                                   |
| GET    | /api/prices           | Current prices for all 10 tracked assets       |
| GET    | /api/prices/:symbol   | Current price for a specific asset (e.g. BTC)  |
| GET    | /api/candles/:symbol  | OHLCV candles (`?interval=1h&limit=100`)       |

Supported candle intervals: `5m`, `15m`, `1h`, `4h`, `1d`. Default limit is 100 (max 500).

## Telegram Bot

Set `TELEGRAM_BOT_TOKEN` in your `.env` file to enable the bot.

| Command         | Description                              |
|-----------------|------------------------------------------|
| /ping           | Health check — replies `pong`            |
| /price BTC      | Current price, 24h change, 24h volume    |
| /volume SOL     | 24h trading volume, price, 24h change    |

Supported symbols: BTC, ETH, SOL, XRP, ADA, DOGE, DOT, AVAX, LINK, MATIC.

## Background Polling

The app runs a 3-tier background poller against the CoinGecko free API (~1.5 calls/min):

| Tier | What                      | Frequency  |
|------|---------------------------|------------|
| 1    | Current prices (all 10)   | Every 60s  |
| 2    | Short candles (5m/15m/1h) | Every 5min |
| 3    | Long candles (4h/1d)      | Every 30min|

Polling interval is configurable via `COINGECKO_POLL_SECS` (default 60).


## Regenerating Swagger Docs

After adding or modifying handler annotations:

```sh
swag init -g cmd/server/main.go
```

This runs automatically during the Docker build.
