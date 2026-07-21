CREATE TABLE IF NOT EXISTS platform_idempotency_records (
    scope TEXT NOT NULL,
    idempotency_key TEXT NOT NULL,
    fingerprint TEXT NOT NULL,
    state TEXT NOT NULL,
    claim_token TEXT,
    lease_expires_at TIMESTAMPTZ,
    record_expires_at TIMESTAMPTZ NOT NULL,
    response_status INTEGER,
    response_headers JSONB,
    response_body BYTEA,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (scope, idempotency_key),
    CONSTRAINT platform_idempotency_scope_length
        CHECK (char_length(scope) BETWEEN 1 AND 255),
    CONSTRAINT platform_idempotency_key_length
        CHECK (char_length(idempotency_key) BETWEEN 1 AND 512),
    CONSTRAINT platform_idempotency_fingerprint_length
        CHECK (char_length(fingerprint) BETWEEN 1 AND 128),
    CONSTRAINT platform_idempotency_state
        CHECK (state IN ('processing', 'completed')),
    CONSTRAINT platform_idempotency_response_status
        CHECK (response_status IS NULL OR response_status BETWEEN 100 AND 599),
    CONSTRAINT platform_idempotency_state_shape CHECK (
        (
            state = 'processing'
            AND claim_token IS NOT NULL
            AND lease_expires_at IS NOT NULL
            AND response_status IS NULL
            AND response_headers IS NULL
            AND response_body IS NULL
        )
        OR
        (
            state = 'completed'
            AND claim_token IS NULL
            AND lease_expires_at IS NULL
            AND response_status IS NOT NULL
            AND response_headers IS NOT NULL
            AND response_body IS NOT NULL
        )
    )
);

CREATE INDEX IF NOT EXISTS platform_idempotency_records_expiry_idx
    ON platform_idempotency_records (record_expires_at);
