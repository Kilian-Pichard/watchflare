-- Migration 005: Add reactivation tracking
-- Adds reactivated_at timestamp to track when agents are reactivated via UUID reuse

-- Add reactivated_at column to servers table
ALTER TABLE servers ADD COLUMN IF NOT EXISTS reactivated_at TIMESTAMP;

-- Create index for querying reactivated agents
CREATE INDEX IF NOT EXISTS idx_servers_reactivated_at ON servers(reactivated_at) WHERE reactivated_at IS NOT NULL;
