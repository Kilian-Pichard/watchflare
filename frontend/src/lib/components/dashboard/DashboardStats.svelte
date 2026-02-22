<script lang="ts">
    import StatsCard from "$lib/components/StatsCard.svelte";
    import { Server, Cpu, MemoryStick, HardDrive } from "lucide-svelte";

    interface Stats {
        totalServers: number;
        onlineServers: number;
        offlineServers: number;
        avgCPU: number;
        avgMemory: number;
        avgDisk: number;
        cpuTrend: number;
        memoryTrend: number;
        diskTrend: number;
    }

    const { stats, compact = false }: { stats: Stats; compact?: boolean } =
        $props();
</script>

<!-- 4 Stats Cards -->
<div class="mb-6 grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
    <StatsCard
        title="Active Servers"
        value="{stats.onlineServers}/{stats.totalServers}"
        trend={0}
        trendLabel="Server"
        icon={Server}
        {compact}
    />
    <StatsCard
        title="Avg CPU Load"
        value="{stats.avgCPU.toFixed(1)}%"
        trend={stats.cpuTrend}
        trendLabel="vs 24h ago"
        icon={Cpu}
        {compact}
    />
    <StatsCard
        title="Memory Usage"
        value="{stats.avgMemory.toFixed(1)}%"
        trend={stats.memoryTrend}
        trendLabel="vs 24h ago"
        icon={MemoryStick}
        {compact}
    />
    <StatsCard
        title="Disk Usage"
        value="{stats.avgDisk.toFixed(1)}%"
        trend={stats.diskTrend}
        trendLabel="vs 24h ago"
        icon={HardDrive}
        {compact}
    />
</div>
