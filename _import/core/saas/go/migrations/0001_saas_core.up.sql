-- saas-core: consolidated SaaS infrastructure schema
-- Consolidated tenant and identity bootstrap schema.

CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS orgs (
    id         uuid PRIMARY KEY,
    name       text NOT NULL UNIQUE,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS org_api_keys (
    id           uuid PRIMARY KEY,
    org_id       uuid NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
    api_key_hash text NOT NULL UNIQUE,
    name         text NOT NULL,
    created_at   timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS org_api_key_scopes (
    id         uuid PRIMARY KEY,
    api_key_id uuid NOT NULL REFERENCES org_api_keys(id) ON DELETE CASCADE,
    scope      text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_org_api_key_scopes_api_key_id
    ON org_api_key_scopes(api_key_id);

CREATE TABLE IF NOT EXISTS users (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    external_id text NOT NULL UNIQUE,
    email       text NOT NULL UNIQUE,
    name        text NOT NULL DEFAULT '',
    avatar_url  text,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_users_external_id ON users(external_id);

CREATE TABLE IF NOT EXISTS org_members (
    id        uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id    uuid NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
    user_id   uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role      text NOT NULL DEFAULT 'member' CHECK (role IN ('admin', 'member', 'secops')),
    joined_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE(org_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_org_members_org_id ON org_members(org_id);
CREATE INDEX IF NOT EXISTS idx_org_members_user_id ON org_members(user_id);

CREATE TABLE IF NOT EXISTS tenant_settings (
    org_id                  uuid PRIMARY KEY REFERENCES orgs(id) ON DELETE CASCADE,
    plan_code               text NOT NULL DEFAULT 'starter',
    hard_limits_json        jsonb NOT NULL DEFAULT '{}'::jsonb,
    stripe_customer_id      text UNIQUE,
    stripe_subscription_id  text UNIQUE,
    billing_status          text NOT NULL DEFAULT 'trialing'
        CHECK (billing_status IN ('trialing', 'active', 'past_due', 'canceled', 'unpaid')),
    past_due_since          timestamptz,
    status                  text NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'suspended', 'deleted')),
    deleted_at              timestamptz,
    updated_by              text,
    updated_at              timestamptz NOT NULL DEFAULT now(),
    created_at              timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_tenant_settings_stripe_customer
    ON tenant_settings(stripe_customer_id) WHERE stripe_customer_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_tenant_settings_past_due_since
    ON tenant_settings(past_due_since) WHERE billing_status = 'past_due';

CREATE TABLE IF NOT EXISTS org_usage_counters (
    org_id     uuid NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
    period     date NOT NULL,
    counter    text NOT NULL,
    value      bigint NOT NULL DEFAULT 0,
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (org_id, period, counter)
);

CREATE TABLE IF NOT EXISTS saas_usage_event_dedup (
    event_id   text PRIMARY KEY,
    org_id     uuid NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
    counter    text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS admin_activity_events (
    id            uuid PRIMARY KEY,
    org_id        uuid NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
    actor         text,
    action        text NOT NULL,
    resource_type text NOT NULL,
    resource_id   text,
    payload_json  jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at    timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_admin_activity_events_org_created
    ON admin_activity_events(org_id, created_at DESC);

CREATE TABLE IF NOT EXISTS notification_preferences (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id           uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    notification_type text NOT NULL,
    channel           text NOT NULL DEFAULT 'email',
    enabled           boolean NOT NULL DEFAULT true,
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now(),
    UNIQUE(user_id, notification_type, channel)
);

CREATE INDEX IF NOT EXISTS idx_notification_prefs_user
    ON notification_preferences(user_id);

CREATE TABLE IF NOT EXISTS notification_log (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id            uuid NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
    user_id           uuid REFERENCES users(id) ON DELETE SET NULL,
    notification_type text NOT NULL,
    channel           text NOT NULL DEFAULT 'email',
    recipient         text NOT NULL,
    subject           text NOT NULL,
    status            text NOT NULL DEFAULT 'sent',
    dedup_key         text,
    error_message     text,
    created_at        timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_notification_log_org_created
    ON notification_log(org_id, created_at DESC);

CREATE UNIQUE INDEX IF NOT EXISTS idx_notification_log_dedup_key
    ON notification_log(dedup_key);

CREATE TABLE IF NOT EXISTS in_app_notifications (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id     uuid NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
    actor_id   text NOT NULL DEFAULT '',
    type       text NOT NULL,
    title      text NOT NULL,
    body       text NOT NULL DEFAULT '',
    read_at    timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_inapp_notif_org_unread
    ON in_app_notifications(org_id, read_at) WHERE read_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_inapp_notif_actor_created
    ON in_app_notifications(actor_id, created_at DESC);

CREATE TABLE IF NOT EXISTS protected_resources (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id        uuid NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
    name          text NOT NULL,
    resource_type text NOT NULL,
    match_value   text NOT NULL,
    match_mode    text NOT NULL DEFAULT 'exact',
    environment   text NOT NULL DEFAULT '*',
    reason        text NOT NULL DEFAULT '',
    enabled       boolean NOT NULL DEFAULT true,
    created_by    text,
    updated_by    text,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_protected_resources_org_created
    ON protected_resources(org_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_protected_resources_org_enabled
    ON protected_resources(org_id, enabled);

CREATE TABLE IF NOT EXISTS restore_evidence (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id          uuid NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
    environment     text NOT NULL DEFAULT 'prod',
    system          text NOT NULL,
    status          text NOT NULL,
    snapshot_id     text NOT NULL DEFAULT '',
    restore_target  text NOT NULL DEFAULT '',
    started_at      timestamptz,
    completed_at    timestamptz,
    source          text NOT NULL DEFAULT '',
    artifact_sha256 text NOT NULL DEFAULT '',
    summary_json    jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at      timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_restore_evidence_org_created
    ON restore_evidence(org_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_restore_evidence_org_system_env
    ON restore_evidence(org_id, system, environment, created_at DESC);
