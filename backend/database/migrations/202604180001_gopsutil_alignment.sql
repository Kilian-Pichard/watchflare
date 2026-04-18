-- +goose Up
-- Rename host columns to align with gopsutil naming conventions.
-- Each block is conditional because GORM AutoMigrate (which runs before Goose)
-- may have already added the new column, leaving both old and new columns present.

-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='hosts' AND column_name='architecture')
       AND NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='hosts' AND column_name='kernel_arch') THEN
        ALTER TABLE hosts RENAME COLUMN architecture TO kernel_arch;
    ELSIF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='hosts' AND column_name='architecture') THEN
        ALTER TABLE hosts DROP COLUMN architecture;
    END IF;

    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='hosts' AND column_name='kernel')
       AND NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='hosts' AND column_name='kernel_version') THEN
        ALTER TABLE hosts RENAME COLUMN kernel TO kernel_version;
    ELSIF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='hosts' AND column_name='kernel') THEN
        ALTER TABLE hosts DROP COLUMN kernel;
    END IF;

    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='hosts' AND column_name='hypervisor')
       AND NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='hosts' AND column_name='virtualization_system') THEN
        ALTER TABLE hosts RENAME COLUMN hypervisor TO virtualization_system;
    ELSIF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='hosts' AND column_name='hypervisor') THEN
        ALTER TABLE hosts DROP COLUMN hypervisor;
    END IF;
END $$;
-- +goose StatementEnd

-- Add new gopsutil host fields
ALTER TABLE hosts ADD COLUMN IF NOT EXISTS os VARCHAR(50);
ALTER TABLE hosts ADD COLUMN IF NOT EXISTS virtualization_role VARCHAR(20);
ALTER TABLE hosts ADD COLUMN IF NOT EXISTS host_id VARCHAR(100);
ALTER TABLE hosts ADD COLUMN IF NOT EXISTS cpu_model_name VARCHAR(200);
ALTER TABLE hosts ADD COLUMN IF NOT EXISTS cpu_physical_count INTEGER;
ALTER TABLE hosts ADD COLUMN IF NOT EXISTS cpu_logical_count INTEGER;
ALTER TABLE hosts ADD COLUMN IF NOT EXISTS cpu_mhz DOUBLE PRECISION;

-- Add swap and process metrics
ALTER TABLE metrics ADD COLUMN IF NOT EXISTS swap_total_bytes BIGINT NOT NULL DEFAULT 0;
ALTER TABLE metrics ADD COLUMN IF NOT EXISTS swap_used_bytes BIGINT NOT NULL DEFAULT 0;
ALTER TABLE metrics ADD COLUMN IF NOT EXISTS processes_count BIGINT NOT NULL DEFAULT 0;

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='hosts' AND column_name='kernel_arch')
       AND NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='hosts' AND column_name='architecture') THEN
        ALTER TABLE hosts RENAME COLUMN kernel_arch TO architecture;
    END IF;

    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='hosts' AND column_name='kernel_version')
       AND NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='hosts' AND column_name='kernel') THEN
        ALTER TABLE hosts RENAME COLUMN kernel_version TO kernel;
    END IF;

    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='hosts' AND column_name='virtualization_system')
       AND NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='hosts' AND column_name='hypervisor') THEN
        ALTER TABLE hosts RENAME COLUMN virtualization_system TO hypervisor;
    END IF;
END $$;
-- +goose StatementEnd

ALTER TABLE hosts DROP COLUMN IF EXISTS os;
ALTER TABLE hosts DROP COLUMN IF EXISTS virtualization_role;
ALTER TABLE hosts DROP COLUMN IF EXISTS host_id;
ALTER TABLE hosts DROP COLUMN IF EXISTS cpu_model_name;
ALTER TABLE hosts DROP COLUMN IF EXISTS cpu_physical_count;
ALTER TABLE hosts DROP COLUMN IF EXISTS cpu_logical_count;
ALTER TABLE hosts DROP COLUMN IF EXISTS cpu_mhz;

ALTER TABLE metrics DROP COLUMN IF EXISTS swap_total_bytes;
ALTER TABLE metrics DROP COLUMN IF EXISTS swap_used_bytes;
ALTER TABLE metrics DROP COLUMN IF EXISTS processes_count;
