CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE SCHEMA IF NOT EXISTS iam;

CREATE TABLE IF NOT EXISTS iam.organizations (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    provider text NOT NULL CHECK (btrim(provider) <> ''),
    external_id text,
    name text NOT NULL CHECK (btrim(name) <> ''),
    slug text,
    status text NOT NULL CHECK (status IN ('provisioning', 'active', 'suspended', 'disabled')),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CHECK (external_id IS NULL OR btrim(external_id) <> ''),
    CHECK (slug IS NULL OR btrim(slug) <> '')
);

CREATE UNIQUE INDEX IF NOT EXISTS iam_organizations_provider_external_uidx
    ON iam.organizations (provider, external_id)
    WHERE external_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS iam_organizations_slug_uidx
    ON iam.organizations (lower(slug))
    WHERE slug IS NOT NULL;

CREATE TABLE IF NOT EXISTS iam.users (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    provider text NOT NULL CHECK (btrim(provider) <> ''),
    external_id text NOT NULL CHECK (btrim(external_id) <> ''),
    primary_email text NOT NULL CHECK (btrim(primary_email) <> ''),
    email_verified boolean NOT NULL DEFAULT false,
    name text NOT NULL DEFAULT '',
    avatar_url text,
    status text NOT NULL CHECK (status IN ('active', 'disabled')),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (provider, external_id)
);

CREATE INDEX IF NOT EXISTS iam_users_primary_email_idx
    ON iam.users (lower(primary_email));

CREATE TABLE IF NOT EXISTS iam.memberships (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id uuid NOT NULL REFERENCES iam.organizations(id) ON DELETE CASCADE,
    user_id uuid NOT NULL REFERENCES iam.users(id) ON DELETE CASCADE,
    provider text NOT NULL CHECK (btrim(provider) <> ''),
    external_id text,
    role text NOT NULL CHECK (btrim(role) <> ''),
    status text NOT NULL CHECK (status IN ('pending', 'active', 'removed', 'quarantined')),
    joined_at timestamptz,
    removed_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (org_id, user_id),
    CHECK (external_id IS NULL OR btrim(external_id) <> ''),
    CHECK (
        (status = 'removed' AND removed_at IS NOT NULL)
        OR (status <> 'removed' AND removed_at IS NULL)
    )
);

CREATE UNIQUE INDEX IF NOT EXISTS iam_memberships_provider_external_uidx
    ON iam.memberships (provider, external_id)
    WHERE external_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS iam_memberships_org_status_idx
    ON iam.memberships (org_id, status, created_at);

CREATE INDEX IF NOT EXISTS iam_memberships_user_status_idx
    ON iam.memberships (user_id, status, created_at);

CREATE TABLE IF NOT EXISTS iam.invitations (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id uuid NOT NULL REFERENCES iam.organizations(id) ON DELETE CASCADE,
    provider text NOT NULL CHECK (btrim(provider) <> ''),
    external_id text,
    email_normalized text NOT NULL CHECK (btrim(email_normalized) <> ''),
    role text NOT NULL CHECK (btrim(role) <> ''),
    status text NOT NULL CHECK (status IN ('pending', 'accepted', 'revoked', 'expired')),
    expires_at timestamptz NOT NULL,
    accepted_at timestamptz,
    revoked_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CHECK (external_id IS NULL OR btrim(external_id) <> ''),
    CHECK (
        (status = 'accepted' AND accepted_at IS NOT NULL)
        OR (status <> 'accepted' AND accepted_at IS NULL)
    ),
    CHECK (
        (status = 'revoked' AND revoked_at IS NOT NULL)
        OR (status <> 'revoked' AND revoked_at IS NULL)
    )
);

CREATE UNIQUE INDEX IF NOT EXISTS iam_invitations_provider_external_uidx
    ON iam.invitations (provider, external_id)
    WHERE external_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS iam_invitations_pending_email_uidx
    ON iam.invitations (org_id, email_normalized)
    WHERE status = 'pending';

CREATE INDEX IF NOT EXISTS iam_invitations_org_status_idx
    ON iam.invitations (org_id, status, created_at DESC);

CREATE TABLE IF NOT EXISTS iam.webhook_events (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    provider text NOT NULL CHECK (btrim(provider) <> ''),
    external_id text NOT NULL CHECK (btrim(external_id) <> ''),
    event_type text NOT NULL CHECK (btrim(event_type) <> ''),
    payload jsonb NOT NULL,
    occurred_at timestamptz NOT NULL,
    status text NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'processed', 'failed')),
    attempts integer NOT NULL DEFAULT 0 CHECK (attempts >= 0),
    processed_at timestamptz,
    last_error text,
    received_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (provider, external_id),
    CHECK (
        (status = 'processed' AND processed_at IS NOT NULL)
        OR (status <> 'processed' AND processed_at IS NULL)
    )
);

CREATE INDEX IF NOT EXISTS iam_webhook_events_pending_idx
    ON iam.webhook_events (occurred_at, received_at)
    WHERE status IN ('pending', 'failed');
