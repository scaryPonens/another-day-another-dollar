CREATE TABLE IF NOT EXISTS market_intel_items (
    id                    BIGSERIAL PRIMARY KEY,
    source                TEXT        NOT NULL,
    source_item_id        TEXT        NOT NULL,
    title                 TEXT        NOT NULL DEFAULT '',
    url                   TEXT        NOT NULL DEFAULT '',
    excerpt               TEXT        NOT NULL DEFAULT '',
    author                TEXT        NOT NULL DEFAULT '',
    published_at          TIMESTAMPTZ NOT NULL,
    fetched_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    metadata_json         TEXT        NOT NULL DEFAULT '{}',
    sentiment_score       DOUBLE PRECISION,
    sentiment_confidence  DOUBLE PRECISION,
    sentiment_label       TEXT,
    sentiment_model       TEXT,
    sentiment_reason      TEXT,
    scored_at             TIMESTAMPTZ,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (source, source_item_id)
);

CREATE INDEX IF NOT EXISTS idx_market_intel_items_published_source
    ON market_intel_items (published_at DESC, source);

CREATE INDEX IF NOT EXISTS idx_market_intel_items_unscored
    ON market_intel_items (scored_at)
    WHERE scored_at IS NULL;

CREATE TABLE IF NOT EXISTS market_intel_item_symbols (
    item_id     BIGINT      NOT NULL REFERENCES market_intel_items(id) ON DELETE CASCADE,
    symbol      TEXT        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (item_id, symbol)
);

CREATE INDEX IF NOT EXISTS idx_market_intel_item_symbols_symbol_item
    ON market_intel_item_symbols (symbol, item_id DESC);

CREATE TABLE IF NOT EXISTS market_onchain_snapshots (
    symbol        TEXT        NOT NULL,
    interval      TEXT        NOT NULL,
    bucket_time   TIMESTAMPTZ NOT NULL,
    provider_key  TEXT        NOT NULL,
    onchain_score DOUBLE PRECISION NOT NULL,
    confidence    DOUBLE PRECISION NOT NULL,
    details_json  TEXT        NOT NULL DEFAULT '{}',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (symbol, interval, bucket_time, provider_key)
);

CREATE INDEX IF NOT EXISTS idx_market_onchain_snapshots_lookup
    ON market_onchain_snapshots (symbol, interval, bucket_time DESC);

CREATE TABLE IF NOT EXISTS market_composite_snapshots (
    symbol                 TEXT        NOT NULL,
    interval               TEXT        NOT NULL,
    open_time              TIMESTAMPTZ NOT NULL,
    fear_greed_value       INTEGER,
    fear_greed_score       DOUBLE PRECISION,
    news_score             DOUBLE PRECISION,
    reddit_score           DOUBLE PRECISION,
    onchain_score          DOUBLE PRECISION,
    composite_score        DOUBLE PRECISION NOT NULL,
    confidence             DOUBLE PRECISION NOT NULL,
    direction              TEXT        NOT NULL,
    risk                   SMALLINT    NOT NULL,
    component_weights_json TEXT        NOT NULL DEFAULT '{}',
    details_json           TEXT        NOT NULL DEFAULT '{}',
    signal_id              BIGINT REFERENCES signals(id) ON DELETE SET NULL,
    created_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (symbol, interval, open_time)
);

CREATE INDEX IF NOT EXISTS idx_market_composite_snapshots_lookup
    ON market_composite_snapshots (symbol, interval, open_time DESC);

CREATE INDEX IF NOT EXISTS idx_market_composite_snapshots_interval_time
    ON market_composite_snapshots (interval, open_time DESC);
