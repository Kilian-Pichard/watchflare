-- +goose Up
-- Add update availability columns to packages table
ALTER TABLE packages
  ADD COLUMN IF NOT EXISTS available_version VARCHAR(100),
  ADD COLUMN IF NOT EXISTS has_security_update BOOLEAN NOT NULL DEFAULT false;

-- Index for efficient filtering of outdated / security packages
CREATE INDEX IF NOT EXISTS idx_packages_has_update
  ON packages (host_id, available_version)
  WHERE available_version IS NOT NULL AND available_version != '';

CREATE INDEX IF NOT EXISTS idx_packages_security_update
  ON packages (host_id, has_security_update)
  WHERE has_security_update = true;

-- +goose Down
DROP INDEX IF EXISTS idx_packages_security_update;
DROP INDEX IF EXISTS idx_packages_has_update;
ALTER TABLE packages DROP COLUMN IF EXISTS has_security_update;
ALTER TABLE packages DROP COLUMN IF EXISTS available_version;
