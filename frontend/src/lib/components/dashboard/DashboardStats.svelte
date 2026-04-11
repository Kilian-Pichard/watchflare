<script lang="ts">
    import { Server, Cpu, MemoryStick, HardDrive } from "lucide-svelte";

    interface Stats {
        totalHosts: number;
        onlineHosts: number;
        offlineHosts: number;
        avgCPU: number;
        avgMemory: number;
        avgDisk: number;
        cpuTrend: number;
        memoryTrend: number;
        diskTrend: number;
    }

    const { stats }: { stats: Stats } = $props();
</script>

<div class="mb-6 grid grid-cols-2 gap-3 sm:grid-cols-4 sm:gap-4">
    <div
        class="flex items-center gap-2.5 rounded-lg border bg-card px-3 py-2.5"
    >
        <div
            class="flex items-center justify-center rounded-md bg-primary/10 text-primary h-8 w-8 shrink-0"
        >
            <Server class="h-4 w-4" />
        </div>
        <div class="min-w-0">
            <p class="text-xs text-muted-foreground truncate">Hosts</p>
            <p class="text-sm font-semibold text-foreground">
                {stats.onlineHosts}/{stats.totalHosts}
            </p>
        </div>
    </div>
    <div
        class="flex items-center gap-2.5 rounded-lg border bg-card px-3 py-2.5"
    >
        <div
            class="flex items-center justify-center rounded-md bg-primary/10 text-primary h-8 w-8 shrink-0"
        >
            <Cpu class="h-4 w-4" />
        </div>
        <div class="min-w-0">
            <p class="text-xs text-muted-foreground truncate">CPU</p>
            <p class="text-sm font-semibold text-foreground">
                {stats.avgCPU.toFixed(1)}%
            </p>
        </div>
    </div>
    <div
        class="flex items-center gap-2.5 rounded-lg border bg-card px-3 py-2.5"
    >
        <div
            class="flex items-center justify-center rounded-md bg-primary/10 text-primary h-8 w-8 shrink-0"
        >
            <MemoryStick class="h-4 w-4" />
        </div>
        <div class="min-w-0">
            <p class="text-xs text-muted-foreground truncate">Memory</p>
            <p class="text-sm font-semibold text-foreground">
                {stats.avgMemory.toFixed(1)}%
            </p>
        </div>
    </div>
    <div
        class="flex items-center gap-2.5 rounded-lg border bg-card px-3 py-2.5"
    >
        <div
            class="flex items-center justify-center rounded-md bg-primary/10 text-primary h-8 w-8 shrink-0"
        >
            <HardDrive class="h-4 w-4" />
        </div>
        <div class="min-w-0">
            <p class="text-xs text-muted-foreground truncate">Disk</p>
            <p class="text-sm font-semibold text-foreground">
                {stats.avgDisk.toFixed(1)}%
            </p>
        </div>
    </div>
</div>
