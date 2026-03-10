<script lang="ts">
    import { onMount, onDestroy } from "svelte";
    import { goto } from "$app/navigation";
    import { page } from "$app/stores";
    import * as api from "$lib/api.js";
    import { sseStore } from "$lib/stores/sse";
    import { handleSSEReactivation, logger } from "$lib/utils";
    import { MAX_METRICS_POINTS_DETAIL } from "$lib/constants";
    import type { Server, Metric, ContainerMetric, PackageStats, SSEEvent, TimeRange } from "$lib/types";
    import ConfirmDialog from "$lib/components/ConfirmDialog.svelte";
    import Modal from "$lib/components/Modal.svelte";
    import ServerDetailHeader from "$lib/components/server/ServerDetailHeader.svelte";
    import ServerAlerts from "$lib/components/server/ServerAlerts.svelte";
    import ServerMetricsCharts from "$lib/components/server/ServerMetricsCharts.svelte";
    import InstallInstructions from "$lib/components/InstallInstructions.svelte";

    let server: Server | null = $state(null);
    let loading = $state(true);
    let error = $state("");
    let showDeleteConfirm = $state(false);
    let showRegenerateConfirm = $state(false);
    let showChangeIP = $state(false);
    let newIP = $state("");
    let showRename = $state(false);
    let newServerName = $state("");
    let regeneratedToken = $state("");
    let backendHost = $state("");
    let copiedToken = $state(false);
    let packageStats: PackageStats | null = $state(null);
    let metrics: Metric[] = $state([]);
    let containerMetrics: ContainerMetric[] = $state([]);
    let timeRange: TimeRange = $state("1h");
    let sseUnsubscribe: (() => void) | null = null;

    const serverId = $derived($page.params.id);

    function handleSSEMessage(event: SSEEvent) {
        handleSSEReactivation(event);

        // Handle server_update events for this specific server
        if (event.type === "server_update") {
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
            }
        }

        // Handle metrics_update events for this specific server
        if (event.type === "metrics_update") {
            const metric = event.data;
            if (server && metric.server_id === server.id) {
                // Add new metric to the array
                metrics = [...metrics, metric];

                // Keep only last N points to avoid memory issues
                if (metrics.length > MAX_METRICS_POINTS_DETAIL) {
                    metrics = metrics.slice(-MAX_METRICS_POINTS_DETAIL);
                }
            }
        }

        // Handle container_metrics_update events for this specific server
        if (event.type === "container_metrics_update") {
            const update = event.data as { server_id: string; metrics: ContainerMetric[] };
            if (server && update.server_id === server.id) {
                containerMetrics = [...containerMetrics, ...update.metrics];

                // Keep only last N points to avoid memory issues
                if (containerMetrics.length > MAX_METRICS_POINTS_DETAIL) {
                    containerMetrics = containerMetrics.slice(-MAX_METRICS_POINTS_DETAIL);
                }
            }
        }
    }

    onMount(() => {
        sseUnsubscribe = sseStore.connect(handleSSEMessage);
    });

    onDestroy(() => {
        if (sseUnsubscribe) {
            sseUnsubscribe();
        }
    });

    // Load data when serverId changes
    $effect(() => {
        if (serverId) {
            loadServer();
            loadMetrics();
        }
    });

    async function loadServer() {
        try {
            const response = await api.getServer(serverId);
            server = response.server;

            if (server.status === "online") {
                try {
                    packageStats = await api.getPackageStats(serverId);
                } catch (err) {
                    logger.error("Failed to load package stats:", err);
                }
            }
        } catch (err) {
            error = err.message || "Failed to load server";
        } finally {
            loading = false;
        }
    }

    async function loadMetrics() {
        try {
            const data = await api.getServerMetrics(serverId, {
                time_range: timeRange,
            });
            metrics = data.metrics || [];
        } catch (err) {
            logger.error("Failed to load metrics:", err);
        }

        // Load container metrics
        try {
            const containerData = await api.getContainerMetrics(serverId, timeRange);
            containerMetrics = containerData.metrics || [];
        } catch (err) {
            logger.error("Failed to load container metrics:", err);
            containerMetrics = [];
        }
    }

    async function handleDelete() {
        try {
            await api.deleteServer(serverId);
            goto("/servers");
        } catch (err) {
            error = err.message || "Failed to delete server";
            showDeleteConfirm = false;
        }
    }

    async function handleRegenerateToken() {
        try {
            const response = await api.regenerateToken(serverId);
            regeneratedToken = response.token;
            backendHost = window.location.hostname;
            showRegenerateConfirm = false;
            await loadServer();
        } catch (err) {
            error = err.message || "Failed to regenerate token";
            showRegenerateConfirm = false;
        }
    }

    async function handleRename() {
        try {
            await api.renameServer(serverId, newServerName);
            showRename = false;
            newServerName = "";
            await loadServer();
        } catch (err) {
            error = err.message || "Failed to rename server";
        }
    }

    async function handleChangeIP() {
        try {
            await api.updateConfiguredIP(serverId, newIP);
            showChangeIP = false;
            newIP = "";
            await loadServer();
        } catch (err) {
            error = err.message || "Failed to update IP";
        }
    }

    async function handleUpdateIP() {
        if (!server) return;
        try {
            await api.updateConfiguredIP(server.id, server.ip_address_v4);
            await loadServer();
        } catch (err) {
            error = err.message || "Failed to update IP";
        }
    }

    async function handleIgnoreIP() {
        if (!server) return;
        try {
            await api.ignoreIPMismatch(server.id);
            await loadServer();
        } catch (err) {
            error = err.message || "Failed to ignore IP mismatch";
        }
    }

    async function handleDismissReactivation() {
        if (!server) return;
        try {
            await api.dismissReactivation(server.id);
            await loadServer();
        } catch (err) {
            error = err.message || "Failed to dismiss reactivation";
        }
    }

    function handleCopy(text: string) {
        navigator.clipboard.writeText(text);
    }

    async function handlePause() {
        if (!server) return;
        try {
            await api.pauseServer(server.id);
            server = { ...server, status: 'paused' };
        } catch (err) {
            error = err.message || "Failed to pause server";
        }
    }

    async function handleResume() {
        if (!server) return;
        try {
            await api.resumeServer(server.id);
            server = { ...server, status: 'online' };
        } catch (err) {
            error = err.message || "Failed to resume server";
        }
    }

    function handleTimeRangeChange() {
        loadMetrics();
    }

    const showIPMismatchWarning = $derived(
        !!(server &&
            server.configured_ip &&
            server.ip_address_v4 &&
            server.configured_ip !== server.ip_address_v4 &&
            !server.ignore_ip_mismatch),
    );



    function closeChangeIPModal() {
        showChangeIP = false;
        newIP = "";
    }
