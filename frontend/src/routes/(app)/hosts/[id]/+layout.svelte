<script lang="ts">
    import { onMount, onDestroy, setContext } from 'svelte';
    import { goto } from '$app/navigation';
    import { page } from '$app/stores';
    import * as api from '$lib/api.js';
    import { sseStore } from '$lib/stores/sse';
    import { handleSSEReactivation, logger } from '$lib/utils';
    import type { Host, Metric, SSEEvent, Package, PackageStats, PackageCollection, HostIncident, IncidentStatusFilter, TimeRange, ContainerMetric } from '$lib/types';
    import HostDetailHeader from '$lib/components/host/HostDetailHeader.svelte';
    import HostAlerts from '$lib/components/host/HostAlerts.svelte';
    import HostAlertRulesDrawer from '$lib/components/host/HostAlertRulesDrawer.svelte';
    import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';
    import Modal from '$lib/components/Modal.svelte';
    import InstallInstructions from '$lib/components/InstallInstructions.svelte';

    const { children } = $props();

    const hostId = $derived($page.params.id);
    const currentPath = $derived($page.url.pathname);

    let host: Host | null = $state(null);
    let loading = $state(true);
    let error = $state('');
    let clockDesync = $state(false);
    let latestAgentVersion: string | null = $state(null);
    let latestMetric: Metric | null = $state(null);
    let hasActiveAlertRules = $state(false);

    // Tab data caches — persist between tab switches for the duration of the host detail session
    let overviewCache: { metrics: Metric[]; containerMetrics: ContainerMetric[]; timeRange: TimeRange } | null = $state(null);
    let packagesCache: { packages: Package[]; stats: PackageStats | null; collections: PackageCollection[]; totalCount: number; offset: number; searchTerm: string; allManagerKeys: string[]; selectedManagers: string[] } | null = $state(null);
    let incidentsCache: { incidents: HostIncident[]; totalCount: number; offset: number; statusFilter: IncidentStatusFilter } | null = $state(null);

    // Modals
    let showDeleteConfirm = $state(false);
    let showRegenerateConfirm = $state(false);
    let showChangeIP = $state(false);
    let showRename = $state(false);
    let showAlertRules = $state(false);
    let newHostName = $state('');
    let newIP = $state('');
    let regeneratedToken = $state('');
    let copiedToken = $state(false);
    let backendHost = $state('');

    setContext('hostDetail', {
        get host() { return host; },
        get loading() { return loading; },
        get latestMetric() { return latestMetric; },
        setLatestMetric: (m: Metric | null) => { latestMetric = m; },
        get overviewCache() { return overviewCache; },
        setOverviewCache: (data: typeof overviewCache) => { overviewCache = data; },
        get packagesCache() { return packagesCache; },
        setPackagesCache: (data: typeof packagesCache) => { packagesCache = data; },
        get incidentsCache() { return incidentsCache; },
        setIncidentsCache: (data: typeof incidentsCache) => { incidentsCache = data; },
    });

    const showIPMismatchWarning = $derived(
        !!(host &&
            host.configured_ip &&
            host.ip_address_v4 &&
            host.configured_ip !== host.ip_address_v4 &&
            !host.ignore_ip_mismatch),
    );

    function isActiveTab(tab: 'overview' | 'packages' | 'incidents'): boolean {
        const base = `/hosts/${hostId}`;
        if (tab === 'overview') return currentPath === base;
        return currentPath.startsWith(`${base}/${tab}`);
    }

    let sseUnsubscribe: (() => void) | null = null;

    onMount(() => {
        sseUnsubscribe = sseStore.connect(handleSSEMessage);
        loadHost();
    });

    onDestroy(() => {
        if (sseUnsubscribe) sseUnsubscribe();
    });

    function handleSSEMessage(event: SSEEvent) {
        handleSSEReactivation(event);
        if (event.type === 'host_update') {
            const update = event.data;
            if (host && update.id === host.id) {
                host = {
                    ...host,
                    status: update.status,
                    ip_address_v4: update.ip_address_v4,
                    ip_address_v6: update.ip_address_v6,
                    configured_ip: update.configured_ip,
                    ignore_ip_mismatch: update.ignore_ip_mismatch,
                    last_seen: update.last_seen,
                };
                clockDesync = update.clock_desync || false;
            }
        }
        if (event.type === 'metrics_update') {
            const metric = event.data;
            if (host && metric.host_id === host.id) {
                latestMetric = metric;
            }
        }
    }

    async function loadHost() {
        try {
            const [response] = await Promise.all([
                api.getHost(hostId),
                latestAgentVersion === null
                    ? api.getLatestAgentVersion().then(r => { latestAgentVersion = r.latest_version || null; }).catch(() => {})
                    : Promise.resolve(),
            ]);
            host = response.host;
            clockDesync = response.clock_desync || false;
        } catch (err) {
            error = err instanceof Error ? err.message : 'Failed to load host';
        } finally {
            loading = false;
        }
        // Load alert rules for bell indicator (non-critical)
        try {
            const rulesData = await api.getHostAlertRules(hostId);
            hasActiveAlertRules = rulesData.rules.some(r => r.enabled);
        } catch {
            // non-critical
        }
    }

    async function handleDelete() {
        try {
            await api.deleteHost(hostId);
            goto('/hosts');
        } catch (err) {
            error = err instanceof Error ? err.message : 'Failed to delete host';
            showDeleteConfirm = false;
        }
    }

    async function handleRegenerateToken() {
        try {
            const response = await api.regenerateToken(hostId);
            regeneratedToken = response.token;
            backendHost = window.location.host;
            showRegenerateConfirm = false;
            await loadHost();
        } catch (err) {
            error = err instanceof Error ? err.message : 'Failed to regenerate token';
            showRegenerateConfirm = false;
        }
    }

    async function handleRename() {
        try {
            await api.renameHost(hostId, newHostName);
            showRename = false;
            newHostName = '';
            await loadHost();
        } catch (err) {
            error = err instanceof Error ? err.message : 'Failed to rename host';
        }
    }

    async function handleChangeIP() {
        try {
            await api.updateConfiguredIP(hostId, newIP);
            showChangeIP = false;
            newIP = '';
            await loadHost();
        } catch (err) {
            error = err instanceof Error ? err.message : 'Failed to update IP';
        }
    }

    async function handleUpdateIP() {
        if (!host) return;
        try {
            await api.updateConfiguredIP(host.id, host.ip_address_v4);
            await loadHost();
        } catch (err) {
            error = err instanceof Error ? err.message : 'Failed to update IP';
        }
    }

    async function handleIgnoreIP() {
        if (!host) return;
        try {
            await api.ignoreIPMismatch(host.id);
            await loadHost();
        } catch (err) {
            error = err instanceof Error ? err.message : 'Failed to ignore IP mismatch';
        }
    }

    async function handleDismissReactivation() {
        if (!host) return;
        try {
            await api.dismissReactivation(host.id);
            await loadHost();
        } catch (err) {
            error = err instanceof Error ? err.message : 'Failed to dismiss reactivation';
        }
    }

    async function handlePause() {
        if (!host) return;
        const previousStatus = host.status;
        try {
            await api.pauseHost(host.id);
            host = { ...host, status: 'paused' };
        } catch (err) {
            host = { ...host, status: previousStatus };
            error = err instanceof Error ? err.message : 'Failed to pause host';
        }
    }

    async function handleResume() {
        if (!host) return;
        const previousStatus = host.status;
        try {
            await api.resumeHost(host.id);
            host = { ...host, status: 'online' };
        } catch (err) {
            host = { ...host, status: previousStatus };
            error = err instanceof Error ? err.message : 'Failed to resume host';
        }
    }

    let copyError = $state(false);

    async function handleCopy(text: string) {
        try {
            await navigator.clipboard.writeText(text);
        } catch {
            copyError = true;
            setTimeout(() => copyError = false, 2000);
        }
    }

    function closeChangeIPModal() {
        showChangeIP = false;
        newIP = '';
    }
