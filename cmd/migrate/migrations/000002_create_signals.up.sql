CREATE TABLE IF NOT EXISTS signals (
    id          BIGSERIAL PRIMARY KEY,
    symbol      TEXT        NOT NULL,
    interval    TEXT        NOT NULL,
    indicator   TEXT        NOT NULL,
    direction   TEXT        NOT NULL,
    risk        SMALLINT    NOT NULL,
    timestamp   TIMESTAMPTZ NOT NULL,
    details     TEXT        NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (symbol, interval, indicator, timestamp, direction)
);

CREATE INDEX IF NOT EXISTS idx_signals_lookup
    ON signals (symbol, risk, indicator, timestamp DESC);
