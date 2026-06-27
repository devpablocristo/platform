-- Org-first IAM model. Historical tenant_* table names are renamed in place.

DO $$
BEGIN
    IF to_regclass('public.org_settings') IS NULL
       AND to_regclass('public.tenant_settings') IS NOT NULL THEN
        ALTER TABLE tenant_settings RENAME TO org_settings;
    END IF;

    IF to_regclass('public.org_invitations') IS NULL
       AND to_regclass('public.tenant_invitations') IS NOT NULL THEN
        ALTER TABLE tenant_invitations RENAME TO org_invitations;
    END IF;
END $$;

ALTER INDEX IF EXISTS idx_tenant_settings_stripe_customer RENAME TO idx_org_settings_stripe_customer;
ALTER INDEX IF EXISTS idx_tenant_settings_past_due_since RENAME TO idx_org_settings_past_due_since;

CREATE TABLE IF NOT EXISTS org_settings (
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

CREATE INDEX IF NOT EXISTS idx_org_settings_stripe_customer
    ON org_settings(stripe_customer_id) WHERE stripe_customer_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_org_settings_past_due_since
    ON org_settings(past_due_since) WHERE billing_status = 'past_due';

CREATE TABLE IF NOT EXISTS org_invitations (
    id                    uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id                uuid NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
    email_normalized      text NOT NULL,
    role                  text NOT NULL DEFAULT 'member',
    status                text NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'accepted', 'revoked', 'expired')),
    token_hash            text NOT NULL UNIQUE,
    provider              text NOT NULL DEFAULT 'clerk',
    provider_invitation_id text,
    invited_by_user_id    uuid REFERENCES users(id) ON DELETE SET NULL,
    accepted_by_user_id   uuid REFERENCES users(id) ON DELETE SET NULL,
    expires_at            timestamptz NOT NULL,
    accepted_at           timestamptz,
    revoked_at            timestamptz,
    created_at            timestamptz NOT NULL DEFAULT now(),
    updated_at            timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_org_invitations_org_status
    ON org_invitations(org_id, status, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_org_invitations_email
    ON org_invitations(email_normalized);
