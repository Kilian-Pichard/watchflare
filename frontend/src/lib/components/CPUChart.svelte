<script lang="ts">
	import { LineChart } from 'layerchart';
	import { scaleTime } from 'd3-scale';
	import * as ChartUI from '$lib/components/ui/chart';
	import ChartTooltip from '$lib/components/ChartTooltip.svelte';
	import { computeXDomain, formatXAxis, CHART_PADDING_PERCENT } from '$lib/chart-utils';
	import type { Metric, AggregatedMetric, TimeRange } from '$lib/types';

	let { data = [], timeRange }: { data: (Metric | AggregatedMetric)[]; timeRange?: TimeRange } =
		$props();

	let chartData = $derived(
		data.map((d) => ({
			date: new Date(d.timestamp),
			cpu: d.cpu_usage_percent
		}))
	);

	let xDomain = $derived(computeXDomain(chartData, timeRange));

	const chartConfig = {
		cpu: { label: 'CPU Usage', color: 'var(--chart-1)' }
	};
</script>

{#if chartData.length > 0}
	<div class="h-48 sm:h-64">
		<ChartUI.Container config={chartConfig} class="h-full w-full">
			<LineChart
				data={chartData}
				x="date"
				xScale={scaleTime()}
				{xDomain}
				yDomain={[0, 100]}
				padding={CHART_PADDING_PERCENT}
				series={[
					{
						key: 'cpu',
						label: 'CPU Usage',
						color: chartConfig.cpu.color
					}
				]}
				props={{
					line: { class: 'stroke-2 stroke-[var(--chart-1)]' },
					xAxis: { format: formatXAxis }
				}}
			>
				{#snippet tooltip()}
					<ChartTooltip valueFormatter={(v) => v.toFixed(1) + '%'} />
				{/snippet}
			</LineChart>
		</ChartUI.Container>
	</div>
{:else}
	<div class="h-64 flex items-center justify-center text-muted-foreground">No data available</div>
{/if}
