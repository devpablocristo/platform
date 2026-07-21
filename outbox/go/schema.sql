CREATE TABLE IF NOT EXISTS "platform_outbox_messages" (
    id TEXT PRIMARY KEY,
    idempotency_key TEXT NOT NULL UNIQUE,
    topic TEXT NOT NULL,
    payload BYTEA NOT NULL,
    headers JSONB NOT NULL DEFAULT '{}'::jsonb,
    available_at TIMESTAMPTZ NOT NULL,
    attempts INTEGER NOT NULL DEFAULT 0,
    max_attempts INTEGER NOT NULL,
    last_attempt_at TIMESTAMPTZ,
    lease_token TEXT,
    leased_at TIMESTAMPTZ,
    lease_expires_at TIMESTAMPTZ,
    last_error TEXT,
    last_error_at TIMESTAMPTZ,
    published_at TIMESTAMPTZ,
    failed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT platform_outbox_attempts_valid CHECK (
        attempts >= 0 AND max_attempts > 0 AND attempts <= max_attempts
    ),
    CONSTRAINT platform_outbox_headers_object CHECK (jsonb_typeof(headers) = 'object'),
    CONSTRAINT platform_outbox_lease_consistent CHECK (
        (lease_token IS NULL AND leased_at IS NULL AND lease_expires_at IS NULL)
        OR
        (lease_token IS NOT NULL AND leased_at IS NOT NULL AND lease_expires_at IS NOT NULL)
    ),
    CONSTRAINT platform_outbox_terminal_state CHECK (published_at IS NULL OR failed_at IS NULL)
);

CREATE INDEX IF NOT EXISTS "platform_outbox_messages_ready_idx"
    ON "platform_outbox_messages" (available_at, created_at, id)
    WHERE published_at IS NULL AND failed_at IS NULL;