</script>

<svelte:head>
    <title>{host?.name || 'Host'} - Watchflare</title>
</svelte:head>

{#if loading}
    <div class="flex items-center justify-center py-20">
        <p class="text-muted-foreground">Loading host details...</p>
    </div>
{:else if error}
    <div class="rounded-lg border border-destructive bg-destructive/10 p-4">
        <p class="text-sm text-destructive">{error}</p>
    </div>
{:else if host}
    <HostDetailHeader
        {host}
        metric={latestMetric}
        {latestAgentVersion}
        {hasActiveAlertRules}
        onDelete={() => (showDeleteConfirm = true)}
        onRegenerateToken={() => (showRegenerateConfirm = true)}
        onChangeIP={() => (showChangeIP = true)}
        onRename={() => { newHostName = host?.name || ''; showRename = true; }}
        onPause={handlePause}
        onResume={handleResume}
        onAlertRules={() => { showAlertRules = true; }}
    />

    {#if regeneratedToken}
        <div class="mb-6 rounded-lg border border-warning bg-warning/10 p-4 space-y-3">
            <div class="flex items-center justify-between gap-4 flex-wrap">
                <p class="text-sm font-medium text-warning">This token is valid for 24 hours and will not be displayed again. Make sure to copy it or use it now.</p>
                <div class="flex items-center gap-2 shrink-0">
                    <button
                        onclick={() => { handleCopy(regeneratedToken); copiedToken = true; setTimeout(() => copiedToken = false, 2000); }}
                        disabled={copiedToken}
                        class="rounded-lg border bg-background px-3 py-1.5 text-sm font-medium transition-colors hover:bg-muted disabled:opacity-60 {copyError ? 'text-destructive border-destructive/40' : 'text-foreground'}"
                    >
                        {copiedToken ? 'Copied!' : copyError ? 'Copy failed' : 'Copy Token'}
                    </button>
                    <button
                        onclick={() => { regeneratedToken = ''; }}
                        class="rounded-lg border bg-background px-3 py-1.5 text-sm font-medium text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
                    >
                        Dismiss
                    </button>
                </div>
            </div>
            <!-- svelte-ignore a11y_click_events_have_key_events -->
            <!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
            <input
                readonly
                value={regeneratedToken}
                onclick={(e) => (e.currentTarget as HTMLInputElement).select()}
                class="w-full font-mono text-xs bg-background border rounded-lg px-3 py-2 text-foreground select-all cursor-text focus:outline-none focus-visible:ring-2 focus-visible:ring-primary"
            />
        </div>
        <InstallInstructions {host} token={regeneratedToken} {backendHost} />
    {/if}

    <HostAlerts
        {host}
        {showIPMismatchWarning}
        {clockDesync}
        onUpdateIP={handleUpdateIP}
        onIgnoreIP={handleIgnoreIP}
        onDismissReactivation={handleDismissReactivation}
    />

    <!-- Tab Navigation -->
    <div class="mb-6 flex gap-1 border-b">
        <a
            href="/hosts/{hostId}"
            class="px-4 py-2.5 text-sm font-medium transition-colors border-b-2 -mb-px {isActiveTab('overview')
                ? 'border-primary text-foreground'
                : 'border-transparent text-muted-foreground hover:text-foreground'}"
        >
            Overview
        </a>
        <a
            href="/hosts/{hostId}/packages"
            class="px-4 py-2.5 text-sm font-medium transition-colors border-b-2 -mb-px {isActiveTab('packages')
                ? 'border-primary text-foreground'
                : 'border-transparent text-muted-foreground hover:text-foreground'}"
        >
            Packages
        </a>
        <a
            href="/hosts/{hostId}/incidents"
            class="px-4 py-2.5 text-sm font-medium transition-colors border-b-2 -mb-px {isActiveTab('incidents')
                ? 'border-primary text-foreground'
                : 'border-transparent text-muted-foreground hover:text-foreground'}"
        >
            Incidents
        </a>
    </div>

    {@render children()}

    <HostAlertRulesDrawer
        hostId={hostId}
        open={showAlertRules}
        onClose={() => { showAlertRules = false; }}
        onSave={(hasActive) => { hasActiveAlertRules = hasActive; }}
    />
{/if}

