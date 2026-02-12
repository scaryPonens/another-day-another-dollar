CREATE TABLE IF NOT EXISTS conversation_messages (
    id          BIGSERIAL       PRIMARY KEY,
    chat_id     BIGINT          NOT NULL,
    role        TEXT            NOT NULL,
    content     TEXT            NOT NULL,
    created_at  TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_conversation_messages_chat_created
    ON conversation_messages (chat_id, created_at DESC);
