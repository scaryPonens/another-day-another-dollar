CREATE TABLE IF NOT EXISTS ml_feature_rows (
    symbol         TEXT        NOT NULL,
    interval       TEXT        NOT NULL,
    open_time      TIMESTAMPTZ NOT NULL,
    ret_1h         DOUBLE PRECISION NOT NULL,
    ret_4h         DOUBLE PRECISION NOT NULL,
    ret_12h        DOUBLE PRECISION NOT NULL,
    ret_24h        DOUBLE PRECISION NOT NULL,
    volatility_6h  DOUBLE PRECISION NOT NULL,
    volatility_24h DOUBLE PRECISION NOT NULL,
    volume_z_24h   DOUBLE PRECISION NOT NULL,
    rsi_14         DOUBLE PRECISION NOT NULL,
    macd_line      DOUBLE PRECISION NOT NULL,
    macd_signal    DOUBLE PRECISION NOT NULL,
    macd_hist      DOUBLE PRECISION NOT NULL,
    bb_pos         DOUBLE PRECISION NOT NULL,
    bb_width       DOUBLE PRECISION NOT NULL,
    target_up_4h   BOOLEAN,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (symbol, interval, open_time)
);

CREATE INDEX IF NOT EXISTS idx_ml_feature_rows_symbol_interval_time
    ON ml_feature_rows (symbol, interval, open_time DESC);

CREATE INDEX IF NOT EXISTS idx_ml_feature_rows_labeled
    ON ml_feature_rows (symbol, interval, open_time DESC)
    WHERE target_up_4h IS NOT NULL;

CREATE TABLE IF NOT EXISTS ml_model_versions (
    id                   BIGSERIAL PRIMARY KEY,
    model_key            TEXT        NOT NULL,
    version              INTEGER     NOT NULL,
    feature_spec_version TEXT        NOT NULL,
    trained_from         TIMESTAMPTZ NOT NULL,
    trained_to           TIMESTAMPTZ NOT NULL,
    trained_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    hyperparams_json     TEXT        NOT NULL DEFAULT '{}',
    metrics_json         TEXT        NOT NULL DEFAULT '{}',
    artifact_format      TEXT        NOT NULL,
    artifact_blob        BYTEA       NOT NULL,
    is_active            BOOLEAN     NOT NULL DEFAULT FALSE,
    activated_at         TIMESTAMPTZ,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (model_key, version)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_ml_model_versions_active_unique
    ON ml_model_versions (model_key)
    WHERE is_active = TRUE;

CREATE INDEX IF NOT EXISTS idx_ml_model_versions_lookup
    ON ml_model_versions (model_key, version DESC);

CREATE TABLE IF NOT EXISTS ml_predictions (
    id              BIGSERIAL PRIMARY KEY,
    symbol          TEXT        NOT NULL,
    interval        TEXT        NOT NULL,
    open_time       TIMESTAMPTZ NOT NULL,
    target_time     TIMESTAMPTZ NOT NULL,
    model_key       TEXT        NOT NULL,
    model_version   INTEGER     NOT NULL,
    prob_up         DOUBLE PRECISION NOT NULL,
    confidence      DOUBLE PRECISION NOT NULL,
    direction       TEXT        NOT NULL,
    risk            SMALLINT    NOT NULL,
    signal_id       BIGINT REFERENCES signals(id) ON DELETE SET NULL,
    details_json    TEXT        NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at     TIMESTAMPTZ,
    actual_up       BOOLEAN,
    is_correct      BOOLEAN,
    realized_return DOUBLE PRECISION,
    UNIQUE (symbol, interval, open_time, model_key, model_version)
);

CREATE INDEX IF NOT EXISTS idx_ml_predictions_unresolved
    ON ml_predictions (model_key, target_time)
    WHERE resolved_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_ml_predictions_history
    ON ml_predictions (model_key, created_at DESC);

CREATE OR REPLACE VIEW ml_accuracy_daily AS
SELECT
    model_key,
    DATE_TRUNC('day', resolved_at AT TIME ZONE 'UTC') AS day_utc,
    COUNT(*) AS total,
    COUNT(*) FILTER (WHERE is_correct IS TRUE) AS correct,
    CASE WHEN COUNT(*) = 0 THEN 0
         ELSE COUNT(*) FILTER (WHERE is_correct IS TRUE)::DOUBLE PRECISION / COUNT(*)::DOUBLE PRECISION
    END AS accuracy
FROM ml_predictions
WHERE resolved_at IS NOT NULL
GROUP BY model_key, DATE_TRUNC('day', resolved_at AT TIME ZONE 'UTC');
