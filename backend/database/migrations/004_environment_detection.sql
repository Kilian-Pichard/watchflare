-- +goose Up
-- Migration 004: Environment Detection
-- Adds environment_type, hypervisor, container_runtime fields to servers table

-- =====================================================
-- Add Environment Detection Fields to Servers Table
-- =====================================================

-- Add environment_type column
ALTER TABLE servers
ADD COLUMN IF NOT EXISTS environment_type VARCHAR(50);

-- Add hypervisor column
ALTER TABLE servers
ADD COLUMN IF NOT EXISTS hypervisor VARCHAR(50);

-- Add container_runtime column
ALTER TABLE servers
ADD COLUMN IF NOT EXISTS container_runtime VARCHAR(50);

-- Create index for environment type queries (useful for filtering by environment)
CREATE INDEX IF NOT EXISTS idx_servers_environment_type ON servers(environment_type);

-- Comments for documentation
COMMENT ON COLUMN servers.environment_type IS 'Environment type: physical, physical_with_containers, vm, vm_with_containers, container';
COMMENT ON COLUMN servers.hypervisor IS 'Hypervisor type if running in VM: kvm, vmware, virtualbox, hyperv, xen, unknown (NULL if physical)';
COMMENT ON COLUMN servers.container_runtime IS 'Container runtime if running in container: docker, lxc, podman, kubernetes, unknown (NULL if not in container)';

-- +goose Down
DROP INDEX IF EXISTS idx_servers_environment_type;
ALTER TABLE servers DROP COLUMN IF EXISTS container_runtime;
ALTER TABLE servers DROP COLUMN IF EXISTS hypervisor;
ALTER TABLE servers DROP COLUMN IF EXISTS environment_type;
