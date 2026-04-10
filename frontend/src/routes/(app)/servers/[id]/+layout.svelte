<script lang="ts">
    import { onMount, onDestroy, setContext } from 'svelte';
    import { goto } from '$app/navigation';
    import { page } from '$app/stores';
    import * as api from '$lib/api.js';
    import { sseStore } from '$lib/stores/sse';
    import { handleSSEReactivation, logger } from '$lib/utils';
    import type { Server, Metric, SSEEvent, Package, PackageStats, PackageCollection, ServerIncident, IncidentStatusFilter, TimeRange, ContainerMetric } from '$lib/types';
    import ServerDetailHeader from '$lib/components/server/ServerDetailHeader.svelte';
    import ServerAlerts from '$lib/components/server/ServerAlerts.svelte';
    import ServerAlertRulesDrawer from '$lib/components/server/ServerAlertRulesDrawer.svelte';
    import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';
    import Modal from '$lib/components/Modal.svelte';
    import InstallInstructions from '$lib/components/InstallInstructions.svelte';

    const { children } = $props();

    const serverId = $derived($page.params.id);
    const currentPath = $derived($page.url.pathname);

    let server: Server | null = $state(null);
    let loading = $state(true);
    let error = $state('');
    let clockDesync = $state(false);
    let latestAgentVersion: string | null = $state(null);
    let latestMetric: Metric | null = $state(null);
    let hasActiveAlertRules = $state(false);

    // Tab data caches — persist between tab switches for the duration of the server detail session
    let overviewCache: { metrics: Metric[]; containerMetrics: ContainerMetric[]; timeRange: TimeRange } | null = $state(null);
    let packagesCache: { packages: Package[]; stats: PackageStats | null; collections: PackageCollection[]; totalCount: number; offset: number; searchTerm: string; allManagerKeys: string[]; selectedManagers: string[] } | null = $state(null);
    let incidentsCache: { incidents: ServerIncident[]; totalCount: number; offset: number; statusFilter: IncidentStatusFilter } | null = $state(null);

    // Modals
    let showDeleteConfirm = $state(false);
    let showRegenerateConfirm = $state(false);
    let showChangeIP = $state(false);
    let showRename = $state(false);
    let showAlertRules = $state(false);
    let newServerName = $state('');
    let newIP = $state('');
    let regeneratedToken = $state('');
    let copiedToken = $state(false);
    let backendHost = $state('');

    setContext('serverDetail', {
        get server() { return server; },
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
        !!(server &&
            server.configured_ip &&
            server.ip_address_v4 &&
            server.configured_ip !== server.ip_address_v4 &&
            !server.ignore_ip_mismatch),
    );

    function isActiveTab(tab: 'overview' | 'packages' | 'incidents'): boolean {
        const base = `/servers/${serverId}`;
        if (tab === 'overview') return currentPath === base;
        return currentPath.startsWith(`${base}/${tab}`);
    }

    let sseUnsubscribe: (() => void) | null = null;

    onMount(() => {
        sseUnsubscribe = sseStore.connect(handleSSEMessage);
        loadServer();
    });

    onDestroy(() => {
        if (sseUnsubscribe) sseUnsubscribe();
    });

    function handleSSEMessage(event: SSEEvent) {
        handleSSEReactivation(event);
        if (event.type === 'server_update') {
            const update = event.data;
            if (server && update.id === server.id) {
                server = {
                    ...server,
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
            if (server && metric.server_id === server.id) {
                latestMetric = metric;
            }
        }
    }

    async function loadServer() {
        try {
            const [response] = await Promise.all([
                api.getServer(serverId),
                latestAgentVersion === null
                    ? api.getLatestAgentVersion().then(r => { latestAgentVersion = r.latest_version || null; }).catch(() => {})
                    : Promise.resolve(),
            ]);
            server = response.server;
            clockDesync = response.clock_desync || false;
        } catch (err) {
            error = err instanceof Error ? err.message : 'Failed to load server';
        } finally {
            loading = false;
        }
        // Load alert rules for bell indicator (non-critical)
        try {
            const rulesData = await api.getServerAlertRules(serverId);
            hasActiveAlertRules = rulesData.rules.some(r => r.enabled);
        } catch {
            // non-critical
        }
    }

    async function handleDelete() {
        try {
            await api.deleteServer(serverId);
            goto('/servers');
        } catch (err) {
            error = err instanceof Error ? err.message : 'Failed to delete server';
            showDeleteConfirm = false;
        }
    }

    async function handleRegenerateToken() {
        try {
            const response = await api.regenerateToken(serverId);
            regeneratedToken = response.token;
            backendHost = window.location.host;
            showRegenerateConfirm = false;
            await loadServer();
        } catch (err) {
            error = err instanceof Error ? err.message : 'Failed to regenerate token';
            showRegenerateConfirm = false;
        }
    }

    async function handleRename() {
        try {
            await api.renameServer(serverId, newServerName);
            showRename = false;
            newServerName = '';
            await loadServer();
        } catch (err) {
            error = err instanceof Error ? err.message : 'Failed to rename server';
        }
    }

    async function handleChangeIP() {
        try {
            await api.updateConfiguredIP(serverId, newIP);
            showChangeIP = false;
            newIP = '';
            await loadServer();
        } catch (err) {
            error = err instanceof Error ? err.message : 'Failed to update IP';
        }
    }

    async function handleUpdateIP() {
        if (!server) return;
        try {
            await api.updateConfiguredIP(server.id, server.ip_address_v4);
            await loadServer();
        } catch (err) {
            error = err instanceof Error ? err.message : 'Failed to update IP';
        }
    }

    async function handleIgnoreIP() {
        if (!server) return;
        try {
            await api.ignoreIPMismatch(server.id);
            await loadServer();
        } catch (err) {
            error = err instanceof Error ? err.message : 'Failed to ignore IP mismatch';
        }
    }

    async function handleDismissReactivation() {
        if (!server) return;
        try {
            await api.dismissReactivation(server.id);
            await loadServer();
        } catch (err) {
            error = err instanceof Error ? err.message : 'Failed to dismiss reactivation';
        }
    }

    async function handlePause() {
        if (!server) return;
        const previousStatus = server.status;
        try {
            await api.pauseServer(server.id);
            server = { ...server, status: 'paused' };
        } catch (err) {
            server = { ...server, status: previousStatus };
            error = err instanceof Error ? err.message : 'Failed to pause server';
        }
    }

    async function handleResume() {
        if (!server) return;
        const previousStatus = server.status;
        try {
            await api.resumeServer(server.id);
            server = { ...server, status: 'online' };
        } catch (err) {
            server = { ...server, status: previousStatus };
            error = err instanceof Error ? err.message : 'Failed to resume server';
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
    <title>{server?.name || 'Server'} - Watchflare</title>
</svelte:head>

{#if loading}
    <div class="flex items-center justify-center py-20">
        <p class="text-muted-foreground">Loading server details...</p>
    </div>
{:else if error}
    <div class="rounded-lg border border-destructive bg-destructive/10 p-4">
        <p class="text-sm text-destructive">{error}</p>
    </div>
{:else if server}
    <ServerDetailHeader
        {server}
        metric={latestMetric}
        {latestAgentVersion}
        {hasActiveAlertRules}
        onDelete={() => (showDeleteConfirm = true)}
        onRegenerateToken={() => (showRegenerateConfirm = true)}
        onChangeIP={() => (showChangeIP = true)}
        onRename={() => { newServerName = server?.name || ''; showRename = true; }}
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
        <InstallInstructions {server} token={regeneratedToken} {backendHost} />
    {/if}

    <ServerAlerts
        {server}
        {showIPMismatchWarning}
        {clockDesync}
        onUpdateIP={handleUpdateIP}
        onIgnoreIP={handleIgnoreIP}
        onDismissReactivation={handleDismissReactivation}
    />

    <!-- Tab Navigation -->
    <div class="mb-6 flex gap-1 border-b">
        <a
            href="/servers/{serverId}"
            class="px-4 py-2.5 text-sm font-medium transition-colors border-b-2 -mb-px {isActiveTab('overview')
                ? 'border-primary text-foreground'
                : 'border-transparent text-muted-foreground hover:text-foreground'}"
        >
            Overview
        </a>
        <a
            href="/servers/{serverId}/packages"
            class="px-4 py-2.5 text-sm font-medium transition-colors border-b-2 -mb-px {isActiveTab('packages')
                ? 'border-primary text-foreground'
                : 'border-transparent text-muted-foreground hover:text-foreground'}"
        >
            Packages
        </a>
        <a
            href="/servers/{serverId}/incidents"
            class="px-4 py-2.5 text-sm font-medium transition-colors border-b-2 -mb-px {isActiveTab('incidents')
                ? 'border-primary text-foreground'
                : 'border-transparent text-muted-foreground hover:text-foreground'}"
        >
            Incidents
        </a>
    </div>

    {@render children()}

    <ServerAlertRulesDrawer
        serverId={serverId}
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
    confirmLabel="Delete Server"
    confirmVariant="destructive"
>
    <p class="text-sm text-muted-foreground mb-4">Are you sure you want to delete "{server?.name}"?</p>
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
        This will generate a new registration token and set the server to pending until the agent
        re-registers. Use the new token to run <code class="font-mono">watchflare-agent register</code>
        on the server.
    </p>
</ConfirmDialog>

<Modal open={showRename} onClose={() => { showRename = false; newServerName = ''; }}>
    <h3 class="text-lg font-semibold text-foreground mb-3">Rename Server</h3>
    <div class="mb-4">
        <label for="newname" class="block text-sm font-medium text-foreground mb-2">New Name</label>
        <input
            id="newname"
            type="text"
            bind:value={newServerName}
            placeholder="e.g., production-web-01"
            class="w-full rounded-lg border bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus-visible:ring-2 focus-visible:ring-primary"
        />
    </div>
    <div class="flex gap-3 justify-end">
        <button
            onclick={() => { showRename = false; newServerName = ''; }}
            class="rounded-lg border bg-background px-4 py-2 text-sm font-medium text-foreground transition-colors hover:bg-muted"
        >
            Cancel
        </button>
        <button
            onclick={handleRename}
            disabled={newServerName.length < 2}
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
