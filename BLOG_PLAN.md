### Blog Series Plan: Building a Go-Based Crypto Trading Advisor

**Objective:** A series of short, focused articles walking readers through the journey of creating a sophisticated crypto analysis bot, using this project as the real-world example.

---

### **Post 1: The Foundation - A Go-Powered Crypto Bot** ✅ **(Written — `post1.md`)**

*   **Core Idea:** Introduce the project's goal and the foundational technology choices.
*   **Key Talking Points:**
    *   Why Go is a great fit: concurrency for polling, performance for data processing.
    *   **The Core Stack:** Introducing Gin for the API, PostgreSQL for persistent data, and Redis for high-speed caching.
    *   **Docker-First Development:** Explain how `docker-compose.yml` creates a consistent, one-command setup for all services (API, DB, Cache, etc.).
    *   **Project Entrypoint:** A brief look at `cmd/server/main.go` to show how services are wired together.

---

### **Post 2: Spec-Driven Development - Building with an AI Co-Pilot Rotation** ✅ **(Written — `post2.md`)**

*   **Core Idea:** How maintaining a living specification document makes it possible to work fluidly across multiple LLM platforms — and never lose context between sessions.
*   **Key Talking Points:**
    *   **The Token Wall:** The practical reality of hitting context limits mid-feature in Claude, ChatGPT, or Gemini, and why this is a workflow problem worth solving.
    *   **The Living Spec:** How `PLAN.md` acts as the single source of truth — not just for the project, but for whichever AI you're currently talking to. Structured phases, completion status, and explicit deliverables give any model enough context to resume intelligently.
    *   **The Handoff Protocol:** A look at the discipline of updating `PLAN.md` before switching platforms — marking phases complete, noting in-progress state, and capturing any non-obvious decisions so the next session (and the next model) doesn't re-litigate them.
    *   **Platform Complementarity:** Each model has different strengths. Using Claude for architecture decisions, ChatGPT for boilerplate generation, and Gemini for long-context review isn't chaos — it's a workflow, as long as the spec stays current.
    *   **Spec as Documentation:** The same `PLAN.md` that guides AI sessions becomes the canonical record of *why* the system is built the way it is, which is valuable long after the code is written.

---

### **Post 3: The Data Pipeline - Ingesting Market Data** ✅ **(Written — `post3.md`)**


*   **Core Idea:** Detail the process of reliably fetching and storing market data, the lifeblood of the application.
*   **Key Talking Points:**
    *   **Polling the Market:** A dive into `internal/job/price_poller.go` and the multi-tier polling strategy for different candle intervals.
    *   **Respectful Scraping:** Implementing a rate limiter (`internal/provider/ratelimiter.go`) to politely interact with the CoinGecko API.
    *   **Storing Time-Series Data:** Discussing the database schema and the role of `internal/repository/candle_repository.go` in persisting OHLCV candles.

---

### **Post 4: Finding the Signal in the Noise - A TA Engine** ✅ **(Written — `post4.md`)**

*   **Core Idea:** Explain how the raw candle data is transformed into actionable trading signals using technical analysis.
*   **Key Talking Points:**
    *   **The Signal Engine:** A look inside `internal/signal/engine.go` to see how indicators like RSI, MACD, and Bollinger Bands are calculated.
    *   **Automated Analysis:** How the `signal_poller` job automates signal generation across different timeframes.
    *   **Serving the Signals:** Showcasing the `/api/signals` endpoint.

---

### **Post 5: A Picture is Worth a Thousand Pips - Visualizing Signals**

*   **Core Idea:** Showcasing the unique feature of generating chart images on-the-fly.
*   **Key Talking Points:**
    *   **Go-Native Charting:** Highlighting `internal/chart/renderer.go` and the process of drawing candlestick charts with indicator overlays without external dependencies.
    *   **Visuals on Demand:** How the `/api/signals/:id/image` endpoint serves these charts.
    *   **Bringing Charts to Chat:** Integrating these images into the Telegram bot for a richer user experience.

---

### **Post 6: Beyond Price - Integrating Sentiment and On-Chain Data**

*   **Core Idea:** Introduce the "Phase 7" market intelligence features, moving beyond pure price action.
*   **Key Talking Points:**
    *   **A Composite View:** Explaining the `fund_sentiment_composite` indicator.
    *   **The Data Sources:** A tour of the providers in `internal/provider/` for Fear & Greed, RSS, Reddit, and on-chain metrics.
    *   **Scoring and Weighting:** How subjective data is turned into a quantitative signal in `internal/marketintel/`.

---

### **Post 7: The Ghost in the Machine - An ML-Powered Ensemble**

*   **Core Idea:** Dive into the most advanced feature: using machine learning to create a "smarter" signal.
*   **Key Talking Points:**
    *   **Ensemble Power:** Explain how the ensemble model (`internal/ml/ensemble/`) combines classic TA signals with predictions from Logistic Regression and XGBoost models.
    *   **Anomaly Detection:** The role of the Isolation Forest model in identifying unusual market conditions and dampening the main model's conviction.
    *   **The ML Lifecycle:** Briefly touch upon the `mlbackfill` command for data preparation and the `ml_training_job` for keeping models fresh.

---

### **Post 8: Hello, World - Interacting with the Bot**

*   **Core Idea:** Showcase the three primary ways users and other programs can interact with the system.
*   **Key Talking Points:**
    *   **The REST API:** For traditional programmatic access.
    *   **The Telegram Bot:** For human-centric, on-the-go interaction (`internal/bot/`).
    *   **The MCP Service:** For next-generation AI agent integration (`internal/mcp/`), explaining the concept of tools and resources.

---