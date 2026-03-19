<script lang="ts">
    import { goto } from "$app/navigation";
    import { formatBytes, formatPercent, getStatusClass } from "$lib/utils";
    import {
        EllipsisVertical,
        Pencil,
        Pause,
        Play,
        Trash2,
    } from "lucide-svelte";
    import * as DropdownMenu from "$lib/components/ui/dropdown-menu";
    import type { ServerWithMetrics, Metric, Server } from "$lib/types";

    const {
        servers,
        latestMetrics,
        onRename,
        onPause,
        onResume,
        onDelete,
    }: {
        servers: ServerWithMetrics[];
        latestMetrics: Record<string, Metric>;
        onRename: (server: Server) => void;
        onPause: (serverId: string) => void;
        onResume: (serverId: string) => void;
        onDelete: (server: Server) => void;
    } = $props();

    let sortColumn = $state("name");
    let sortOrder = $state<"asc" | "desc">("asc");

    function handleSort(column: string) {
        if (sortColumn === column) {
            sortOrder = sortOrder === "asc" ? "desc" : "asc";
        } else {
            sortColumn = column;
            sortOrder = "desc";
        }
    }

    function getLastMetrics(serverId: string) {
        const latest = latestMetrics[serverId];
        if (!latest) {
            return {
                hasData: false,
                cpu: 0,
                memory: 0,
                disk: 0,
                load1: 0,
                load5: 0,
                load15: 0,
                netRx: 0,
                netTx: 0,
                temp: 0,
            };
        }

        return {
            hasData: true,
            cpu: latest.cpu_usage_percent || 0,
            memory:
                latest.memory_total_bytes > 0
                    ? (latest.memory_used_bytes / latest.memory_total_bytes) *
                      100
                    : 0,
            disk:
                latest.disk_total_bytes > 0
                    ? (latest.disk_used_bytes / latest.disk_total_bytes) * 100
                    : 0,
            load1: latest.load_avg_1min || 0,
            load5: latest.load_avg_5min || 0,
            load15: latest.load_avg_15min || 0,
            netRx: latest.network_rx_bytes_per_sec || 0,
            netTx: latest.network_tx_bytes_per_sec || 0,
            temp: latest.cpu_temperature_celsius || 0,
        };
    }

    function getBarColor(percent: number): string {
        if (percent >= 90) return "bg-danger";
        if (percent >= 70) return "bg-warning";
        return "bg-primary";
    }

    const sortedServers = $derived(() => {
        const sorted = [...servers].sort((a, b) => {
            let valA, valB;
            switch (sortColumn) {
                case "name":
                    valA = (a.server.name || "").toLowerCase();
                    valB = (b.server.name || "").toLowerCase();
                    break;
                case "status":
                    valA = a.server.status || "";
                    valB = b.server.status || "";
                    break;
                case "cpu": {
                    const mA = getLastMetrics(a.server.id);
                    const mB = getLastMetrics(b.server.id);
                    valA = mA.cpu;
                    valB = mB.cpu;
                    break;
                }
                case "memory": {
                    const mA = getLastMetrics(a.server.id);
                    const mB = getLastMetrics(b.server.id);
                    valA = mA.memory;
                    valB = mB.memory;
                    break;
                }
                case "disk": {
                    const mA = getLastMetrics(a.server.id);
                    const mB = getLastMetrics(b.server.id);
                    valA = mA.disk;
                    valB = mB.disk;
                    break;
                }
                case "load": {
                    const mA = getLastMetrics(a.server.id);
                    const mB = getLastMetrics(b.server.id);
                    valA = mA.load1;
                    valB = mB.load1;
                    break;
                }
                case "net": {
                    const mA = getLastMetrics(a.server.id);
                    const mB = getLastMetrics(b.server.id);
                    valA = mA.netRx + mA.netTx;
                    valB = mB.netRx + mB.netTx;
                    break;
                }
                case "temp": {
                    const mA = getLastMetrics(a.server.id);
                    const mB = getLastMetrics(b.server.id);
                    valA = mA.temp;
                    valB = mB.temp;
                    break;
                }
                default:
                    return 0;
            }
            if (valA < valB) return sortOrder === "asc" ? -1 : 1;
            if (valA > valB) return sortOrder === "asc" ? 1 : -1;
            return 0;
        });
        return sorted;
    });
