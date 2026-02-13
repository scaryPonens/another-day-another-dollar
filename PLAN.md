# Crypto Trading Advisor Bot — Phased Plan

> **Project:** A Go-based crypto trading advisor bot that follows crypto prices, trade volumes, and sentiment using deterministic signal generation plus an advisor layer. Recommendations are scored on a 1–5 risk spectrum. Interfaces via Telegram bot and, later, an SSH terminal UI (à la terminal.shop). Deploy target: Railway app service with NeonDB + Redis Cloud.

---


## Phase 0: Foundation & Skeleton ✅ **(Completed)**

**Goal:** Deployable Go service on Railway that does *one thing* end-to-end.

- Go module scaffolded with clean project layout (`cmd/`, `internal/`, `pkg/`)
- Telegram bot basics via [telebot](https://github.com/tucnak/telebot) — responds to `/ping`
- Postgres (local via Docker, Railway-ready) as primary store
- Redis (local via Docker, Railway-ready) for caching/rate limiting
- Config management (env vars, Railway secrets, .env file supported)
- CI: GitHub Actions → deploy to Railway (**pending**)
- Basic domain types: `Asset`, `Signal`, `Recommendation`, `RiskLevel(1-5)`

**Deliverable:**
- Bot replies to you on Telegram (`/ping` → `pong`)
- App runs locally with Docker Compose (includes Postgres, Redis, OTel, Jaeger)
- Database and Redis connections established
- Configurable via `.env` file

**Status:**
Phase 0 is complete. The project is scaffolded, all core services are wired, and local development is fully supported. CI/CD to Railway is the only remaining step for full automation.

---

## Phase 1: Price Feeds & Storage ✅ **(Completed)**

**Goal:** Ingest and persist real-time crypto price/volume data.

- CoinGecko free tier API integration (no key needed, ~10 calls/min)
- Top 10 assets tracked: BTC, ETH, SOL, XRP, ADA, DOGE, DOT, AVAX, LINK, MATIC
- OHLCV candle intervals: 5m, 15m, 1h, 4h, 1d
- 3-tier background polling (current prices every 60s, short candles every 5m, long candles every 30m)
- Token bucket rate limiter (8 tokens/min) to stay within CoinGecko limits
- Candles stored in Postgres with composite PK `(symbol, interval, open_time)` and idempotent upserts
- Redis cache-aside for latest prices (90s TTL, refreshed every 60s)
- HTTP endpoints: `GET /api/prices`, `GET /api/prices/:symbol`, `GET /api/candles/:symbol`
- Telegram commands: `/price BTC`, `/volume SOL`
- Automated CI: GitHub Actions workflow runs `go test ./...` with coverage on every push to `main`, with README badges surfacing test/coverage status
- Extensive unit test suite covering services, handlers, providers, caching, tracing, and bootstrapping logic (added in previous iteration)

**Deliverable:** Bot responds with live prices. Data accumulates in Postgres. Prices cached in Redis.

**Status:**
Phase 1 is complete. Price ingestion pipeline is fully wired — CoinGecko provider, Postgres candle storage, Redis caching, background polling, HTTP endpoints, and Telegram commands are all operational. CI now enforces the growing unit test suite (85%+ package-level coverage; ~77% total) and README badges advertise build health.

---

## Phase 2: Signal Engine v1 — Classic Indicators ✅ **(Completed)**

**Goal:** First actual trade signals using well-understood technical analysis.

- Implement as a pipeline of pure functions: `[]Candle → []Signal`
- Start with a few classics:
  - **RSI** (oversold/overbought)
  - **MACD** crossovers
  - **Bollinger Band** breakouts
  - **Volume anomaly detection** (simple z-score)
- Each signal tagged with a `RiskLevel`:
  - RSI divergence on daily = risk 2
  - MACD cross on 15m = risk 4
  - Bollinger squeeze breakout on 5m = risk 5
- Store signals in Postgres with timestamp, asset, indicator, risk, direction (long/short/hold)
- Telegram: `/signals BTC`, `/signals --risk 3` filters

**Deliverable:** Bot proactively (or on-demand) tells you "RSI on ETH is oversold on 4h, risk 2."

**Status:**
Phase 2 is complete. The repository now includes:
- A deterministic classic indicator engine (`RSI`, `MACD`, `Bollinger breakout`, `volume z-score`) implemented as pure candle-to-signal functions.
- Signal persistence in Postgres with idempotent inserts and query filters.
- Signal generation poller (short and long interval tiers) plus HTTP endpoint `GET /api/signals`.
- Telegram signal command support (`/signals BTC`, `/signals --risk 3`).
- Proactive Telegram signal alerts with chat-level opt-in/out commands (`/alerts on|off|status`).

---

## Phase 3: MCP Service Layer (Data + Tools First) ✅ **(Completed)**

**Goal:** Expose bot capabilities through an MCP service before building the advisor UI/LLM layer.

- Stand up a dedicated Go MCP binary at `cmd/mcp` using existing business services as source of truth (`PriceService`, `SignalService`)
- Expose both transports (env-selected, one per process):
  - `stdio` for local trusted clients
  - streamable HTTP for remote clients
- Add MCP tools:
  - `prices_list_latest`
  - `prices_get_by_symbol`
  - `candles_list`
  - `signals_list`
  - `signals_generate` (generate + persist)
- Add MCP resources:
  - `market://supported-symbols`
  - `market://supported-intervals`
  - `prices://latest`
  - `prices://symbol/{symbol}`
  - `candles://{symbol}/{interval}?limit={n}`
  - `signals://latest?symbol={s}&risk={r}&indicator={i}&limit={n}`
- Enforce HTTP bearer-token auth from day one (`MCP_AUTH_TOKEN`)
- Add operational guardrails:
  - request timeout middleware
  - max HTTP body size
  - in-process rate limiting
  - tracing spans around MCP tool/resource calls
- Use MCP client-compatible tool naming (`[a-zA-Z0-9_-]` only, no dot-separated names)

**Deliverable:** Any MCP-compatible client can query prices/candles/signals and invoke supported tools without using Telegram or direct DB access.

**Status:**
Phase 3 is complete. MCP package + binary, transports, auth/guardrails, tests, and docs are implemented and integrated without changing existing API/Telegram behavior.

---

## Phase 4: LLM Integration — The Advisor Layer ✅ **(Completed)**

**Goal:** Natural language interaction powered by an LLM that synthesizes signals via the MCP surface.

- OpenAI API integrated as the "advisor brain" via official Go SDK (`github.com/openai/openai-go`)
- System prompt encodes risk framework (1–5 spectrum), trading philosophy, and live market data as dynamic context
- Data fetched through the same service interfaces MCP uses (`PriceService`, `SignalService`) for consistency
- Flow: `user message → extract symbols → gather context (signals + prices) → construct prompt → LLM response → Telegram`
- Conversation history stored in Postgres (`conversation_messages` table, keyed by Telegram `chat_id`)
- Two interaction modes: free-text messages and `/ask` command both route to the advisor
- Existing slash commands (`/price`, `/signals`, `/alerts`) remain unchanged for structured data access
- The LLM doesn't *make* the signals — it **interprets and communicates** them
- Graceful degradation: advisor is optional (`OPENAI_API_KEY` not set → disabled), LLM/store/context failures handled independently
- OpenTelemetry tracing on advisor calls (`advisor.ask`, `advisor.gather-context`, `advisor.llm-call`)

**Deliverable:** You chat with the bot naturally. It references real data and real signals through a stable integration layer.

**Status:**
Phase 4 is complete. The advisor layer includes:
- `internal/advisor/` package: `AdvisorService` (orchestrates context + LLM), `BuildSystemPrompt` (dynamic prompt with market data), `ExtractSymbols` (targeted context from user messages), `NewOpenAIClient` (SDK wrapper)
- `internal/repository/conversation_repository.go`: Postgres persistence for per-user conversation threads
- Migration `000003_create_conversations` for the `conversation_messages` table
- Telegram bot updated with `/ask` command and `tele.OnText` free-text handler
- Config: `OPENAI_API_KEY`, `OPENAI_MODEL` (default `gpt-4o-mini`), `ADVISOR_MAX_HISTORY` (default 20)
- Full unit test coverage across all new code

---

## Phase 5: Signal Visualization — Chart Images for Signals ✅ **(Completed)**

**Goal:** Generate a visual chart artifact for each generated signal so users can see the exact technical setup (candles + triggering indicator + signal annotation).

- Generate PNG candlestick charts at signal-generation time.
- Render only the triggering indicator for each signal.
- Store image bytes in Postgres (`BYTEA`) with metadata and expiry.
- Attach one image per signal in Telegram signal outputs and proactive alerts.
- Expose image metadata in HTTP signal responses and expose binary retrieval endpoints.
- Use non-blocking failure policy (signal persists even if chart render fails; retry async).
- Retain images for 24 hours with scheduled cleanup.

**Deliverable:** Every newly generated signal has either an associated image or a recorded render failure state; Telegram/API consumers can fetch chart images for recent signals.

**Status:**
Phase 5 is complete. The repository now includes:
- `internal/chart/` Go-native signal chart renderer (candlesticks + per-signal indicator overlays).
- `signal_images` Postgres table with status/error/retry metadata and expiry cleanup support.
- Signal pipeline integration that persists signal IDs, renders/stores images, and records retryable failures.
- Image maintenance job for periodic retry attempts and hourly expiry cleanup.
- HTTP support for image retrieval (`GET /api/signals/:id/image`) and signal payload metadata.
- Telegram `/signals` and proactive alerts now send one image per signal when available, with text fallback.

---

## Phase 6: ML Signal Engine v2 ✅ **(Completed)**

**Goal:** Layer in learned models alongside classic indicators.

- **Feature store** in Postgres: pre-computed features per asset per timeframe
- Start simple:
  - **Logistic regression / XGBoost** (call out to a Python sidecar or use a Go ML lib like `golearn`) for "will price be higher in N hours?" binary classification
  - Train on your accumulated OHLCV + indicator data
- More advanced (later in this phase):
  - **Anomaly detection** on volume/price patterns (isolation forest)
- Each ML model outputs a confidence score → mapped to risk level
- Ensemble: combine classic signals + ML signals with weighted voting
- A/B track: store predictions vs actuals to measure model accuracy over time

**Deliverable:** Signals now come from both rule-based and ML sources. You have a backtesting table showing accuracy.

**Status:**
Phase 6 development is **complete in-repo** (Go-native implementation finished). The repository now includes:
- Postgres ML schema + reporting view via migration `000005_create_ml_phase6`:
  - `ml_feature_rows`
  - `ml_model_versions`
  - `ml_predictions`
  - `ml_accuracy_daily` view
- ML domain additions (`MLFeatureRow`, `MLModelVersion`, `MLPrediction`) and ML indicator keys:
  - `ml_logreg_up4h`
  - `ml_xgboost_up4h`
  - `ml_ensemble_up4h`
- Go-native ML stack under `internal/ml/`:
  - feature engine (1h interval, 4h target labels)
  - logistic regression model
  - XGBoost-style model via `github.com/rmera/boo`
  - Isolation Forest anomaly model via `github.com/narumiruna/go-iforest`
  - model registry repository
  - prediction repository (including resolve lifecycle)
  - training service (chronological split + metrics + promotion logic + anomaly training on 1h/4h)
  - inference service (ML + ensemble predictions and signal emission + anomaly persistence/dampening)
  - ensemble scorer (classic + ML weighted voting)
- New `internal/ta/` shared indicator helpers for RSI/MACD/Bollinger/statistics.
- ML orchestration service and background jobs:
  - feature/inference cycle (multi-interval feature refresh)
  - daily training cycle
  - outcome resolver cycle (directional-only resolution)
- Config support for ML runtime controls (`ML_ENABLED`, cadence/threshold/window vars), including:
  - `ML_INTERVALS`
  - `ML_ENABLE_IFOREST`
  - `ML_ANOMALY_THRESHOLD`
  - `ML_ANOMALY_DAMP_MAX`
  - `ML_IFOREST_TREES`
  - `ML_IFOREST_SAMPLE_SIZE`
- Existing signal surfaces now accept ML indicators:
  - HTTP `GET /api/signals`
  - MCP `signals_list` indicator normalization
- Manual trigger endpoint for training:
  - `POST /api/ml/train`
- Backfill CLI supports interval selection:
  - `go run ./cmd/mlbackfill --intervals 1h,4h ...`

**Post-completion rollout checklist (ops/environment):**
- Run one-time historical candle backfill in each target environment (`1h` + `4h`).
- Enable `ML_ENABLED=true` in production and validate active model versions (`logreg`, `xgboost`, `iforest_1h`, `iforest_4h`).
- Monitor directional accuracy and anomaly-score drift using `ml_accuracy_daily` and `ml_predictions` details.

---

## Phase 7: Fundamentals & Sentiment ✅ **(Completed)**

**Goal:** Add non-price market intelligence signals without introducing paid on-chain dependencies.

**Status:**
Phase 7 is complete with a Go-native implementation in this repo.

Implemented scope:
- New migration `000006_create_market_intel_phase7` with:
  - `market_intel_items`
  - `market_intel_item_symbols`
  - `market_onchain_snapshots`
  - `market_composite_snapshots`
- Composite fundamentals/sentiment indicator:
  - `fund_sentiment_composite`
  - emitted on `1h` and `4h`
  - persisted to existing `signals` surfaces (`/api/signals`, Telegram `/signals`, MCP `signals_list`)
- Sentiment ingestion and scoring pipeline:
  - Fear & Greed ingestion (Alternative.me)
  - RSS news ingestion
  - Reddit ingestion
  - LLM scoring with deterministic heuristic fallback
- Free chain-specific on-chain proxy adapters (no Glassnode):
  - BTC: mempool.space
  - ETH: Blockscout
  - ADA: Koios
  - XRP: XRPSCAN
- Composite scoring logic:
  - weighted component aggregation with renormalization when components are missing
  - directional thresholds for `long`/`short`
  - `hold` retained only in `market_composite_snapshots` (no `signals` row)
- Job + manual trigger:
  - scheduled market-intel background job
  - `POST /api/market-intel/run` manual run endpoint
- Advisor context update:
  - advisor now explicitly includes fundamentals/sentiment composite signals when available

Compatibility behavior:
- Classic and ML signal engines remain unchanged.
- Phase 7 signals are text-only in this phase (no chart image rendering).
- Existing API/Telegram/MCP signal interfaces remain backward compatible.

**Post-completion rollout checklist (ops/environment):**
- Run `./migrate up` to apply `000006`.
- Deploy with `MARKET_INTEL_ENABLED=false`.
- Trigger `POST /api/market-intel/run` and verify new tables + `fund_sentiment_composite` signal rows.
- Enable `MARKET_INTEL_ENABLED=true`.
- Monitor run warnings (`errors` array from manual trigger) plus composite signal distribution by interval/symbol.

---

## Phase 8: SSH Terminal Interface

**Goal:** `ssh trading@yourdomain.com` gives you a TUI dashboard.

- Use [Wish](https://github.com/charmbracelet/wish) (Charm's SSH server library for Go) + [Bubble Tea](https://github.com/charmbracelet/bubbletea) for the TUI
- Screens:
  - **Dashboard**: live prices, active signals, portfolio heat map
  - **Chat**: same LLM advisor, but in terminal
  - **Signal explorer**: filter by asset, risk, indicator type
  - **Backtest viewer**: model accuracy over time
- Auth via SSH keys (map public keys to users in Supabase)
- Deploy as a separate Railway service exposing port 22 (or custom port)

**Deliverable:** Full terminal.shop-style experience. You SSH in and get a live trading advisor dashboard.

---

## Phase 9: Hardening & Portfolio Tracking

- Paper trading / portfolio simulation (track hypothetical positions based on signals you "accept")
- Risk-adjusted returns tracking (Sharpe ratio per risk level)
- Alert system: proactive Telegram/SSH notifications when high-confidence signals fire
- Rate limiting, error budgets, observability (structured logging, OpenTelemetry)
- Multi-user support if you want to share it

**Smoke testing strategy (add as a deploy gate in this phase):**
- Add `scripts/smoke/server_http_smoke.sh` for core API checks:
  - `GET /health` returns `200`
  - `GET /api/prices` returns non-empty symbols
  - `GET /api/signals?limit=1` returns valid JSON
- Add `scripts/smoke/mcp_http_smoke.sh` for MCP JSON-RPC over HTTP:
  - unauthorized request without bearer token must fail (`401`/`403`)
  - authorized `tools/list` succeeds
  - authorized `tools/call` for `prices_list_latest` succeeds
  - authorized `tools/call` for `signals_generate` succeeds and returns `generated_count`
  - authorized `resources/read` for `market://supported-symbols` succeeds
- Add `scripts/smoke/mcp_stdio_smoke.sh` for local/CI stdio transport:
  - starts `go run ./cmd/mcp` with `MCP_TRANSPORT=stdio`
  - sends minimal JSON-RPC initialize + tool call payload
  - verifies expected response shape (tool result present, no protocol error)
- Add a single entrypoint script `scripts/smoke/run_all.sh`:
  - fails fast on first failed check
  - prints a concise pass/fail summary per subsystem (API, MCP HTTP, MCP stdio)
- Wire smoke scripts into release flow:
  - run locally before Railway deploy
  - run in CI on `main` after `go test ./...`
  - run as first post-deploy verification against Railway service URLs

**Status update (early work pulled forward):**
- Telegram-side proactive alerts are already partially implemented in Phase 2 (`/alerts on|off|status`), with dedupe logic to avoid repeated notifications for identical generated signals.

---

## Cross-Phase Infra Update ✅ **(Completed)**

**Goal:** Make schema evolution and deployment safer for production.

- Migration SQL moved out of repository `RunMigrations` methods into versioned SQL files.
- Dedicated migration runner added at `cmd/migrate` (`up`, `down`, `version`).
- App startup no longer mutates database schema automatically.
- Container image now includes both `main` and `migrate` binaries for deploy-time migrations.

**Status:**
Complete. Recommended deployment flow is now:
1. Run `./migrate up`
2. Start `./main`

## Architecture at a Glance

```
┌─────────────┐     ┌─────────────┐
│  Telegram    │     │  SSH/TUI    │
│  Interface   │     │  (Wish+BT)  │
└──────┬───────┘     └──────┬──────┘
       │                    │
       └────────┬───────────┘
                │
         ┌──────▼──────┐
         │  MCP Service │ ← tools/resources contract
         │  (Go service)│
         └──────┬───────┘
                │
         ┌──────▼──────┐
         │  LLM Advisor │ ← Claude/OpenAI API
         │  (consumer)  │
         └──────┬───────┘
                │
         ┌──────▼──────┐
         │ Signal Engine │
         │ ┌───────────┐│
         │ │ Classic TA ││ ← pure functions
         │ ├───────────┤│
         │ │ ML Models  ││ ← Go or Python sidecar
         │ ├───────────┤│
         │ │ Sentiment  ││ ← LLM batch scoring
         │ └───────────┘│
         └──────┬───────┘
                │
         ┌──────▼──────┐
         │Chart Renderer│ ← Go PNG rendering for signal artifacts
         │    (Go)      │
         └──────┬───────┘
                │
    ┌───────────┼───────────┐
    │           │           │
┌───▼──────────┐ ┌────▼───┐ ┌────▼───┐
│NeonDB        │ │Supabase│ │ Redis  │
│(OHLCV,       │ │(users, │ │(cache, │
│ signals,     │ │signals)│ │ rates) │
│ signal_images)│ └────────┘ └────────┘
└──────────────┘
```

---

## Suggested Tech Choices (Revalidated)

| Concern | Choice | Why |
|---|---|---|
| Telegram bot | `telebot` (already integrated) | Existing implementation is live and tested |
| MCP | Go MCP server | Stable integration contract before advisor clients |
| SSH TUI | Wish + Bubble Tea | Charm ecosystem, built for this |
| LLM | Claude API | Already in the ecosystem |
| TA indicators | `github.com/sdcoffey/techan` or hand-roll | Techan is decent; pure functions are easy to write in Go |
| ML sidecar | Python (FastAPI + scikit-learn/XGBoost) | Lean into Python strength for ML parts |
| Observability | Structured logging + OpenTelemetry | Production-grade from day one |

---

## Sprint Plan

| Sprint | Phases | Duration | Key Milestone |
|---|---|---|---|
| 1 | 0 + 1 | 1–2 weeks | Bot responds with live prices, data accumulating |
| 2 | 2 | 1–2 weeks | First trading signals (RSI, MACD, Bollinger) ✅ |
| 3 | 3 | 1 week | MCP service exposed for prices/candles/signals |
| 4 | 4 | 1 week | Natural language advisor uses MCP-backed context |
| 5 | 5 | 1 week | Signal chart images generated/stored and delivered via Telegram/API/MCP |
| 6 | 6 | 2–3 weeks | ML models running, backtesting accuracy tracked |
| 7 | 7 | 2 weeks | Sentiment + on-chain signals integrated |
| 8 | 8 | 1–2 weeks | SSH terminal interface live |
| 9 | 9 | Ongoing | Paper trading, alerts, hardening |
