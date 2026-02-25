# Claude Code Project Context: bug-free-umbrella

This file is automatically loaded by Claude Code at the start of every session.

## Project Overview

`bug-free-umbrella` is a Go-based crypto trading advisor bot. It ingests live market data, generates trading signals via classic TA, ML models, and sentiment analysis, and exposes everything through a REST API, Telegram bot, and MCP service.

## Current Status

Phases 0–7 are complete. Phase 8 (SSH terminal UI via Wish + Bubble Tea) is next.

See `PLAN.md` for the full phase history, deliverables, and status notes. Always check it before starting new work to understand what's been built and why.

## Architecture

```
cmd/server/            Main entrypoint — all dependency wiring lives here
cmd/mcp/               MCP server binary (stdio + HTTP transports)
cmd/migrate/           Migration runner (up/down/version)
cmd/mlbackfill/        One-time CLI for backfilling historical candle data

internal/bot/          Telegram bot command handlers
internal/advisor/      LLM advisor (OpenAI) — context gathering + prompt construction
internal/signal/       Classic TA engine (RSI, MACD, Bollinger, volume z-score)
internal/ml/           ML stack: features, logreg, xgboost, iforest, ensemble, training
internal/marketintel/  Sentiment/fundamentals pipeline (Fear & Greed, RSS, Reddit, on-chain)
internal/chart/        Go-native PNG chart renderer for signal artifacts
internal/job/          Background jobs (price poller, signal poller, image maintenance, ML)
internal/handler/      HTTP handlers (Gin) with Swagger annotations
internal/mcp/          MCP tools, resources, transport auth, middleware
internal/provider/     External API clients (CoinGecko) + token-bucket rate limiter
internal/repository/   All Postgres persistence (candles, signals, images, ML, conversations)
internal/service/      Business logic (PriceService, SignalService, MLOrchestrator, etc.)
internal/domain/       Shared domain types (Candle, Signal, Asset, MLFeatureRow, etc.)
internal/config/       Env var loading
pkg/tracing/           OpenTelemetry setup
```

## Key Conventions

- **Dependency injection everywhere** — no globals, no init() side effects. All wiring is explicit in `cmd/server/main.go`.
- **Repository pattern** — only repositories touch the DB. Services call repositories.
- **Pure signal functions** — the TA engine is `[]Candle → []Signal`, no side effects.
- **Idempotent upserts** — all DB writes use `ON CONFLICT DO UPDATE` or equivalent.
- **Non-blocking failure policy** — signal chart rendering failures are recorded and retried async; they never block signal persistence.
- **Migrations are versioned SQL files** — never modify schema in Go code. Use `cmd/migrate`.

## Running Locally

```bash
# Start all services (app, Postgres, Redis, Jaeger)
docker compose --env-file .env up --build

# Apply migrations
docker compose --env-file .env run --rm app ./migrate up

# Run tests
go test ./...

# Backfill ML candle data
go run ./cmd/mlbackfill
```

## Environment Variables (key ones)

| Var | Purpose |
|---|---|
| `DATABASE_URL` | Postgres connection string |
| `REDIS_URL` | Redis address |
| `TELEGRAM_BOT_TOKEN` | Telegram bot |
| `OPENAI_API_KEY` | LLM advisor (optional — disables advisor if unset) |
| `MCP_AUTH_TOKEN` | Bearer token for MCP HTTP transport |
| `ML_ENABLED` | Enable ML inference + training jobs |
| `MARKET_INTEL_ENABLED` | Enable sentiment/fundamentals pipeline |

## Blog Writing Guidelines

When writing or planning blog posts for this series:

- **Ask clarifying questions first** — before drafting, confirm the angle, the audience emphasis (aspiring programmer / manager / IC), and any specific anecdotes or personal details to weave in.
- **Voice** — wax nerd poetic. First-person, reflective, technically honest. Write like someone who genuinely loves building things and wants to bring the reader along, not like a tutorial author ticking boxes.
- **Technical language** — keep it light and engaging. Explain concepts through analogy and narrative before reaching for jargon. When jargon is unavoidable, earn it.
- **Ground everything in the codebase** — always pull concrete examples from `PLAN.md`, actual file paths, real snippets, or lived decisions from this project. No hypotheticals when the real thing exists.
- **Refer to `PLAN.md`** for phase history, deliverables, and status notes before writing about any phase or feature.
- **Match the established style** — read `post0.md`, `post1.md`, and `post2.md` before drafting a new post to stay consistent with tone, structure, and section rhythm.

## Database Schema (migrations in order)

- `000001` — candles, assets
- `000002` — signals, signal_images
- `000003` — conversation_messages (advisor history)
- `000004` — (reserved)
- `000005` — ML tables (ml_feature_rows, ml_model_versions, ml_predictions)
- `000006` — market intel tables (market_intel_items, on-chain snapshots, composite snapshots)
