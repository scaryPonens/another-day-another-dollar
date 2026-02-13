# bug-free-umbrella

[![Tests](https://github.com/scaryPonens/bug-free-umbrella/actions/workflows/tests.yml/badge.svg)](https://github.com/scaryPonens/bug-free-umbrella/actions/workflows/tests.yml)
![Coverage](https://img.shields.io/badge/Coverage-77%25-brightgreen.svg)


Go-based crypto trading advisor bot. Tracks live crypto prices, stores OHLCV candles, and serves data via HTTP, Telegram, and MCP.

- [Gin](https://github.com/gin-gonic/gin) web API with Swagger docs
- CoinGecko integration — live prices for 10 assets (BTC, ETH, SOL, XRP, ADA, DOGE, DOT, AVAX, LINK, MATIC)
- OHLCV candle storage in Postgres (5m, 15m, 1h, 4h, 1d intervals)
- Redis cache-aside for latest prices
- Background polling with rate-limited CoinGecko API calls
- Fundamentals/sentiment composite signals (`fund_sentiment_composite`) on `1h` and `4h`
- Telegram bot (`/ping`, `/price`, `/volume`, `/signals`, `/alerts`)
- MCP service (`stdio` + streamable HTTP transport) with tools/resources for prices, candles, and signals
- Signal chart imaging (candlestick + triggering indicator) stored in Postgres and served to Telegram/API/MCP
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
cmd/mcp/               MCP server binary (stdio + HTTP)
cmd/migrate/           Versioned Postgres schema migrations runner
internal/bot/          Telegram bot commands
internal/cache/        Redis client initialization
internal/chart/        Go-native signal chart image rendering
internal/config/       Environment variable loading
internal/db/           Postgres connection pool
internal/domain/       Domain types (Candle, PriceSnapshot, Asset, Signal)
internal/handler/      HTTP handlers with Swagger annotations
internal/job/          Background jobs (price/signal pollers + signal-image maintenance)
internal/provider/     External API clients (CoinGecko) and rate limiter
internal/repository/   Postgres persistence (candle repository, migrations)
internal/signal/       Pure technical-analysis signal engine (RSI/MACD/Bollinger/Volume)
internal/service/      Business logic (price service, signal service, work service)
internal/mcp/          MCP tools/resources, transport auth, and middleware
internal/marketintel/  Fundamentals/sentiment ingestion, scoring, and composite signal logic
pkg/tracing/           OpenTelemetry initialization
docs/                  Generated Swagger spec (do not edit manually)
```


## Setup & Running

1. Copy `.env.example` to `.env` and fill in required secrets (see below).
2. Run schema migrations:

```sh
docker compose --env-file .env run --rm app ./migrate up
```

3. Start all services (API, Postgres, Redis, OTel, Jaeger) with:

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
# or Redis Cloud URL (TLS):
# REDIS_URL=rediss://default:password@your-host:6380/0

# CoinGecko polling interval in seconds (default 60)
COINGECKO_POLL_SECS=60

# MCP
MCP_TRANSPORT=stdio
MCP_HTTP_ENABLED=false
MCP_HTTP_BIND=127.0.0.1
MCP_HTTP_PORT=8090
MCP_AUTH_TOKEN=change-me
MCP_REQUEST_TIMEOUT_SECS=5
MCP_RATE_LIMIT_PER_MIN=60
```

> **Note:** The default Docker Compose setup will run Postgres and Redis containers for you. The app will auto-connect using the above variables.
> In Railway, the platform sets `PORT` automatically and the app binds to that port.
> Migrations are no longer executed during app startup. Run `go run ./cmd/migrate up` (or `./migrate up` in the container) before starting the server.

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
| GET    | /api/signals          | Technical signals (`?symbol=BTC&risk=3&limit=50`) |
| GET    | /api/signals/:id/image | Signal chart image (`image/png`)                  |
| POST   | /api/ml/train         | Manually trigger ML training cycle (when ML is enabled) |
| POST   | /api/market-intel/run | Manually trigger one fundamentals/sentiment cycle |

Supported candle intervals: `5m`, `15m`, `1h`, `4h`, `1d`. Default limit is 100 (max 500).

## Telegram Bot

Set `TELEGRAM_BOT_TOKEN` in your `.env` file to enable the bot.

| Command         | Description                              |
|-----------------|------------------------------------------|
| /ping           | Health check — replies `pong`            |
| /price BTC      | Current price, 24h change, 24h volume    |
| /volume SOL     | 24h trading volume, price, 24h change    |
| /signals BTC    | Latest generated signals + chart images for an asset     |
| /signals --risk 3 | Latest signals + chart images filtered by risk level   |
| /alerts on      | Enable proactive signal push alerts       |
| /alerts off     | Disable proactive signal push alerts      |
| /alerts status  | Check whether proactive alerts are enabled |

Supported symbols: BTC, ETH, SOL, XRP, ADA, DOGE, DOT, AVAX, LINK, MATIC.

## Background Polling

The app runs a 3-tier background poller against the CoinGecko free API (~1.5 calls/min):

| Tier | What                      | Frequency  |
|------|---------------------------|------------|
| 1    | Current prices (all 10)   | Every 60s  |
| 2    | Short candles (5m/15m/1h) | Every 5min |
| 3    | Long candles (4h/1d)      | Every 30min|

Signal generation runs in a separate poller:

| Tier | What                            | Frequency  |
|------|---------------------------------|------------|
| 1    | Short-interval signals (5m/15m/1h) | Every 5min |
| 2    | Long-interval signals (4h/1d)       | Every 30min|

Polling interval is configurable via `COINGECKO_POLL_SECS` (default 60).

Signal image maintenance runs alongside polling:
- Retry failed signal renders every 5 minutes (bounded retries)
- Delete expired signal images every hour
- Image retention window: 24 hours

Market-intel polling (Phase 7):
- Ingests Fear & Greed + RSS news + Reddit and scores sentiment
- Collects on-chain proxy snapshots for BTC/ETH/ADA/XRP
- Writes composite snapshots every poll cycle for `1h` and `4h`
- Emits directional `fund_sentiment_composite` rows into `signals` (`long`/`short` only)

## MCP Service

Run MCP over stdio (local agent tools):

```sh
MCP_TRANSPORT=stdio go run ./cmd/mcp
```

Run MCP over HTTP (token required):

```sh
MCP_TRANSPORT=http \
MCP_HTTP_ENABLED=true \
MCP_HTTP_BIND=127.0.0.1 \
MCP_HTTP_PORT=8090 \
MCP_AUTH_TOKEN=change-me \
go run ./cmd/mcp
```

MCP tools:
- `prices_list_latest`
- `prices_get_by_symbol`
- `candles_list`
- `signals_list`
- `signals_generate` (generate + persist)

MCP resources:
- `market://supported-symbols`
- `market://supported-intervals`
- `prices://latest`
- `prices://symbol/{symbol}`
- `candles://{symbol}/{interval}?limit={n}`
- `signals://latest?symbol={s}&risk={r}&indicator={i}&limit={n}`

### Claude Desktop (stdio) local testing

1. Ensure Postgres + Redis are running and migrations are applied.
2. Add an MCP server entry to `~/Library/Application Support/Claude/claude_desktop_config.json`:
   (merge this under `mcpServers`; do not remove your existing `preferences` block)

```json
{
  "mcpServers": {
    "bug-free-umbrella": {
      "command": "/bin/zsh",
      "args": [
        "-lc",
        "cd /Users/reuben/Workspace/bug-free-umbrella && go run ./cmd/mcp"
      ],
      "env": {
        "MCP_TRANSPORT": "stdio"
      }
    }
  }
}
```

3. Restart Claude Desktop.

Sample validation prompts:
- `Give me a current market snapshot and rank the top 3 symbols by 24h volume.`
- `How is BTC doing right now? Include price, 24h change, and volume.`
- `Show the latest 20 candles for BTC on the 1h interval and summarize the trend.`
- `Generate fresh BTC signals for 1h and 4h, then summarize what was generated.`

## Database Migrations

Run pending migrations:

```sh
go run ./cmd/migrate up
```

If using Docker Compose:

```sh
docker compose --env-file .env run --rm app ./migrate up
```

Roll back one migration:

```sh
go run ./cmd/migrate down
```

Check current applied version:

```sh
go run ./cmd/migrate version
```

## ML Backfill (1h/4h candles)

Before enabling `ML_ENABLED=true`, backfill enough candle history for training and anomaly scoring.
Default behavior now follows `ML_INTERVALS` (fallback: `ML_INTERVAL`, then `1h`).

Run one-shot backfill for all supported symbols:

```sh
go run ./cmd/mlbackfill
```

Override days, symbols, and intervals:

```sh
go run ./cmd/mlbackfill --days 90 --symbols BTC,ETH,SOL --intervals 1h,4h
```

Defaults:
- `--days` defaults to `ML_BACKFILL_DAYS`, then `ML_TRAIN_WINDOW_DAYS`, then `90`
- `--symbols` defaults to all `SupportedSymbols`
- `--intervals` defaults to `ML_INTERVALS`, then `ML_INTERVAL`, then `1h`

Isolation Forest anomaly detection:
- Runs for configured ML intervals (for example `1h,4h`)
- Persists anomaly predictions to `ml_predictions` with `model_key=iforest_<interval>`
- Dampens ensemble conviction and can increase ensemble risk
- Does **not** emit standalone anomaly signal rows

## Ensemble Usage

The ensemble is emitted as `indicator=ml_ensemble_up4h` and combines classic TA signals with ML probabilities.

How it works:
- Base score:
  - `ensemble_base = 0.30*classic_score + 0.35*logreg_score + 0.35*xgboost_score`
  - `logreg_score` and `xgboost_score` are mapped from probability with `2*prob_up - 1`
  - `classic_score` is derived from same-timestamp classic signals (`long=+1`, `short=-1`, `hold=0`), weighted by `(6-risk)/5`
- Anomaly dampening:
  - `ensemble_score = ensemble_base * (1 - ML_ANOMALY_DAMP_MAX * anomaly_score)`
  - `anomaly_score` comes from `iforest_<interval>` prediction (0 to 1)
- Direction thresholds:
  - `ensemble_score > 0.15` => `long`
  - `ensemble_score < -0.15` => `short`
  - otherwise `hold` (prediction row only; no signal row)
- Risk:
  - Derived from confidence `abs(prob_up - 0.5) * 2`
  - If `anomaly_score >= ML_ANOMALY_THRESHOLD`, ensemble risk is bumped by `+1` (capped at 5)

Practical usage:
- Trigger/update models: `POST /api/ml/train`
- Fetch ensemble signals:
  - `GET /api/signals?indicator=ml_ensemble_up4h&limit=50`
  - Optional filter by symbol: `GET /api/signals?symbol=BTC&indicator=ml_ensemble_up4h`
- Read signal `details`:
  - includes `model_key=ensemble_v1`, `prob_up`, `confidence`, `target=4h`, `ensemble_score`
  - when anomaly is active, it also includes `anomaly_score` and `damp_factor`

## Fundamentals + Sentiment (Phase 7)

Phase 7 adds a non-price composite indicator:
- `indicator=fund_sentiment_composite`
- Intervals: `1h`, `4h`
- Inputs:
  - Fear & Greed index (Alternative.me)
  - RSS news headlines
  - Reddit posts
  - Free on-chain proxy metrics:
    - BTC: mempool.space
    - ETH: Blockscout
    - ADA: Koios
    - XRP: XRPSCAN

Scoring behavior:
- News/Reddit are scored by LLM when available, with deterministic heuristic fallback.
- Components are weighted and renormalized when data is missing.
- `signals` rows are created only for `long`/`short`; `hold` remains internal in `market_composite_snapshots`.

Operational usage:
- Manual run: `POST /api/market-intel/run`
- Query composite signals:
  - `GET /api/signals?indicator=fund_sentiment_composite&limit=50`
  - `GET /api/signals?symbol=BTC&indicator=fund_sentiment_composite`
- Example `details` format:
  - `model_key=fund_sent_v1;interval=1h;score=0.2431;confidence=0.5810;fng=-0.1200;news=0.4100;reddit=0.2100;onchain=0.0900`

Chart behavior:
- Phase 7 signals are text-only.
- `fund_sentiment_composite` does not render signal chart images in this phase.


## Regenerating Swagger Docs

After adding or modifying handler annotations:

```sh
swag init -g cmd/server/main.go
```

This runs automatically during the Docker build.
