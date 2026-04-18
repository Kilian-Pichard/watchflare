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
        hostsStore,
        hosts,
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
    import HostTable from "$lib/components/HostTable.svelte";
    import DashboardStats from "$lib/components/dashboard/DashboardStats.svelte";
    import DashboardCharts from "$lib/components/dashboard/DashboardCharts.svelte";
    import DroppedMetricsAlert from "$lib/components/dashboard/DroppedMetricsAlert.svelte";
    import TimeRangeSelector from "$lib/components/TimeRangeSelector.svelte";
    import { ChevronUp, ChevronDown } from "lucide-svelte";
    import ConfirmDialog from "$lib/components/ConfirmDialog.svelte";
    import Modal from "$lib/components/Modal.svelte";
    import type { Host, SSEEvent, TimeRange } from "$lib/types";

    // SSE unsubscribe function
    let sseUnsubscribe: (() => void) | null = null;

    // Modal state
    let showDeleteConfirm = $state(false);
    let showRename = $state(false);
    let selectedHost: Host | null = $state(null);
    let newHostName = $state("");

    // Derived states from stores
    let loading = $state(true);
    let user = $derived($currentUser);
    let hostsList = $derived($hosts);
    let stats = $derived($dashboardStats);
    let droppedAlerts = $derived($alertsStore.droppedMetrics);
    let activeIncidentHostIds = $derived(
        $alertsStore.activeIncidents.reduce((map, i) => {
            map.set(i.host_id, (map.get(i.host_id) ?? 0) + 1);
            return map;
        }, new Map<string, number>())
    );
    let metricsCollapsed = $derived($uiStore.metricsCollapsed);
    let selectedTimeRange = $derived($currentTimeRange);

    async function loadHostMetrics(
        hostIds: string[],
        timeRangeValue: TimeRange,
    ) {
        if (hostIds.length > 0) {
            await metricsStore.loadForHosts(hostIds, timeRangeValue);
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

            // Load hosts and get IDs directly from API response
            await hostsStore.load();

            // Load dropped metrics alerts, aggregated metrics, and per-host metrics in parallel
            const hostIds = get(hostsStore).hosts.map((s) => s.host.id);

            await Promise.all([
                loadHostMetrics(hostIds, migratedTimeRange as TimeRange),
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

        const hostIds = get(hostsStore).hosts.map((s) => s.host.id);

        // Load aggregated data and per-host metrics in parallel
        await Promise.all([
            aggregatedStore.load(newTimeRange),
            loadHostMetrics(hostIds, newTimeRange),
        ]);
    }

    function handleSSEMessage(event: SSEEvent) {
        handleSSEReactivation(event);

        if (event.type === "host_update") {
            // Update host status via store
            hostsStore.updateStatus(
                event.data.id,
                event.data.status,
                event.data.last_seen,
            );
        } else if (event.type === "metrics_update") {
            // Update individual host metrics via store
            metricsStore.updateHostMetrics(event.data.host_id, event.data);
        } else if (event.type === "aggregated_metrics_update") {
            // Add new aggregated metrics point via store
            aggregatedStore.addMetricPoint(event.data);
        }
    }

    function handleRename(host: Host) {
        selectedHost = host;
        newHostName = host.display_name;
        showRename = true;
    }

    async function handleRenameSubmit() {
        if (!selectedHost) return;
        try {
            await api.renameHost(selectedHost.id, newHostName);
            showRename = false;
            newHostName = "";
            selectedHost = null;
            await hostsStore.load();
        } catch (err) {
            logger.error("Failed to rename host:", err);
        }
    }

    async function handlePause(hostId: string) {
        try {
            await api.pauseHost(hostId);
            hostsStore.updateStatus(hostId, "paused", "");
        } catch (err) {
            logger.error("Failed to pause host:", err);
        }
    }

    async function handleResume(hostId: string) {
        try {
            await api.resumeHost(hostId);
            hostsStore.updateStatus(hostId, "online", "");
        } catch (err) {
            logger.error("Failed to resume host:", err);
        }
    }

    function handleDeleteRequest(host: Host) {
        selectedHost = host;
        showDeleteConfirm = true;
    }

    async function handleDeleteConfirm() {
        if (!selectedHost) return;
        try {
            await api.deleteHost(selectedHost.id);
            showDeleteConfirm = false;
            selectedHost = null;
            await hostsStore.load();
        } catch (err) {
            logger.error("Failed to delete host:", err);
        }
    }

    onMount(() => {
        loadData();

        // Connect to SSE for host status updates and real-time aggregated metrics
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
    <!-- Skeleton: header -->
    <div class="mb-6 animate-pulse">
        <div class="h-7 w-48 rounded bg-muted mb-2"></div>
        <div class="h-4 w-64 rounded bg-muted"></div>
    </div>
    <!-- Skeleton: stats cards -->
    <div class="mb-6 grid grid-cols-2 gap-3 md:grid-cols-4 md:gap-4 animate-pulse">
        {#each Array(4) as _}
            <div class="flex items-center gap-2.5 rounded-lg border bg-card px-3 py-2.5">
                <div class="h-8 w-8 rounded-md bg-muted shrink-0"></div>
                <div class="flex flex-col gap-1.5">
                    <div class="h-3 w-12 rounded bg-muted"></div>
                    <div class="h-4 w-8 rounded bg-muted"></div>
                </div>
            </div>
        {/each}
    </div>
    <!-- Skeleton: charts -->
    <div class="mb-6 flex items-center h-10 animate-pulse">
        <div class="h-5 w-32 rounded bg-muted"></div>
    </div>
    <div class="mb-6 grid gap-4 xl:grid-cols-2 animate-pulse">
        {#each Array(2) as _}
            <div class="rounded-lg border bg-card p-4">
                <div class="mb-3 flex items-center justify-between">
                    <div class="h-4 w-20 rounded bg-muted"></div>
                    <div class="h-4 w-12 rounded bg-muted"></div>
                </div>
                <div class="h-40 rounded bg-muted"></div>
            </div>
        {/each}
    </div>
    <!-- Skeleton: table -->
    <div class="mb-4 h-6 w-32 rounded bg-muted animate-pulse"></div>
    <div class="rounded-xl border bg-card animate-pulse">
        <div class="border-b bg-table-header px-4 py-2.5 flex gap-8">
            {#each Array(5) as _}
                <div class="h-4 w-16 rounded bg-muted"></div>
            {/each}
        </div>
        {#each Array(5) as _}
            <div class="border-b px-4 py-3 flex gap-8">
                <div class="h-4 w-24 rounded bg-muted"></div>
                <div class="h-4 w-12 rounded bg-muted"></div>
                <div class="h-4 w-10 rounded bg-muted"></div>
                <div class="h-4 w-10 rounded bg-muted"></div>
                <div class="h-4 w-10 rounded bg-muted"></div>
            </div>
        {/each}
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
            {#if stats.totalHosts === 0}
                No hosts monitored yet
            {:else}
                Global uptime at <span class="font-medium text-foreground"
                    >{formatPercent(
                        (stats.onlineHosts / stats.totalHosts) * 100,
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

    <!-- Hosts Table -->
    <div class="mb-4 flex items-center justify-between">
        <h2 class="text-lg font-semibold">Host Summary</h2>
    </div>
    <HostTable
        hosts={hostsList.filter((s) => s.host.status !== "pending")}
        latestMetrics={$latestMetrics}
        {activeIncidentHostIds}
        onRename={handleRename}
        onPause={handlePause}
        onResume={handleResume}
        onDelete={handleDeleteRequest}
    />
{/if}

<!-- Delete Confirmation Modal -->
<ConfirmDialog
    open={showDeleteConfirm}
    title="Confirm Delete"
    onConfirm={handleDeleteConfirm}
    onClose={() => { showDeleteConfirm = false; selectedHost = null; }}
    confirmLabel="Delete Host"
    confirmVariant="destructive"
>
    <p class="text-sm text-muted-foreground mb-4">
        Are you sure you want to delete "{selectedHost?.display_name}"?
    </p>
    <p class="text-sm font-medium text-destructive">
        This action cannot be undone.
    </p>
</ConfirmDialog>

<!-- Rename Host Modal -->
<Modal open={showRename} onClose={() => { showRename = false; newHostName = ''; selectedHost = null; }}>
    <h3 class="text-lg font-semibold text-foreground mb-3">
        Rename Host
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
            bind:value={newHostName}
            placeholder="e.g., production-web-01"
            class="w-full rounded-lg border bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus-visible:ring-2 focus-visible:ring-primary"
        />
    </div>
    <div class="flex gap-3 justify-end">
        <button
            onclick={() => { showRename = false; newHostName = ''; selectedHost = null; }}
            class="rounded-lg border bg-background px-4 py-2 text-sm font-medium text-foreground transition-colors hover:bg-muted"
        >
            Cancel
        </button>
        <button
            onclick={handleRenameSubmit}
            disabled={newHostName.length < 2}
            class="rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90 disabled:opacity-50"
        >
            Rename
        </button>
    </div>
</Modal>
