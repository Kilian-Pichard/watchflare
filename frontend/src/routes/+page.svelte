<script lang="ts">
    import { onMount, onDestroy } from "svelte";
    import { goto } from "$app/navigation";
    import {
        logout,
        getCurrentUser,
        listServers,
        getDroppedMetrics,
        getAggregatedMetrics,
        getServerMetrics,
    } from "$lib/api";
    import { connectSSE } from "$lib/sse";
    import { formatPercent } from "$lib/utils";
    import { toasts } from "$lib/stores/toasts";
    import { sidebarCollapsed } from "$lib/stores/sidebar";
    import DesktopSidebar from "$lib/components/DesktopSidebar.svelte";
    import MobileSidebar from "$lib/components/MobileSidebar.svelte";
    import Header from "$lib/components/Header.svelte";
    import ServerTable from "$lib/components/ServerTable.svelte";
    import DashboardStats from "$lib/components/dashboard/DashboardStats.svelte";
    import DashboardCharts from "$lib/components/dashboard/DashboardCharts.svelte";
    import DroppedMetricsAlert from "$lib/components/dashboard/DroppedMetricsAlert.svelte";
    import RightSidebar from "$lib/components/RightSidebar.svelte";
    import type {
        User,
        Server,
        Metric,
        AggregatedMetric,
        DroppedMetric,
        TimeRange,
        ServerWithMetrics
    } from "$lib/types";

    // State
    let user = $state<User | null>(null);
    let servers = $state<ServerWithMetrics[]>([]);
    let timeRange = $state<TimeRange>("24h");
    let loading = $state<boolean>(true);
    let metricsData = $state<Record<string, Metric[]>>({}); // { serverId: [metrics] }
    let sseDisconnect: (() => void) | null = null;
    let droppedMetricsAlerts = $state<DroppedMetric[]>([]);
    let aggregatedMetrics = $state<AggregatedMetric[]>([]);
    let aggregatedMetrics24h = $state<AggregatedMetric[]>([]); // Metrics for 24h trend calculation
    let rightSidebarOpen = $state<boolean>(false); // Right sidebar toggle state

    // Computed aggregates - uses last point from backend aggregated metrics
    let stats = $derived.by(() => {
        const lastPoint = aggregatedMetrics.length > 0
            ? aggregatedMetrics[aggregatedMetrics.length - 1]
            : null;

        // Calculate trend by comparing current to first point from 24h data (true 24h trend)
        const firstPoint24h = aggregatedMetrics24h.length > 0
            ? aggregatedMetrics24h[0]
            : null;

        const totalServers = servers.length;
        const onlineServers = servers.filter((s) => s.server.status === "online").length;
        const avgCPU = lastPoint?.cpu_usage_percent || 0;
        const totalMemory = lastPoint?.memory_total_bytes || 0;
        const usedMemory = lastPoint?.memory_used_bytes || 0;
        const totalDisk = lastPoint?.disk_total_bytes || 0;
        const usedDisk = lastPoint?.disk_used_bytes || 0;

        // Calculate trends (comparing current to 24h ago)
        const cpuTrend = firstPoint24h && firstPoint24h.cpu_usage_percent > 0
            ? ((avgCPU - firstPoint24h.cpu_usage_percent) / firstPoint24h.cpu_usage_percent) * 100
            : 0;

        const memoryPercent = totalMemory > 0 ? (usedMemory / totalMemory) * 100 : 0;
        const compareMemoryPercent = firstPoint24h && firstPoint24h.memory_total_bytes > 0
            ? (firstPoint24h.memory_used_bytes / firstPoint24h.memory_total_bytes) * 100
            : memoryPercent;
        const memoryTrend = compareMemoryPercent > 0
            ? ((memoryPercent - compareMemoryPercent) / compareMemoryPercent) * 100
            : 0;

        const diskPercent = totalDisk > 0 ? (usedDisk / totalDisk) * 100 : 0;
        const compareDiskPercent = firstPoint24h && firstPoint24h.disk_total_bytes > 0
            ? (firstPoint24h.disk_used_bytes / firstPoint24h.disk_total_bytes) * 100
            : diskPercent;
        const diskTrend = compareDiskPercent > 0
            ? ((diskPercent - compareDiskPercent) / compareDiskPercent) * 100
            : 0;

        return {
            totalServers,
            onlineServers,
            offlineServers: totalServers - onlineServers,
            avgCPU,
            avgMemory: memoryPercent,
            avgDisk: diskPercent,
            totalMemory,
            usedMemory,
            totalDisk,
            usedDisk,
            cpuTrend: isFinite(cpuTrend) ? cpuTrend : 0,
            memoryTrend: isFinite(memoryTrend) ? memoryTrend : 0,
            diskTrend: isFinite(diskTrend) ? diskTrend : 0,
        };
    });


    async function handleLogout() {
        try {
            await logout();
            goto("/login");
        } catch (err) {
            console.error("Logout failed:", err);
            goto("/login");
        }
    }

    function toggleRightSidebar() {
        rightSidebarOpen = !rightSidebarOpen;
    }

    async function loadData() {
        try {
            loading = true;

            // Load user preferences
            const userData = await getCurrentUser();
            if (!userData || !userData.user) {
                console.error("No user data received");
                return;
            }
            user = userData.user;
            // Migrate old 6h preference to 12h
            const userTimeRange = user.default_time_range || "24h";
            timeRange = userTimeRange === "6h" ? "12h" : userTimeRange;

            // Load servers
            const serversData = await listServers();
            servers = serversData.servers.map((server) => ({
                server,
            }));

            // Load metrics for each server (for table display)
            await loadServerMetrics();

            // Load dropped metrics alerts
            await loadDroppedMetrics();

            // Load aggregated metrics (for stats cards and charts)
            await loadAggregatedMetrics();

            // Load 24h metrics for trend calculation
            await loadAggregatedMetrics24h();
        } catch (err) {
            console.error("Failed to load data:", err);
            // Error will trigger redirect in apiRequest
        } finally {
            loading = false;
        }
    }

    async function loadServerMetrics() {
        try {
            const promises = servers.map(async ({ server }) => {
                try {
                    const data = await getServerMetrics(server.id, { time_range: "1h" });
                    metricsData[server.id] = data.metrics || [];
                } catch (err) {
                    console.error(`Failed to load metrics for ${server.hostname}:`, err);
                    metricsData[server.id] = [];
                }
            });
            await Promise.all(promises);
        } catch (err) {
            console.error("Failed to load server metrics:", err);
        }
    }

    async function loadDroppedMetrics() {
        try {
            const data = await getDroppedMetrics();
            droppedMetricsAlerts = data || [];
        } catch (err) {
            console.error("Failed to load dropped metrics:", err);
        }
    }

    async function loadAggregatedMetrics() {
        try {
            const data = await getAggregatedMetrics(timeRange);
            aggregatedMetrics = data.metrics || [];
        } catch (err) {
            console.error("Failed to load aggregated metrics:", err);
        }
    }

    async function loadAggregatedMetrics24h() {
        try {
            const data = await getAggregatedMetrics("24h");
            aggregatedMetrics24h = data.metrics || [];
        } catch (err) {
            console.error("Failed to load 24h aggregated metrics for trends:", err);
        }
    }

    function handleTimeRangeChange() {
        loadAggregatedMetrics();
    }

    function handleSSEMessage(event) {
        console.log("SSE event received:", event.type, event.data);

        if (event.type === "server_update") {
            // Show toast notification if agent was reactivated
            if (event.data.reactivated && event.data.hostname) {
                toasts.add(
                    `Agent "${event.data.hostname}" was reactivated (same physical server detected via UUID)`,
                    'info',
                    8000 // 8 seconds
                );
            }

            // Update server status
            const serverIndex = servers.findIndex(
                (s) => s.server.id === event.data.id,
            );
            if (serverIndex !== -1) {
                servers[serverIndex].server.status = event.data.status;
                servers[serverIndex].server.last_seen = event.data.last_seen;
            }
        } else if (event.type === "metrics_update") {
            // Update individual server metrics for table display
            const serverId = event.data.server_id;
            if (!metricsData[serverId]) {
                metricsData[serverId] = [];
            }
            metricsData[serverId] = [...metricsData[serverId], event.data];

            // Keep only last 50 points per server
            if (metricsData[serverId].length > 50) {
                metricsData[serverId] = metricsData[serverId].slice(-50);
            }
        } else if (event.type === "aggregated_metrics_update") {
            console.log("Aggregated metrics update received:", event.data);
            // Add new aggregated metrics point in real-time
            const newPoint = event.data;
            aggregatedMetrics = [...aggregatedMetrics, newPoint];

            console.log("Updated aggregatedMetrics, new length:", aggregatedMetrics.length);

            // Keep only last 200 points to avoid memory issues
            if (aggregatedMetrics.length > 200) {
                aggregatedMetrics = aggregatedMetrics.slice(-200);
            }
        }
    }

    onMount(() => {
        loadData();

        // Connect to SSE for server status updates and real-time aggregated metrics
        sseDisconnect = connectSSE(handleSSEMessage, (error) => {
            console.error("SSE error:", error);
        });

        // Refresh dropped metrics every 1 hour
        const droppedMetricsInterval = setInterval(loadDroppedMetrics, 3600000);

        // Refresh 24h metrics for trend calculation every 5 minutes
        const trend24hInterval = setInterval(loadAggregatedMetrics24h, 300000);

        // Cleanup on unmount
        return () => {
            clearInterval(droppedMetricsInterval);
            clearInterval(trend24hInterval);
        };
    });

    onDestroy(() => {
        if (sseDisconnect) {
            sseDisconnect();
        }
    });
