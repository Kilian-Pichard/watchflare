-- +goose Up
-- Migration 004: Host attributes
-- Environment detection, reactivation tracking, agent version.

ALTER TABLE hosts ADD COLUMN IF NOT EXISTS environment_type  VARCHAR(50);
ALTER TABLE hosts ADD COLUMN IF NOT EXISTS hypervisor        VARCHAR(50);
ALTER TABLE hosts ADD COLUMN IF NOT EXISTS container_runtime VARCHAR(50);

COMMENT ON COLUMN hosts.environment_type  IS 'Environment type: physical, physical_with_containers, vm, vm_with_containers, container';
COMMENT ON COLUMN hosts.hypervisor        IS 'Hypervisor type if running in VM: kvm, vmware, virtualbox, hyperv, xen, unknown (NULL if physical)';
COMMENT ON COLUMN hosts.container_runtime IS 'Container runtime if running in container: docker, lxc, podman, kubernetes, unknown (NULL if not in container)';

CREATE INDEX IF NOT EXISTS idx_hosts_environment_type ON hosts(environment_type);

ALTER TABLE hosts ADD COLUMN IF NOT EXISTS reactivated_at TIMESTAMPTZ;
CREATE INDEX IF NOT EXISTS idx_hosts_reactivated_at ON hosts(reactivated_at) WHERE reactivated_at IS NOT NULL;

ALTER TABLE hosts ADD COLUMN IF NOT EXISTS agent_version VARCHAR(50);

-- +goose Down
ALTER TABLE hosts DROP COLUMN IF EXISTS agent_version;
DROP INDEX IF EXISTS idx_hosts_reactivated_at;
ALTER TABLE hosts DROP COLUMN IF EXISTS reactivated_at;
DROP INDEX IF EXISTS idx_hosts_environment_type;
ALTER TABLE hosts DROP COLUMN IF EXISTS container_runtime;
ALTER TABLE hosts DROP COLUMN IF EXISTS hypervisor;
ALTER TABLE hosts DROP COLUMN IF EXISTS environment_type;