<!-- Modals -->
<ConfirmDialog
    open={showDeleteConfirm}
    title="Confirm Delete"
    onConfirm={handleDelete}
    onClose={() => (showDeleteConfirm = false)}
    confirmLabel="Delete Host"
    confirmVariant="destructive"
>
    <p class="text-sm text-muted-foreground mb-4">Are you sure you want to delete "{host?.name}"?</p>
    <p class="text-sm font-medium text-destructive">This action cannot be undone.</p>
</ConfirmDialog>

<ConfirmDialog
    open={showRegenerateConfirm}
    title="Regenerate Token"
    onConfirm={handleRegenerateToken}
    onClose={() => (showRegenerateConfirm = false)}
    confirmLabel="Regenerate"
>
    <p class="text-sm text-muted-foreground">
        This will generate a new registration token and set the host to pending until the agent
        re-registers. Use the new token to run <code class="font-mono">watchflare-agent register</code>
        on the host.
    </p>
</ConfirmDialog>

<Modal open={showRename} onClose={() => { showRename = false; newHostName = ''; }}>
    <h3 class="text-lg font-semibold text-foreground mb-3">Rename Host</h3>
    <div class="mb-4">
        <label for="newname" class="block text-sm font-medium text-foreground mb-2">New Name</label>
        <input
            id="newname"
            type="text"
            bind:value={newHostName}
            placeholder="e.g., production-web-01"
            class="w-full rounded-lg border bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus-visible:ring-2 focus-visible:ring-primary"
        />
    </div>
    <div class="flex gap-3 justify-end">
        <button
            onclick={() => { showRename = false; newHostName = ''; }}
            class="rounded-lg border bg-background px-4 py-2 text-sm font-medium text-foreground transition-colors hover:bg-muted"
        >
            Cancel
        </button>
        <button
            onclick={handleRename}
            disabled={newHostName.length < 2}
            class="rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90 disabled:opacity-50"
        >
            Rename
        </button>
    </div>
</Modal>

<Modal open={showChangeIP} onClose={closeChangeIPModal}>
    <h3 class="text-lg font-semibold text-foreground mb-3">Change Configured IP</h3>
    <div class="mb-4">
        <label for="newip" class="block text-sm font-medium text-foreground mb-2">New IP Address</label>
        <input
            id="newip"
            type="text"
            bind:value={newIP}
            placeholder="e.g., 192.168.1.100"
            class="w-full rounded-lg border bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus-visible:ring-2 focus-visible:ring-primary"
        />
    </div>
    <div class="flex gap-3 justify-end">
        <button
            onclick={closeChangeIPModal}
            class="rounded-lg border bg-background px-4 py-2 text-sm font-medium text-foreground transition-colors hover:bg-muted"
        >
            Cancel
        </button>
        <button
            onclick={handleChangeIP}
            disabled={!newIP.trim()}
            class="rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90 disabled:opacity-50"
        >
            Update IP
        </button>
    </div>
</Modal>
