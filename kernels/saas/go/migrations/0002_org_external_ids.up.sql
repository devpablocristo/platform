ALTER TABLE orgs
    ADD COLUMN IF NOT EXISTS external_id text;

CREATE UNIQUE INDEX IF NOT EXISTS idx_orgs_external_id
    ON orgs(external_id)
    WHERE external_id IS NOT NULL;
