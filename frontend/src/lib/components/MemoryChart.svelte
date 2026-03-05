<script lang="ts">
	import { AreaChart } from 'layerchart';
	import { scaleTime } from 'd3-scale';
	import { formatBytes } from '$lib/utils';
	import * as ChartUI from '$lib/components/ui/chart';
	import ChartTooltip from '$lib/components/ChartTooltip.svelte';
	import { computeXDomain, filterByDomain, formatXAxis, CHART_PADDING_BYTES } from '$lib/chart-utils';
	import type { Metric, AggregatedMetric, TimeRange } from '$lib/types';

	let { data = [], timeRange }: { data: (Metric | AggregatedMetric)[]; timeRange?: TimeRange } =
		$props();

	let chartData = $derived(
		data.map((d) => ({
			date: new Date(d.timestamp),
			memory: d.memory_used_bytes
		}))
	);

	let xDomain = $derived(computeXDomain(chartData, timeRange));
	let visibleData = $derived(filterByDomain(chartData, xDomain));

	let maxMemory = $derived(
		data.length > 0 ? Math.max(...data.map((d) => d.memory_total_bytes)) : 0
	);

	const chartConfig = {
		memory: { label: 'Memory Used', color: 'var(--chart-2)' }
	};
</script>

{#if visibleData.length > 0}
	<div class="h-48 sm:h-64">
		<ChartUI.Container config={chartConfig} class="h-full w-full">
			<AreaChart
				data={visibleData}
				x="date"
				xScale={scaleTime()}
				{xDomain}
				yDomain={[0, maxMemory]}
				padding={CHART_PADDING_BYTES}
				series={[
					{
						key: 'memory',
						label: 'Memory Used',
						color: chartConfig.memory.color
					}
				]}
				props={{
					area: {
						'fill-opacity': 0.2,
						line: { class: 'stroke-2' }
					},
					xAxis: { format: formatXAxis },
					yAxis: { format: (d) => formatBytes(d) }
				}}
			>
				{#snippet tooltip()}
					<ChartTooltip valueFormatter={(v) => formatBytes(v)} />
				{/snippet}
			</AreaChart>
		</ChartUI.Container>
	</div>
{:else}
	<div class="h-64 flex items-center justify-center text-muted-foreground">No data available</div>
{/if}
