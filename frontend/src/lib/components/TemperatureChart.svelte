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
		data
			.filter((d) => d.cpu_temperature_celsius > 0)
			.map((d) => ({
				date: new Date(d.timestamp),
				temp: d.cpu_temperature_celsius
			}))
	);

	let xDomain = $derived(computeXDomain(chartData, timeRange));
	let visibleData = $derived(filterByDomain(chartData, xDomain));

	const chartConfig = {
		temp: { label: 'CPU Temperature', color: 'var(--chart-1)' }
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
				yDomain={[0, 120]}
				padding={CHART_PADDING_PERCENT}
				series={[
					{
						key: 'temp',
						label: 'CPU Temp',
						color: chartConfig.temp.color
					}
				]}
				props={{
					line: { class: 'stroke-2 stroke-[var(--chart-1)]' },
					xAxis: { format: formatXAxis }
				}}
			>
				{#snippet tooltip()}
					<ChartTooltip valueFormatter={(v) => v.toFixed(1) + '°C'} />
				{/snippet}
			</LineChart>
		</ChartUI.Container>
	</div>
{:else}
	<div class="h-64 flex items-center justify-center text-muted-foreground">No data available</div>
{/if}