</script>

<svelte:head>
    <title>{server?.name || "Server"} - Watchflare</title>
</svelte:head>

<!-- Back Link -->
<div class="mb-6">
    <a
        href="/servers"
        class="inline-flex items-center gap-2 text-sm font-medium text-primary hover:text-primary/80 transition-colors"
    >
        <svg
            class="h-4 w-4"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
        >
            <path
                stroke-linecap="round"
                stroke-linejoin="round"
                stroke-width="2"
                d="M15 19l-7-7 7-7"
            />
        </svg>
        Back to Servers
    </a>
</div>

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
        {packageStats}
        onDelete={() => (showDeleteConfirm = true)}
        onRegenerateToken={() => (showRegenerateConfirm = true)}
        onChangeIP={() => (showChangeIP = true)}
        onRename={() => { newServerName = server?.name || ''; showRename = true; }}
        onPause={handlePause}
        onResume={handleResume}
    />

    {#if regeneratedToken}
        <div class="mb-6 rounded-lg border border-warning bg-warning/10 p-4 flex items-center justify-between gap-4 flex-wrap">
            <p class="text-sm font-medium text-warning">This token is valid for 24 hours and will not be displayed again. Make sure to copy it or use it now.</p>
            <button
                onclick={() => { handleCopy(regeneratedToken); copiedToken = true; setTimeout(() => copiedToken = false, 2000); }}
                class="shrink-0 rounded-lg border bg-background px-3 py-1.5 text-sm font-medium text-foreground transition-colors hover:bg-muted"
            >
                {copiedToken ? 'Copied!' : 'Copy Token'}
            </button>
        </div>
        <InstallInstructions {server} token={regeneratedToken} {backendHost} />
    {/if}

    <ServerAlerts
        {server}
        {showIPMismatchWarning}
        onUpdateIP={handleUpdateIP}
        onIgnoreIP={handleIgnoreIP}
        onDismissReactivation={handleDismissReactivation}
    />

    <ServerMetricsCharts
        {metrics}
        {containerMetrics}
        bind:timeRange
        onTimeRangeChange={handleTimeRangeChange}
    />
{/if}

<!-- Delete Confirmation Modal -->
<ConfirmDialog
    open={showDeleteConfirm}
    title="Confirm Delete"
    onConfirm={handleDelete}
    onClose={() => (showDeleteConfirm = false)}
    confirmLabel="Delete Server"
    confirmVariant="destructive"
>
    <p class="text-sm text-muted-foreground mb-4">
        Are you sure you want to delete "{server?.name}"?
    </p>
    <p class="text-sm font-medium text-destructive">
        This action cannot be undone.
    </p>
</ConfirmDialog>

<!-- Regenerate Token Confirmation -->
<ConfirmDialog
    open={showRegenerateConfirm}
    title="Regenerate Token"
    onConfirm={handleRegenerateToken}
    onClose={() => (showRegenerateConfirm = false)}
    confirmLabel="Regenerate"
>
    <p class="text-sm text-muted-foreground">
        This will invalidate the current registration token and generate a new one.
    </p>
</ConfirmDialog>

<!-- Rename Server Modal -->
<Modal open={showRename} onClose={() => { showRename = false; newServerName = ''; }}>
    <h3 class="text-lg font-semibold text-foreground mb-3">
        Rename Server
    </h3>
    <div class="mb-4">
        <label
            for="newname"
            class="block text-sm font-medium text-foreground mb-2"
        >
            New Name
        </label>
        <input
            id="newname"
            type="text"
            bind:value={newServerName}
            placeholder="e.g., production-web-01"
            class="w-full rounded-lg border bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary"
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

<!-- Change IP Modal -->
<Modal open={showChangeIP} onClose={closeChangeIPModal}>
    <h3 class="text-lg font-semibold text-foreground mb-3">
        Change Configured IP
    </h3>
    <div class="mb-4">
        <label
            for="newip"
            class="block text-sm font-medium text-foreground mb-2"
        >
            New IP Address
        </label>
        <input
            id="newip"
            type="text"
            bind:value={newIP}
            placeholder="e.g., 192.168.1.100"
            class="w-full rounded-lg border bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary"
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
            class="rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90"
        >
            Update IP
        </button>
    </div>
</Modal>
