<script lang="ts">
    import { onMount, onDestroy } from "svelte";
    import { slide } from "svelte/transition";
    import { get } from "svelte/store";
    import { formatPercent, handleSSEReactivation, logger } from "$lib/utils";
    import * as api from "$lib/api.js";
    import {
        DROPPED_METRICS_POLL_INTERVAL,
        TREND_24H_POLL_INTERVAL,
    } from "$lib/constants";
    import {
        userStore,
        currentUser,
        serversStore,
        servers,
        metricsStore,
        latestMetrics,
        aggregatedStore,
        aggregatedMetrics,
        currentTimeRange,
        dashboardStats,
        alertsStore,
        sseStore,
        uiStore,
    } from "$lib/stores";
    import ServerTable from "$lib/components/ServerTable.svelte";
    import DashboardStats from "$lib/components/dashboard/DashboardStats.svelte";
    import DashboardCharts from "$lib/components/dashboard/DashboardCharts.svelte";
    import DroppedMetricsAlert from "$lib/components/dashboard/DroppedMetricsAlert.svelte";
    import TimeRangeSelector from "$lib/components/TimeRangeSelector.svelte";
    import { ChevronUp, ChevronDown } from "lucide-svelte";
    import ConfirmDialog from "$lib/components/ConfirmDialog.svelte";
    import Modal from "$lib/components/Modal.svelte";
    import type { Server, SSEEvent, TimeRange } from "$lib/types";

    // SSE unsubscribe function
    let sseUnsubscribe: (() => void) | null = null;

    // Modal state
    let showDeleteConfirm = $state(false);
    let showRename = $state(false);
    let selectedServer: Server | null = $state(null);
    let newServerName = $state("");

    // Derived states from stores
    let loading = $state(true);
    let user = $derived($currentUser);
    let serversList = $derived($servers);
    let stats = $derived($dashboardStats);
    let droppedAlerts = $derived($alertsStore.droppedMetrics);
    let activeIncidentServerIds = $derived(
        $alertsStore.activeIncidents.reduce((map, i) => {
            map.set(i.server_id, (map.get(i.server_id) ?? 0) + 1);
            return map;
        }, new Map<string, number>())
    );
    let metricsCollapsed = $derived($uiStore.metricsCollapsed);
    let selectedTimeRange = $derived($currentTimeRange);

    async function loadServerMetrics(
        serverIds: string[],
        timeRangeValue: TimeRange,
    ) {
        if (serverIds.length > 0) {
            await metricsStore.loadForServers(serverIds, timeRangeValue);
        }
    }

    async function loadData() {
        try {
            loading = true;

            // Ensure user preferences are loaded
            if (!$currentUser) {
                await userStore.load();
            }

            // Get time range from user preferences (migrate old 6h to 12h)
            const userTimeRange = $currentUser?.default_time_range || "24h";
            const migratedTimeRange =
                userTimeRange === "6h" ? "12h" : userTimeRange;
            aggregatedStore.setTimeRange(migratedTimeRange);

            // Load servers and get IDs directly from API response
            await serversStore.load();

            // Load dropped metrics alerts, aggregated metrics, and per-server metrics in parallel
            const serverIds = get(serversStore).servers.map((s) => s.server.id);

            await Promise.all([
                loadServerMetrics(serverIds, migratedTimeRange as TimeRange),
                alertsStore.load(),
                aggregatedStore.load(migratedTimeRange),
                aggregatedStore.load24h(),
            ]);
        } catch (err) {
            logger.error("Failed to load data:", err);
            // Error will trigger redirect in apiRequest
        } finally {
            loading = false;
        }
    }

    async function handleTimeRangeChange(newTimeRange: TimeRange) {
        aggregatedStore.setTimeRange(newTimeRange);

        const serverIds = get(serversStore).servers.map((s) => s.server.id);

        // Load aggregated data and per-server metrics in parallel
        await Promise.all([
            aggregatedStore.load(newTimeRange),
            loadServerMetrics(serverIds, newTimeRange),
        ]);
    }

    function handleSSEMessage(event: SSEEvent) {
        handleSSEReactivation(event);

        if (event.type === "server_update") {
            // Update server status via store
            serversStore.updateStatus(
                event.data.id,
                event.data.status,
                event.data.last_seen,
            );
        } else if (event.type === "metrics_update") {
            // Update individual server metrics via store
            metricsStore.updateServerMetrics(event.data.server_id, event.data);
        } else if (event.type === "aggregated_metrics_update") {
            // Add new aggregated metrics point via store
            aggregatedStore.addMetricPoint(event.data);
        }
    }

    function handleRename(server: Server) {
        selectedServer = server;
        newServerName = server.name;
        showRename = true;
    }

    async function handleRenameSubmit() {
        if (!selectedServer) return;
        try {
            await api.renameServer(selectedServer.id, newServerName);
            showRename = false;
            newServerName = "";
            selectedServer = null;
            await serversStore.load();
        } catch (err) {
            logger.error("Failed to rename server:", err);
        }
    }

    async function handlePause(serverId: string) {
        try {
            await api.pauseServer(serverId);
            serversStore.updateStatus(serverId, "paused", "");
        } catch (err) {
            logger.error("Failed to pause server:", err);
        }
    }

    async function handleResume(serverId: string) {
        try {
            await api.resumeServer(serverId);
            serversStore.updateStatus(serverId, "online", "");
        } catch (err) {
            logger.error("Failed to resume server:", err);
        }
    }

    function handleDeleteRequest(server: Server) {
        selectedServer = server;
        showDeleteConfirm = true;
    }

    async function handleDeleteConfirm() {
        if (!selectedServer) return;
        try {
            await api.deleteServer(selectedServer.id);
            showDeleteConfirm = false;
            selectedServer = null;
            await serversStore.load();
        } catch (err) {
            logger.error("Failed to delete server:", err);
        }
    }

    onMount(() => {
        loadData();

        // Connect to SSE for server status updates and real-time aggregated metrics
        sseUnsubscribe = sseStore.connect(handleSSEMessage);

        // Refresh dropped metrics every 1 hour
        const droppedMetricsInterval = setInterval(
            () => alertsStore.load(),
            DROPPED_METRICS_POLL_INTERVAL,
        );

        // Refresh 24h metrics for trend calculation every 5 minutes
        const trend24hInterval = setInterval(
            () => aggregatedStore.load24h(),
            TREND_24H_POLL_INTERVAL,
        );

        // Cleanup on unmount
        return () => {
            clearInterval(droppedMetricsInterval);
            clearInterval(trend24hInterval);
        };
    });

    onDestroy(() => {
        // Unsubscribe from SSE
        if (sseUnsubscribe) {
            sseUnsubscribe();
        }
    });
