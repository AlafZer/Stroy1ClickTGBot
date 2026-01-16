-- 0001_init.sql

CREATE TABLE IF NOT EXISTS schema_migrations (
                                                 version    TEXT PRIMARY KEY,
                                                 applied_at INTEGER NOT NULL
);

-- Одноразовые токены привязки (/start <TOKEN>):
-- token_hash = sha256(token) (32 bytes)
CREATE TABLE IF NOT EXISTS bind_tokens (
                                           token_hash  BLOB PRIMARY KEY,
                                           user_id     INTEGER NOT NULL,
                                           expires_at  INTEGER NOT NULL,  -- unix seconds UTC
                                           used_at     INTEGER            -- unix seconds UTC, NULL = не использован
);

CREATE INDEX IF NOT EXISTS idx_bind_tokens_user_active
    ON bind_tokens(user_id)
    WHERE used_at IS NULL;

-- Привязка userId -> chatId
CREATE TABLE IF NOT EXISTS tg_bindings (
                                           user_id    INTEGER PRIMARY KEY,
                                           chat_id    INTEGER NOT NULL,
                                           bound_at   INTEGER NOT NULL,
                                           updated_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_tg_bindings_chat_id
    ON tg_bindings(chat_id);
