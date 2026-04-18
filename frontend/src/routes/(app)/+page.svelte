<script lang="ts">
    import { onMount, onDestroy } from "svelte";
    import { slide } from "svelte/transition";
    import { formatPercent, handleSSEReactivation, logger } from "$lib/utils";
    import {
        DROPPED_METRICS_POLL_INTERVAL,
        TREND_24H_POLL_INTERVAL,
    } from "$lib/constants";
    import {
        userStore,
        currentUser,
        hostsStore,
        aggregatedStore,
        aggregatedMetrics,
        currentTimeRange,
        dashboardStats,
        alertsStore,
        sseStore,
        uiStore,
    } from "$lib/stores";
    import DashboardStats from "$lib/components/dashboard/DashboardStats.svelte";
    import DashboardCharts from "$lib/components/dashboard/DashboardCharts.svelte";
    import DroppedMetricsAlert from "$lib/components/dashboard/DroppedMetricsAlert.svelte";
    import TimeRangeSelector from "$lib/components/TimeRangeSelector.svelte";
    import { ChevronUp, ChevronDown } from "lucide-svelte";
    import type { SSEEvent, TimeRange, HostUpdateEvent, AggregatedMetricsUpdateEvent } from "$lib/types";

    // SSE unsubscribe function
    let sseUnsubscribe: (() => void) | null = null;

    // Derived states from stores
    let loading = $state(true);
    let user = $derived($currentUser);
    let stats = $derived($dashboardStats);
    let droppedAlerts = $derived($alertsStore.droppedMetrics);
    let metricsCollapsed = $derived($uiStore.metricsCollapsed);
    let selectedTimeRange = $derived($currentTimeRange);

    async function loadData() {
        try {
            loading = true;

            // Ensure user preferences are loaded
            if (!$currentUser) {
                await userStore.load();
            }

            // Get time range from user preferences (migrate old 6h to 12h)
            const userTimeRange = $currentUser?.default_time_range || "24h";
            const migratedTimeRange: TimeRange =
                (userTimeRange as string) === "6h" ? "12h" : userTimeRange;
            aggregatedStore.setTimeRange(migratedTimeRange);

            // Load hosts (for stats) and aggregated metrics in parallel
            await Promise.all([
                hostsStore.load(),
                alertsStore.load(),
                aggregatedStore.load(migratedTimeRange),
                aggregatedStore.load24h(),
            ]);
        } catch (err) {
            logger.error("Failed to load data:", err);
        } finally {
            loading = false;
        }
    }

    async function handleTimeRangeChange(newTimeRange: TimeRange) {
        aggregatedStore.setTimeRange(newTimeRange);
        await aggregatedStore.load(newTimeRange);
    }

    function handleSSEMessage(event: SSEEvent) {
        handleSSEReactivation(event);

        if (event.type === "host_update") {
            const update = event.data as HostUpdateEvent;
            hostsStore.updateStatus(update.id, update.status, update.last_seen);
        } else if (event.type === "aggregated_metrics_update") {
            aggregatedStore.addMetricPoint(event.data as AggregatedMetricsUpdateEvent);
        }
    }

    onMount(() => {
        loadData();

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

        return () => {
            clearInterval(droppedMetricsInterval);
            clearInterval(trend24hInterval);
        };
    });

    onDestroy(() => {
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
    <div
        class="mb-6 grid grid-cols-2 gap-3 md:grid-cols-4 md:gap-4 animate-pulse"
    >
        {#each Array(4) as _}
            <div
                class="flex items-center gap-2.5 rounded-lg border bg-card px-3 py-2.5"
            >
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

    <!-- Dashboard Stats Cards -->
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

    {#if !metricsCollapsed}
        <div transition:slide={{ duration: 250 }}>
            <DashboardCharts
                aggregatedMetrics={$aggregatedMetrics}
                {stats}
                timeRange={selectedTimeRange}
            />
        </div>
    {/if}
{/if}
