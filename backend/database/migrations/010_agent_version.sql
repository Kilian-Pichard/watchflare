-- +goose Up
ALTER TABLE servers ADD COLUMN IF NOT EXISTS agent_version VARCHAR(50);

-- +goose Down
ALTER TABLE servers DROP COLUMN IF EXISTS agent_version;
