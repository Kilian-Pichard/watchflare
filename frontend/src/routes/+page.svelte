<script>
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
    import { formatBytes, formatPercent } from "$lib/utils";
    import { toasts } from "$lib/stores/toasts";
    import { sidebarCollapsed } from "$lib/stores/sidebar";
    import DesktopSidebar from "$lib/components/DesktopSidebar.svelte";
    import MobileSidebar from "$lib/components/MobileSidebar.svelte";
    import Header from "$lib/components/Header.svelte";
    import ServerTable from "$lib/components/ServerTable.svelte";
    import StatsCard from "$lib/components/StatsCard.svelte";
    import RightSidebar from "$lib/components/RightSidebar.svelte";
    import TimeRangeSelector from "$lib/components/TimeRangeSelector.svelte";
    import CPUChart from "$lib/components/CPUChart.svelte";
    import MemoryChart from "$lib/components/MemoryChart.svelte";

    // State
    let user = $state(null);
    let servers = $state([]);
    let timeRange = $state("24h");
    let loading = $state(true);
    let metricsData = $state({}); // { serverId: [metrics] }
    let sseDisconnect = null;
    let droppedMetricsAlerts = $state([]);
    let aggregatedMetrics = $state([]);
    let aggregatedMetrics24h = $state([]); // Metrics for 24h trend calculation
    let rightSidebarOpen = $state(false); // Right sidebar toggle state

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

    function formatDuration(nanoseconds) {
        const seconds = nanoseconds / 1_000_000_000;
        const hours = Math.floor(seconds / 3600);
        const minutes = Math.floor((seconds % 3600) / 60);

        if (hours > 0) {
            return `${hours}h${minutes > 0 ? ` ${minutes}min` : ""}`;
        } else if (minutes > 0) {
            return `${minutes}min`;
        } else {
            return `${Math.floor(seconds)}s`;
        }
    }

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
            {#if droppedMetricsAlerts.length > 0}
                <div class="mb-6 rounded-lg border border-warning bg-warning/5 p-4">
                    <h3 class="mb-3 flex items-center gap-2 text-sm font-semibold text-warning">
                        <svg class="h-4 w-4" fill="currentColor" viewBox="0 0 20 20">
                            <path fill-rule="evenodd" d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z" clip-rule="evenodd"/>
                        </svg>
                        Dropped Metrics
                    </h3>
                    {#each droppedMetricsAlerts as alert}
                        <div class="mb-2 last:mb-0 rounded-md bg-background p-3">
                            <p class="text-sm font-medium">
                                <strong>{alert.hostname}</strong> dropped <strong>{alert.total_dropped} metrics</strong>
                            </p>
                            <p class="text-xs text-muted-foreground mt-1">
                                Backend unavailable for {formatDuration(alert.downtime_duration)}
                                ({new Date(alert.first_dropped_at).toLocaleString()} → {new Date(alert.last_dropped_at).toLocaleString()})
                            </p>
                        </div>
                    {/each}
                </div>
            {/if}

            <!-- 4 Stats Cards -->
            <div class="mb-6 grid gap-4 md:grid-cols-2 lg:grid-cols-4">
                <StatsCard
                    title="Active Servers"
                    value="{stats.onlineServers}/{stats.totalServers}"
                    trend={0}
                    trendLabel="Server"
                    icon='<svg class="h-5 w-5 text-primary" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2m-2-4h.01M17 16h.01"/></svg>'
                />
                <StatsCard
                    title="Avg CPU Load"
                    value="{stats.avgCPU.toFixed(1)}%"
                    trend={stats.cpuTrend}
                    trendLabel="vs 24h ago"
                    icon='<svg class="h-5 w-5 text-primary" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 3v2m6-2v2M9 19v2m6-2v2M5 9H3m2 6H3m18-6h-2m2 6h-2M7 19h10a2 2 0 002-2V7a2 2 0 00-2-2H7a2 2 0 00-2 2v10a2 2 0 002 2zM9 9h6v6H9V9z"/></svg>'
                />
                <StatsCard
                    title="Memory Usage"
                    value="{stats.avgMemory.toFixed(1)}%"
                    trend={stats.memoryTrend}
                    trendLabel="vs 24h ago"
                    icon='<svg class="h-5 w-5 text-primary" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4"/></svg>'
                />
                <StatsCard
                    title="Disk Usage"
                    value="{stats.avgDisk.toFixed(1)}%"
                    trend={stats.diskTrend}
                    trendLabel="vs 24h ago"
                    icon='<svg class="h-5 w-5 text-primary" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4M4 7c0-2.21 3.582-4 8-4s8 1.79 8 4m0 5c0 2.21-3.582 4-8 4s-8-1.79-8-4"/></svg>'
                />
            </div>

            <!-- Central Charts (CPU + Memory only) -->
            <div class="mb-6">
                <div class="mb-4 flex items-center justify-between">
                    <h2 class="text-lg font-semibold">Global Metrics</h2>
                    <TimeRangeSelector
                        bind:value={timeRange}
                        onValueChange={handleTimeRangeChange}
                    />
                </div>
                <div class="grid gap-4 lg:grid-cols-2">
                    <!-- CPU Chart -->
                    <div class="rounded-lg border bg-card p-4">
                        <div class="mb-3 flex items-center justify-between">
                            <h3 class="text-sm font-medium">CPU Usage</h3>
                            <span class="text-xs text-muted-foreground">
                                {formatPercent(stats.avgCPU)}
                            </span>
                        </div>
                        <CPUChart data={aggregatedMetrics} />
                    </div>

                    <!-- Memory Chart -->
                    <div class="rounded-lg border bg-card p-4">
                        <div class="mb-3 flex items-center justify-between">
                            <h3 class="text-sm font-medium">Memory Usage</h3>
                            <span class="text-xs text-muted-foreground">
                                {formatBytes(stats.usedMemory)} / {formatBytes(stats.totalMemory)}
                            </span>
                        </div>
                        <MemoryChart data={aggregatedMetrics} />
                    </div>
                </div>
            </div>

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