</script>

<svelte:head>
    <title>Dashboard - Watchflare</title>
</svelte:head>

{#if loading}
    <div class="flex items-center justify-center py-20">
        <p class="text-muted-foreground">Loading dashboard...</p>
    </div>
{:else}
    <!-- Header with Welcome -->
    <div class="mb-6">
        <h1 class="text-xl sm:text-2xl font-semibold text-foreground">
            Welcome back, <span class="text-primary"
                >{user?.username || user?.email?.split("@")[0] || "User"}</span
            >
        </h1>
        <p class="text-sm text-muted-foreground mt-1">
            {#if stats.totalServers === 0}
                No servers monitored yet
            {:else}
                Global uptime at <span class="font-medium text-foreground"
                    >{formatPercent(
                        (stats.onlineServers / stats.totalServers) * 100,
                    )}</span
                > in the last 24h
            {/if}
        </p>
    </div>

    <!-- Dropped Metrics Alerts -->
    <DroppedMetricsAlert alerts={droppedAlerts} />

    <!-- Dashboard Stats Cards (always visible, always real-time) -->
    <DashboardStats {stats} />

    <!-- Global Metrics Charts Section -->
    <div class="mb-6 flex items-center justify-between h-10">
        <div class="flex items-center gap-2">
            <h2 class="text-lg font-semibold">Global Metrics</h2>
            <button
                onclick={() => uiStore.toggleMetricsCollapsed()}
                class="p-1 rounded-md text-muted-foreground hover:text-foreground hover:bg-muted transition-colors"
                aria-label={metricsCollapsed
                    ? "Expand metrics"
                    : "Collapse metrics"}
            >
                {#if metricsCollapsed}
                    <ChevronDown class="h-5 w-5" />
                {:else}
                    <ChevronUp class="h-5 w-5" />
                {/if}
            </button>
        </div>
        {#if !metricsCollapsed}
            <TimeRangeSelector
                bind:value={selectedTimeRange}
                onValueChange={handleTimeRangeChange}
            />
        {/if}
    </div>

    <!-- Dashboard Charts (hidden when collapsed) -->
    {#if !metricsCollapsed}
        <div transition:slide={{ duration: 250 }}>
            <DashboardCharts
                aggregatedMetrics={$aggregatedMetrics}
                {stats}
                timeRange={selectedTimeRange}
            />
        </div>
    {/if}

    <!-- Servers Table -->
    <div class="mb-6">
        <div class="mb-4 flex items-center justify-between">
            <h2 class="text-lg font-semibold">Server Summary</h2>
        </div>
        <ServerTable
            servers={serversList.filter((s) => s.server.status !== "pending")}
            latestMetrics={$latestMetrics}
            {activeIncidentServerIds}
            onRename={handleRename}
            onPause={handlePause}
            onResume={handleResume}
            onDelete={handleDeleteRequest}
        />
    </div>
{/if}

<!-- Delete Confirmation Modal -->
<ConfirmDialog
    open={showDeleteConfirm}
    title="Confirm Delete"
    onConfirm={handleDeleteConfirm}
    onClose={() => { showDeleteConfirm = false; selectedServer = null; }}
    confirmLabel="Delete Server"
    confirmVariant="destructive"
>
    <p class="text-sm text-muted-foreground mb-4">
        Are you sure you want to delete "{selectedServer?.name}"?
    </p>
    <p class="text-sm font-medium text-destructive">
        This action cannot be undone.
    </p>
</ConfirmDialog>

<!-- Rename Server Modal -->
<Modal open={showRename} onClose={() => { showRename = false; newServerName = ''; selectedServer = null; }}>
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
            class="w-full rounded-lg border bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus-visible:ring-2 focus-visible:ring-primary"
        />
    </div>
    <div class="flex gap-3 justify-end">
        <button
            onclick={() => { showRename = false; newServerName = ''; selectedServer = null; }}
            class="rounded-lg border bg-background px-4 py-2 text-sm font-medium text-foreground transition-colors hover:bg-muted"
        >
            Cancel
        </button>
        <button
            onclick={handleRenameSubmit}
            disabled={newServerName.length < 2}
            class="rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90 disabled:opacity-50"
        >
            Rename
        </button>
    </div>
</Modal>
