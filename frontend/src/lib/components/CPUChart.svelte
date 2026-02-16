<script>
    import { LineChart } from "layerchart";
    import { scaleTime } from "d3-scale";
    import * as ChartUI from "$lib/components/ui/chart";

    let { data = [] } = $props();

    // Transform data for layerchart
    let chartData = $derived(
        data.map((d) => ({
            date: new Date(d.timestamp),
            cpu: d.cpu_usage_percent,
        })),
    );

    const chartConfig = {
        cpu: { label: "CPU Usage", color: "var(--chart-1)" },
    };
</script>

{#if chartData.length > 0}
    <div class="h-48 sm:h-64">
        <ChartUI.Container config={chartConfig} class="h-full w-full">
            <LineChart
                data={chartData}
                x="date"
                xScale={scaleTime()}
                yDomain={[0, 100]}
                padding={{ left: 40, bottom: 24, top: 8, right: 8 }}
                series={[
                    {
                        key: "cpu",
                        label: "CPU Usage",
                        color: chartConfig.cpu.color,
                    },
                ]}
                props={{
                    line: { class: "stroke-2 stroke-[var(--chart-1)]" },
                    xAxis: {
                        format: (d) =>
                            d.toLocaleTimeString([], {
                                hour: "2-digit",
                                minute: "2-digit",
                            }),
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
                                            {value.toFixed(1)}%
                                        </div>
                                    </div>
                                </div>
                            </div>
                        {/snippet}
                    </ChartUI.Tooltip>
                {/snippet}
            </LineChart>
        </ChartUI.Container>
    </div>
{:else}
    <div class="h-64 flex items-center justify-center text-muted-foreground">
        No data available
    </div>
{/if}
