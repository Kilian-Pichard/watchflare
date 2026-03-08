<script lang="ts">
	import { LineChart } from 'layerchart';
	import { scaleTime } from 'd3-scale';
	import * as ChartUI from '$lib/components/ui/chart';
	import ChartTooltip from '$lib/components/ChartTooltip.svelte';
	import { computeXDomain, filterByDomain, formatXAxis, CHART_PADDING_PERCENT } from '$lib/chart-utils';
	import type { Metric, AggregatedMetric, TimeRange } from '$lib/types';

	let { data = [], timeRange }: { data: (Metric | AggregatedMetric)[]; timeRange?: TimeRange } =
		$props();

	let chartData = $derived(
		data.map((d) => ({
			date: new Date(d.timestamp),
			load1: d.load_avg_1min,
			load5: d.load_avg_5min,
			load15: d.load_avg_15min
		}))
	);

	let xDomain = $derived(computeXDomain(chartData, timeRange));
	let visibleData = $derived(filterByDomain(chartData, xDomain));

	const chartConfig = {
		load1: { label: '1 min', color: 'var(--chart-1)' },
		load5: { label: '5 min', color: 'var(--chart-2)' },
		load15: { label: '15 min', color: 'var(--chart-3)' }
	};
</script>

{#if visibleData.length > 0}
	<div class="h-48 sm:h-64">
		<ChartUI.Container config={chartConfig} class="h-full w-full">
			<LineChart
				data={visibleData}
				x="date"
				xScale={scaleTime()}
				{xDomain}
				padding={CHART_PADDING_PERCENT}
				series={[
					{ key: 'load1', label: '1 min', color: chartConfig.load1.color },
					{ key: 'load5', label: '5 min', color: chartConfig.load5.color },
					{ key: 'load15', label: '15 min', color: chartConfig.load15.color }
				]}
				props={{
					line: { class: 'stroke-2' },
					xAxis: { format: formatXAxis }
				}}
			>
				{#snippet tooltip()}
					<ChartTooltip valueFormatter={(v) => v.toFixed(2)} />
				{/snippet}
			</LineChart>
		</ChartUI.Container>
	</div>
{:else}
	<div class="h-64 flex items-center justify-center text-muted-foreground">No data available</div>
{/if}