</script>

<svelte:head>
    <title>Dashboard - Watchflare</title>
</svelte:head>

<div class="min-h-screen bg-background">
    <!-- Header -->
    <Header />

    <!-- Desktop Sidebar -->
    <DesktopSidebar onLogout={handleLogout} />

    <!-- Mobile Sidebar -->
    <MobileSidebar onLogout={handleLogout} />

    <!-- Right Sidebar -->
    <RightSidebar {stats} {servers} isOpen={rightSidebarOpen} onToggle={toggleRightSidebar} />

    <!-- Main Content -->
    <main class="min-h-screen pt-16 p-4 md:p-8 md:pt-20 {$sidebarCollapsed ? 'lg:ml-16' : 'lg:ml-64'} {rightSidebarOpen ? 'xl:mr-80' : 'mr-0'}">
        {#if loading}
            <div class="flex items-center justify-center py-20">
                <p class="text-muted-foreground">Loading dashboard...</p>
            </div>
        {:else}
            <!-- Header with Welcome + Add Server -->
            <div class="mb-6 flex items-center justify-between">
                <div>
                    <h1 class="text-2xl font-semibold text-foreground">
                        Welcome back, <span class="text-primary">{user?.email?.split('@')[0] || 'User'}</span>
                    </h1>
                    <p class="text-sm text-muted-foreground mt-1">
                        Global uptime at <span class="font-medium text-foreground">{formatPercent((stats.onlineServers / stats.totalServers) * 100)}</span> in the last 24h
                    </p>
                </div>
                <button
                    onclick={() => goto('/servers/new')}
                    class="inline-flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90"
                >
                    <svg class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"/>
                    </svg>
                    Add Server
                </button>
            </div>

            <!-- Dropped Metrics Alerts -->
            <DroppedMetricsAlert alerts={droppedMetricsAlerts} />

            <!-- Dashboard Stats Cards -->
            <DashboardStats {stats} />

            <!-- Dashboard Charts -->
            <DashboardCharts
                {aggregatedMetrics}
                {stats}
                {timeRange}
                onTimeRangeChange={handleTimeRangeChange}
            />

            <!-- Servers Table -->
            <div class="mb-6">
                <div class="mb-4 flex items-center justify-between">
                    <h2 class="text-lg font-semibold">Server Summary</h2>
                </div>
                <ServerTable {servers} {metricsData} />
            </div>
        {/if}
    </main>
</div>