</script>

{#snippet sortIcon(column)}
    {#if sortColumn === column}
        <svg class="h-3 w-3" viewBox="0 0 12 12" fill="currentColor">
            {#if sortOrder === "asc"}
                <path d="M6 2l4 5H2z" />
            {:else}
                <path d="M6 10l4-5H2z" />
            {/if}
        </svg>
    {:else}
        <svg
            class="h-3 w-3 opacity-0 group-hover:opacity-50 transition-opacity"
            viewBox="0 0 12 12"
            fill="currentColor"
        >
            <path d="M6 10l4-5H2z" />
        </svg>
    {/if}
{/snippet}

{#snippet metricBar(percent: number)}
    <div class="w-16 h-1.5 rounded-full bg-muted mt-1">
        <div
            class="h-full rounded-full {getBarColor(percent)}"
            style="width: {Math.min(percent, 100)}%"
        ></div>
    </div>
{/snippet}

<div class="rounded-lg border bg-card">
    <!-- Mobile: Cards layout -->
    <div class="md:hidden divide-y divide-border">
        {#each sortedServers() as { server }}
            {@const metrics = getLastMetrics(server.id)}
            <a
                href="/servers/{server.id}"
                class="block p-4 hover:bg-muted/20 transition-colors"
            >
                <!-- Header: name + status -->
                <div class="flex items-center justify-between">
                    <div class="flex items-center gap-2 min-w-0">
                        <span
                            class="h-2 w-2 shrink-0 rounded-full {server.status ===
                            'online'
                                ? 'bg-success'
                                : server.status === 'offline'
                                  ? 'bg-danger'
                                  : 'bg-muted-foreground'}"
                        ></span>
                        <span class="font-medium text-foreground break-all"
                            >{server.name}</span
                        >
                    </div>
                    <span
                        class="shrink-0 inline-flex items-center rounded-full border px-2 py-0.5 ml-3 text-xs font-medium {getStatusClass(
                            server.status,
                        )}"
                    >
                        {server.status}
                    </span>
                </div>
                {#if server.hostname}
                    <p class="text-xs text-muted-foreground mt-0.5 ml-4">
                        {server.hostname}
                    </p>
                {/if}

                {#if metrics.hasData}
                    <!-- < 640px: vertical bars with max-width -->
                    <div class="sm:hidden mt-3 space-y-3">
                        {#each [{ label: "CPU", value: metrics.cpu }, { label: "Mem", value: metrics.memory }, { label: "Disk", value: metrics.disk }] as { label, value }}
                            <div class="flex items-center gap-2">
                                <span
                                    class="w-8 text-xs text-muted-foreground shrink-0"
                                    >{label}</span
                                >
                                <div
                                    class="flex-1 max-w-40 h-2.5 rounded-full bg-muted"
                                >
                                    <div
                                        class="h-full rounded-full {getBarColor(
                                            value,
                                        )}"
                                        style="width: {Math.min(value, 100)}%"
                                    ></div>
                                </div>
                                <span
                                    class="w-12 text-xs text-foreground text-left shrink-0"
                                    >{formatPercent(value)}</span
                                >
                            </div>
                        {/each}
                    </div>

                    <!-- 640-768px: grid cols -->
                    <div class="hidden sm:grid my-4 grid-cols-3 gap-12">
                        {#each [{ label: "CPU", value: metrics.cpu }, { label: "Mem", value: metrics.memory }, { label: "Disk", value: metrics.disk }] as { label, value }}
                            <div>
                                <div
                                    class="flex items-center justify-between text-xs mb-1"
                                >
                                    <span class="text-muted-foreground"
                                        >{label}</span
                                    >
                                    <span class="text-foreground"
                                        >{formatPercent(value)}</span
                                    >
                                </div>
                                <div class="h-2 rounded-full bg-muted">
                                    <div
                                        class="h-full rounded-full {getBarColor(
                                            value,
                                        )}"
                                        style="width: {Math.min(value, 100)}%"
                                    ></div>
                                </div>
                            </div>
                        {/each}
                    </div>

                    <!-- Secondary metrics -->
                    <div
                        class="mt-2 flex flex-wrap items-center gap-x-4 gap-y-0.5 text-xs"
                    >
                        <span>
                            <span class="text-muted-foreground">Load</span>
                            <span class="text-foreground"
                                >{metrics.load1.toFixed(2)}
                                {metrics.load5.toFixed(2)}
                                {metrics.load15.toFixed(2)}</span
                            >
                        </span>
                        <span>
                            <span class="text-muted-foreground">Net</span>
                            <span class="text-foreground"
                                >↓{formatBytes(metrics.netRx)}/s ↑{formatBytes(
                                    metrics.netTx,
                                )}/s</span
                            >
                        </span>
                        {#if metrics.temp > 0}
                            <span>
                                <span class="text-muted-foreground">Temp</span>
                                <span class="text-foreground"
                                    >{Math.round(metrics.temp)}°C</span
                                >
                            </span>
                        {/if}
                    </div>
                {:else}
                    <p class="text-xs text-muted-foreground mt-2">
                        No metrics available
                    </p>
                {/if}
            </a>
        {/each}
    </div>

    <!-- Desktop: Table layout -->
    <div class="hidden md:block overflow-x-auto">
        <table class="w-full min-w-280">
            <colgroup>
                <col class="min-w-50" />
                <col class="w-30" />
                <col class="w-30" />
                <col class="w-30" />
                <col class="w-30" />
                <col class="w-35" />
                <col class="w-40" />
                <col class="w-17.5" />
                <col class="w-25" />
            </colgroup>
            <thead>
                <tr class="border-b bg-muted/30">
                    <th
                        scope="col"
                        class="px-4 py-2 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider"
                        onclick={() => handleSort("name")}
                    >
                        <span
                            class="group inline-flex items-center gap-1 h-8 rounded-md px-2.5 cursor-pointer select-none transition-colors hover:bg-muted hover:text-foreground"
                        >
                            Server
                            {@render sortIcon("name")}
                        </span>
                    </th>
                    <th
                        scope="col"
                        class="px-4 py-2 text-center text-xs font-medium text-muted-foreground uppercase tracking-wider"
                        onclick={() => handleSort("status")}
                    >
                        <span
                            class="group inline-flex items-center gap-1 justify-center h-8 rounded-md px-2.5 mx-auto cursor-pointer select-none transition-colors hover:bg-muted hover:text-foreground"
                        >
                            Status
                            {@render sortIcon("status")}
                        </span>
                    </th>
                    <th
                        scope="col"
                        class="px-4 py-2 text-center text-xs font-medium text-muted-foreground uppercase tracking-wider"
                        onclick={() => handleSort("cpu")}
                    >
                        <span
                            class="group inline-flex items-center gap-1 justify-center h-8 rounded-md px-2.5 mx-auto cursor-pointer select-none transition-colors hover:bg-muted hover:text-foreground"
                        >
                            CPU
                            {@render sortIcon("cpu")}
                        </span>
                    </th>
                    <th
                        scope="col"
                        class="px-4 py-2 text-center text-xs font-medium text-muted-foreground uppercase tracking-wider"
                        onclick={() => handleSort("memory")}
                    >
                        <span
                            class="group inline-flex items-center gap-1 justify-center h-8 rounded-md px-2.5 mx-auto cursor-pointer select-none transition-colors hover:bg-muted hover:text-foreground"
                        >
                            Memory
                            {@render sortIcon("memory")}
                        </span>
                    </th>
                    <th
                        scope="col"
                        class="px-4 py-2 text-center text-xs font-medium text-muted-foreground uppercase tracking-wider"
                        onclick={() => handleSort("disk")}
                    >
                        <span
                            class="group inline-flex items-center gap-1 justify-center h-8 rounded-md px-2.5 mx-auto cursor-pointer select-none transition-colors hover:bg-muted hover:text-foreground"
                        >
                            Disk
                            {@render sortIcon("disk")}
                        </span>
                    </th>
                    <th
                        scope="col"
                        class="px-4 py-2 text-center text-xs font-medium text-muted-foreground uppercase tracking-wider"
                        onclick={() => handleSort("load")}
                    >
                        <span
                            class="group inline-flex items-center gap-1 justify-center h-8 rounded-md px-2.5 mx-auto cursor-pointer select-none transition-colors hover:bg-muted hover:text-foreground"
                        >
                            Load
                            {@render sortIcon("load")}
                        </span>
                    </th>
                    <th
                        scope="col"
                        class="px-4 py-2 text-center text-xs font-medium text-muted-foreground uppercase tracking-wider"
                        onclick={() => handleSort("net")}
                    >
                        <span
                            class="group inline-flex items-center gap-1 justify-center h-8 rounded-md px-2.5 mx-auto cursor-pointer select-none transition-colors hover:bg-muted hover:text-foreground"
                        >
                            Net
                            {@render sortIcon("net")}
                        </span>
                    </th>
                    <th
                        scope="col"
                        class="px-4 py-2 text-center text-xs font-medium text-muted-foreground uppercase tracking-wider"
                        onclick={() => handleSort("temp")}
                    >
                        <span
                            class="group inline-flex items-center gap-1 justify-center h-8 rounded-md px-2.5 mx-auto cursor-pointer select-none transition-colors hover:bg-muted hover:text-foreground"
                        >
                            Temp
                            {@render sortIcon("temp")}
                        </span>
                    </th>
                    <th
                        scope="col"
                        class="px-4 py-2 text-center text-xs font-medium text-muted-foreground uppercase tracking-wider"
                    >
                    </th>
                </tr>
            </thead>
            <tbody class="divide-y divide-border">
                {#each sortedServers() as { server }}
                    {@const metrics = getLastMetrics(server.id)}
                    <tr
                        onclick={() => goto(`/servers/${server.id}`)}
                        class="hover:bg-muted/20 transition-colors cursor-pointer"
                    >
                        <!-- Server Name -->
                        <td class="px-4 py-3.5">
                            <div class="group flex flex-col">
                                <span
                                    class="font-medium text-foreground group-hover:text-primary transition-colors whitespace-nowrap"
                                >
                                    {server.name}
                                </span>
                                {#if server.hostname}
                                    <span
                                        class="text-xs text-muted-foreground whitespace-nowrap"
                                    >
                                        {server.hostname}
                                    </span>
                                {/if}
                            </div>
                        </td>

                        <!-- Status -->
                        <td class="px-4 py-3.5 text-center">
                            <span
                                class="inline-flex items-center gap-1.5 rounded-full border px-2.5 py-0.5 text-xs font-medium {getStatusClass(
                                    server.status,
                                )}"
                            >
                                <span
                                    class="h-1.5 w-1.5 rounded-full {server.status ===
                                    'online'
                                        ? 'bg-success'
                                        : server.status === 'offline'
                                          ? 'bg-danger'
                                          : 'bg-muted-foreground'}"
                                ></span>
                                {server.status}
                            </span>
                        </td>

                        <!-- CPU -->
                        <td class="px-4 py-3.5 text-center">
                            {#if metrics.hasData}
                                <div class="flex flex-col items-center">
                                    <span class="text-foreground">
                                        {formatPercent(metrics.cpu)}
                                    </span>
                                    {@render metricBar(metrics.cpu)}
                                </div>
                            {:else}
                                <span class="text-muted-foreground">-</span>
                            {/if}
                        </td>

                        <!-- Memory -->
                        <td class="px-4 py-3.5 text-center">
                            {#if metrics.hasData}
                                <div class="flex flex-col items-center">
                                    <span class="text-foreground">
                                        {formatPercent(metrics.memory)}
                                    </span>
                                    {@render metricBar(metrics.memory)}
                                </div>
                            {:else}
                                <span class="text-muted-foreground">-</span>
                            {/if}
                        </td>

                        <!-- Disk -->
                        <td class="px-4 py-3.5 text-center">
                            {#if metrics.hasData}
                                <div class="flex flex-col items-center">
                                    <span class="text-foreground">
                                        {formatPercent(metrics.disk)}
                                    </span>
                                    {@render metricBar(metrics.disk)}
                                </div>
                            {:else}
                                <span class="text-muted-foreground">-</span>
                            {/if}
                        </td>

                        <!-- Load Avg -->
                        <td class="px-4 py-3.5 text-center">
                            {#if metrics.hasData}
                                <span
                                    class="text-sm text-foreground whitespace-nowrap"
                                    >{metrics.load1.toFixed(2)}
                                    {metrics.load5.toFixed(2)}
                                    {metrics.load15.toFixed(2)}</span
                                >
                            {:else}
                                <span class="text-muted-foreground">-</span>
                            {/if}
                        </td>

                        <!-- Network -->
                        <td class="px-4 py-3.5 text-center">
                            {#if metrics.hasData}
                                <div
                                    class="flex flex-col items-center text-sm whitespace-nowrap"
                                >
                                    <span class="text-foreground"
                                        >{formatBytes(metrics.netRx)}/s ↓</span
                                    >
                                    <span class="text-foreground"
                                        >{formatBytes(metrics.netTx)}/s ↑</span
                                    >
                                </div>
                            {:else}
                                <span class="text-muted-foreground">-</span>
                            {/if}
                        </td>

                        <!-- Temperature -->
                        <td class="px-4 py-3.5 text-center">
                            {#if metrics.hasData && metrics.temp > 0}
                                <span class="text-foreground"
                                    >{Math.round(metrics.temp)}°C</span
                                >
                            {:else}
                                <span class="text-muted-foreground"
                                    >{metrics.hasData ? "" : "-"}</span
                                >
                            {/if}
                        </td>

                        <!-- Actions menu -->
                        <!-- svelte-ignore a11y_click_events_have_key_events -->
                        <td class="px-4 py-3.5 text-center" onclick={(e) => e.stopPropagation()}>
                            <DropdownMenu.Root>
                                <DropdownMenu.Trigger>
                                    {#snippet child({ props })}
                                        <button
                                            {...props}
                                            class="rounded-lg p-1.5 text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
                                            title="Server actions"
                                        >
                                            <EllipsisVertical class="h-5 w-5" />
                                        </button>
                                    {/snippet}
                                </DropdownMenu.Trigger>
                                <DropdownMenu.Content side="bottom" align="end">
                                    <DropdownMenu.Item onclick={(e) => { e.stopPropagation(); onRename(server); }}>
                                        <Pencil class="h-4 w-4" />
                                        Rename
                                    </DropdownMenu.Item>
                                    {#if server.status !== "pending"}
                                        {#if server.status === "paused"}
                                            <DropdownMenu.Item onclick={(e) => { e.stopPropagation(); onResume(server.id); }}>
                                                <Play class="h-4 w-4" />
                                                Resume
                                            </DropdownMenu.Item>
                                        {:else}
                                            <DropdownMenu.Item onclick={(e) => { e.stopPropagation(); onPause(server.id); }}>
                                                <Pause class="h-4 w-4" />
                                                Pause
                                            </DropdownMenu.Item>
                                        {/if}
                                    {/if}
                                    <DropdownMenu.Separator />
                                    <DropdownMenu.Item
                                        onclick={(e) => { e.stopPropagation(); onDelete(server); }}
                                        class="text-destructive data-highlighted:text-destructive"
                                    >
                                        <Trash2 class="h-4 w-4" />
                                        Delete
                                    </DropdownMenu.Item>
                                </DropdownMenu.Content>
                            </DropdownMenu.Root>
                        </td>
                    </tr>
                {/each}
            </tbody>
        </table>
    </div>

    {#if servers.length === 0}
        <div
            class="flex flex-col items-center justify-center py-12 text-center"
        >
            <svg
                class="h-12 w-12 text-muted-foreground/50 mb-3"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
            >
                <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    stroke-width="1.5"
                    d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2m-2-4h.01M17 16h.01"
                />
            </svg>
            <p class="text-sm text-muted-foreground">No servers found</p>
            <p class="text-xs text-muted-foreground mt-1">
                Add your first server to start monitoring
            </p>
        </div>
    {/if}
</div>
