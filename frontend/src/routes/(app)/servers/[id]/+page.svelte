<script lang="ts">
    import { onMount, onDestroy } from "svelte";
    import { goto } from "$app/navigation";
    import { page } from "$app/stores";
    import * as api from "$lib/api.js";
    import { sseStore } from "$lib/stores/sse";
    import {
        formatBytes,
        getStatusClass,
        formatDateTime,
        formatRelativeTime,
        handleSSEReactivation,
        logger,
    } from "$lib/utils";
    import CPUChart from "$lib/components/CPUChart.svelte";
    import MemoryChart from "$lib/components/MemoryChart.svelte";
    import DiskChart from "$lib/components/DiskChart.svelte";
    import TimeRangeSelector from "$lib/components/TimeRangeSelector.svelte";
    import type { Server, Metric, PackageStats, SSEEvent, TimeRange } from "$lib/types";
    import ConfirmDialog from "$lib/components/ConfirmDialog.svelte";
    import Modal from "$lib/components/Modal.svelte";

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

                // Keep only last 200 points to avoid memory issues
                if (metrics.length > 200) {
                    metrics = metrics.slice(-200);
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
        server &&
            server.configured_ip &&
            server.ip_address_v4 &&
            server.configured_ip !== server.ip_address_v4 &&
            !server.ignore_ip_mismatch,
    );

    const latestMetric = $derived(
        metrics.length > 0 ? metrics[metrics.length - 1] : null,
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
    <!-- Header -->
    <div class="mb-6 rounded-lg border bg-card p-4 md:p-6">
        <!-- Top: Name, status, actions -->
        <div
            class="flex flex-col gap-4 md:flex-row md:items-start md:justify-between"
        >
            <div class="flex items-center gap-3 flex-wrap">
                <h1 class="text-xl sm:text-2xl font-semibold text-foreground">
                    {server.name}
                </h1>
                <span
                    class="inline-flex items-center gap-1.5 rounded-full border px-2.5 py-0.5 text-xs font-medium {getStatusClass(
                        server.status,
                    )}"
                >
                    <span
                        class="h-1.5 w-1.5 rounded-full {server.status ===
                        'online'
                            ? 'bg-success'
                            : 'bg-muted-foreground'}"
                    ></span>
                    {server.status}
                </span>
            </div>
            <div class="flex flex-wrap gap-2">
                <button
                    onclick={() => (showChangeIP = true)}
                    class="rounded-lg border bg-background px-3 py-1.5 text-sm font-medium text-foreground transition-colors hover:bg-muted whitespace-nowrap"
                >
                    Change IP
                </button>
                <button
                    onclick={() => (showRegenerateConfirm = true)}
                    class="rounded-lg border bg-background px-3 py-1.5 text-sm font-medium text-foreground transition-colors hover:bg-muted whitespace-nowrap"
                >
                    Regenerate Token
                </button>
                <button
                    onclick={() => (showDeleteConfirm = true)}
                    class="rounded-lg border border-destructive bg-destructive/10 px-3 py-1.5 text-sm font-medium text-destructive transition-colors hover:bg-destructive/20 whitespace-nowrap"
                >
                    Delete
                </button>
            </div>
        </div>

        <!-- Server details grid -->
        <div class="mt-6 grid grid-cols-2 md:grid-cols-4 gap-x-6 gap-y-3">
            {#if server.hostname}
                <div>
                    <p class="text-xs text-muted-foreground">Hostname</p>
                    <p class="text-sm font-medium text-foreground">
                        {server.hostname}
                    </p>
                </div>
            {/if}
            {#if server.platform}
                <div>
                    <p class="text-xs text-muted-foreground">OS</p>
                    <p class="text-sm font-medium text-foreground">
                        {server.platform_version
                            ? `${server.platform} ${server.platform_version}`
                            : server.platform}
                    </p>
                </div>
            {/if}
            {#if server.architecture}
                <div>
                    <p class="text-xs text-muted-foreground">Architecture</p>
                    <p class="text-sm font-medium text-foreground">
                        {server.architecture}
                    </p>
                </div>
            {/if}
            {#if server.ip_address_v4 || server.configured_ip}
                <div>
                    <p class="text-xs text-muted-foreground">IP Address</p>
                    <p class="text-sm font-medium text-foreground">
                        {server.ip_address_v4 || server.configured_ip}
                    </p>
                </div>
            {/if}
            {#if server.last_seen}
                <div>
                    <p class="text-xs text-muted-foreground">Last Seen</p>
                    <p class="text-sm font-medium text-foreground">
                        {formatRelativeTime(server.last_seen)}
                    </p>
                </div>
            {/if}
        </div>

        <!-- Packages -->
        {#if packageStats}
            <div class="mt-4 pt-4 border-t flex items-center justify-between">
                <div class="flex items-center gap-4">
                    <div>
                        <p class="text-xs text-muted-foreground">Packages</p>
                        <p class="text-sm font-medium text-foreground">
                            {packageStats.total_packages || 0} installed
                        </p>
                    </div>
                    {#if packageStats.recent_changes}
                        <div>
                            <p class="text-xs text-muted-foreground">
                                Recent Changes
                            </p>
                            <p class="text-sm font-medium text-foreground">
                                {packageStats.recent_changes} in the last 30d
                            </p>
                        </div>
                    {/if}
                </div>
                <a
                    href="/servers/{serverId}/packages"
                    class="rounded-lg border bg-background px-3 py-1.5 text-sm font-medium text-foreground transition-colors hover:bg-muted whitespace-nowrap"
                >
                    View Packages →
                </a>
            </div>
        {/if}
    </div>

    <!-- Alerts -->
    {#if showIPMismatchWarning}
        <div class="mb-6 rounded-lg border border-warning bg-warning/10 p-4">
            <div class="flex items-start gap-3">
                <svg
                    class="h-5 w-5 text-warning mt-0.5"
                    fill="currentColor"
                    viewBox="0 0 20 20"
                >
                    <path
                        fill-rule="evenodd"
                        d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z"
                        clip-rule="evenodd"
                    />
                </svg>
                <div class="flex-1">
                    <p class="text-sm font-medium text-foreground">
                        IP Address Mismatch
                    </p>
                    <p class="text-sm text-muted-foreground mt-1">
                        Configured IP: {server.configured_ip} • Actual IP: {server.ip_address_v4}
                    </p>
                    <div class="mt-3 flex gap-2">
                        <button
                            onclick={handleUpdateIP}
                            class="rounded-lg bg-primary px-3 py-1.5 text-xs font-medium text-primary-foreground transition-colors hover:bg-primary/90"
                        >
                            Update to {server.ip_address_v4}
                        </button>
                        <button
                            onclick={handleIgnoreIP}
                            class="rounded-lg border bg-background px-3 py-1.5 text-xs font-medium text-foreground transition-colors hover:bg-muted"
                        >
                            Ignore
                        </button>
                    </div>
                </div>
            </div>
        </div>
    {/if}

    {#if server.reactivated_at}
        <div class="mb-6 rounded-lg border border-primary bg-primary/10 p-4">
            <div class="flex items-start justify-between gap-3">
                <div class="flex items-start gap-3">
                    <svg
                        class="h-5 w-5 text-primary mt-0.5"
                        fill="currentColor"
                        viewBox="0 0 20 20"
                    >
                        <path
                            fill-rule="evenodd"
                            d="M4 2a1 1 0 011 1v2.101a7.002 7.002 0 0111.601 2.566 1 1 0 11-1.885.666A5.002 5.002 0 005.999 7H9a1 1 0 010 2H4a1 1 0 01-1-1V3a1 1 0 011-1zm.008 9.057a1 1 0 011.276.61A5.002 5.002 0 0014.001 13H11a1 1 0 110-2h5a1 1 0 011 1v5a1 1 0 11-2 0v-2.101a7.002 7.002 0 01-11.601-2.566 1 1 0 01.61-1.276z"
                            clip-rule="evenodd"
                        />
                    </svg>
                    <div>
                        <p class="text-sm font-medium text-foreground">
                            Agent Reactivated
                        </p>
                        <p class="text-sm text-muted-foreground mt-1">
                            Same physical server detected via UUID at {formatDateTime(
                                server.reactivated_at,
                            )}
                        </p>
                    </div>
                </div>
                <button
                    onclick={handleDismissReactivation}
                    class="text-primary hover:text-primary/80"
                >
                    <svg
                        class="h-5 w-5"
                        fill="currentColor"
                        viewBox="0 0 20 20"
                    >
                        <path
                            fill-rule="evenodd"
                            d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z"
                            clip-rule="evenodd"
                        />
                    </svg>
                </button>
            </div>
        </div>
    {/if}

    <!-- Charts -->
    <div class="mb-6">
        <div class="mb-4 flex items-center justify-between">
            <h2 class="text-lg font-semibold text-foreground">Metrics</h2>
            <TimeRangeSelector
                bind:value={timeRange}
                onValueChange={handleTimeRangeChange}
            />
        </div>

        <div class="grid gap-4 xl:grid-cols-2">
            <div class="rounded-lg border bg-card p-4">
                <div class="flex items-center justify-between mb-3">
                    <h3 class="text-sm font-medium text-foreground">
                        CPU Usage
                    </h3>
                    {#if latestMetric}
                        <span class="text-sm font-semibold text-foreground"
                            >{latestMetric.cpu_usage_percent.toFixed(1)}%</span
                        >
                    {/if}
                </div>
                <CPUChart data={metrics} />
            </div>
            <div class="rounded-lg border bg-card p-4">
                <div class="flex items-center justify-between mb-3">
                    <h3 class="text-sm font-medium text-foreground">
                        Memory Usage
                    </h3>
                    {#if latestMetric}
                        <span class="text-sm font-semibold text-foreground"
                            >{formatBytes(latestMetric.memory_used_bytes)} / {formatBytes(
                                latestMetric.memory_total_bytes,
                            )}</span
                        >
                    {/if}
                </div>
                <MemoryChart data={metrics} />
            </div>
            <div class="rounded-lg border bg-card p-4">
                <div class="flex items-center justify-between mb-3">
                    <h3 class="text-sm font-medium text-foreground">
                        Disk Usage
                    </h3>
                    {#if latestMetric}
                        <span class="text-sm font-semibold text-foreground"
                            >{formatBytes(latestMetric.disk_used_bytes)} / {formatBytes(
                                latestMetric.disk_total_bytes,
                            )}</span
                        >
                    {/if}
                </div>
                <DiskChart data={metrics} />
            </div>
        </div>
    </div>
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
