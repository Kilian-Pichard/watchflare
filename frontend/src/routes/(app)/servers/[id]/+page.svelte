<script lang="ts">
    import { onMount, onDestroy } from "svelte";
    import { goto } from "$app/navigation";
    import { page } from "$app/stores";
    import * as api from "$lib/api.js";
    import { sseStore } from "$lib/stores/sse";
    import { handleSSEReactivation, logger } from "$lib/utils";
    import { MAX_METRICS_POINTS_DETAIL } from "$lib/constants";
    import type { Server, Metric, PackageStats, SSEEvent, TimeRange } from "$lib/types";
    import ConfirmDialog from "$lib/components/ConfirmDialog.svelte";
    import Modal from "$lib/components/Modal.svelte";
    import ServerDetailHeader from "$lib/components/server/ServerDetailHeader.svelte";
    import ServerAlerts from "$lib/components/server/ServerAlerts.svelte";
    import ServerMetricsCharts from "$lib/components/server/ServerMetricsCharts.svelte";

    let server: Server | null = $state(null);
    let loading = $state(true);
    let error = $state("");
    let showDeleteConfirm = $state(false);
    let showRegenerateConfirm = $state(false);
    let showChangeIP = $state(false);
    let newIP = $state("");
    let regeneratedToken = $state("");
    let packageStats: PackageStats | null = $state(null);
    let metrics: Metric[] = $state([]);
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
            showRegenerateConfirm = false;
            await loadServer();
        } catch (err) {
            error = err.message || "Failed to regenerate token";
            showRegenerateConfirm = false;
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

    function copyToClipboard(text: string) {
        navigator.clipboard.writeText(text);
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



    function closeRegenerateModal() {
        showRegenerateConfirm = false;
        regeneratedToken = "";
    }

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
    />

    <ServerAlerts
        {server}
        {showIPMismatchWarning}
        onUpdateIP={handleUpdateIP}
        onIgnoreIP={handleIgnoreIP}
        onDismissReactivation={handleDismissReactivation}
    />

    <ServerMetricsCharts
        {metrics}
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

<!-- Regenerate Token Modal -->
<Modal open={showRegenerateConfirm} onClose={closeRegenerateModal}>
    {#if !regeneratedToken}
        <h3 class="text-lg font-semibold text-foreground mb-3">
            Regenerate Token
        </h3>
        <p class="text-sm text-muted-foreground mb-6">
            This will invalidate the current registration token. The
            agent will need to re-register.
        </p>
        <div class="flex gap-3 justify-end">
            <button
                onclick={() => (showRegenerateConfirm = false)}
                class="rounded-lg border bg-background px-4 py-2 text-sm font-medium text-foreground transition-colors hover:bg-muted"
            >
                Cancel
            </button>
            <button
                onclick={handleRegenerateToken}
                class="rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90"
            >
                Regenerate
            </button>
        </div>
    {:else}
        <h3 class="text-lg font-semibold text-success mb-3">
            Token Regenerated
        </h3>
        <div class="mb-4">
            <label
                class="block text-sm font-medium text-foreground mb-2"
                >New Registration Token</label
            >
            <div class="flex gap-2">
                <input
                    type="text"
                    readonly
                    value={regeneratedToken}
                    class="flex-1 rounded-lg border bg-muted px-3 py-2 font-mono text-xs text-foreground"
                />
                <button
                    onclick={() => copyToClipboard(regeneratedToken)}
                    class="rounded-lg border bg-background px-4 py-2 text-sm font-medium text-foreground transition-colors hover:bg-muted"
                >
                    Copy
                </button>
            </div>
            <p class="mt-2 text-xs font-medium text-warning">
                Save this token securely. It won't be shown again!
            </p>
        </div>
        <button
            onclick={closeRegenerateModal}
            class="w-full rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90"
        >
            Close
        </button>
    {/if}
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
