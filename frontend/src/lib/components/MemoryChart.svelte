<script>
    import { AreaChart } from "layerchart";
    import { scaleTime } from "d3-scale";
    import { formatBytes } from "$lib/utils";
    import * as ChartUI from "$lib/components/ui/chart";

    let { data = [] } = $props();

    // Transform data for layerchart
    let chartData = $derived(
        data.map((d) => ({
            date: new Date(d.timestamp),
            memory: d.memory_used_bytes,
        })),
    );

    // Calculate max for y-axis
    let maxMemory = $derived(
        data.length > 0
            ? Math.max(...data.map((d) => d.memory_total_bytes))
            : 0,
    );

    const chartConfig = {
        memory: { label: "Memory Used", color: "var(--chart-2)" },
    };
</script>

{#if chartData.length > 0}
    <div class="h-48 sm:h-64">
        <ChartUI.Container config={chartConfig} class="h-full w-full">
            <AreaChart
                data={chartData}
                x="date"
                xScale={scaleTime()}
                yDomain={[0, maxMemory]}
                padding={{ left: 70, bottom: 24, top: 8, right: 8 }}
                series={[
                    {
                        key: "memory",
                        label: "Memory Used",
                        color: chartConfig.memory.color,
                    },
                ]}
                props={{
                    area: {
                        "fill-opacity": 0.2,
                        line: { class: "stroke-2" },
                    },
                    xAxis: {
                        format: (d) =>
                            d.toLocaleTimeString([], {
                                hour: "2-digit",
                                minute: "2-digit",
                            }),
                    },
                    yAxis: {
                        format: (d) => formatBytes(d),
                    },
                }}
            >
                {#snippet tooltip()}
                    <ChartUI.Tooltip indicator="line">
                        {#snippet formatter({ value, name, item })}
                            <div class="flex text-xs">
                                <div
                                    class="w-1 bg-[{item.color}] rounded-sm mr-2"
                                ></div>
                                <div class="flex flex-col">
                                    <div class="text-foreground font-medium">
                                        {item.label.toLocaleDateString(
                                            "fr-FR",
                                            {
                                                day: "numeric",
                                                month: "short",
                                                hour: "2-digit",
                                                minute: "2-digit",
                                                second: "2-digit",
                                            },
                                        )}
                                    </div>
                                    <div class="flex items-center gap-2 mt-1">
                                        <div class="text-muted-foreground">
                                            {name}
                                        </div>
                                        <div
                                            class="font-mono font-medium text-foreground"
                                        >
                                            {formatBytes(value)}
                                        </div>
                                    </div>
                                </div>
                            </div>
                        {/snippet}
                    </ChartUI.Tooltip>
                {/snippet}
            </AreaChart>
        </ChartUI.Container>
    </div>
{:else}
    <div class="h-64 flex items-center justify-center text-muted-foreground">
        No data available
    </div>
{/if}
