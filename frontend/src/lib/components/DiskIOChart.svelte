<script lang="ts">
	import { LineChart } from 'layerchart';
	import { scaleTime } from 'd3-scale';
	import * as ChartUI from '$lib/components/ui/chart';
	import ChartTooltip from '$lib/components/ChartTooltip.svelte';
	import { computeXDomain, filterByDomain, formatXAxis, formatRate, CHART_PADDING_RATE } from '$lib/chart-utils';
	import type { Metric, AggregatedMetric, TimeRange } from '$lib/types';

	let { data = [], timeRange }: { data: (Metric | AggregatedMetric)[]; timeRange?: TimeRange } =
		$props();

	let chartData = $derived(
		data.map((d) => ({
			date: new Date(d.timestamp),
			read: d.disk_read_bytes_per_sec,
			write: d.disk_write_bytes_per_sec
		}))
	);

	let xDomain = $derived(computeXDomain(chartData, timeRange));
	let visibleData = $derived(filterByDomain(chartData, xDomain));

	const chartConfig = {
		read: { label: 'Read', color: 'var(--chart-1)' },
		write: { label: 'Write', color: 'var(--chart-2)' }
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
				padding={CHART_PADDING_RATE}
				series={[
					{ key: 'read', label: 'Read', color: chartConfig.read.color },
					{ key: 'write', label: 'Write', color: chartConfig.write.color }
				]}
				props={{
					line: { class: 'stroke-2' },
					xAxis: { format: formatXAxis },
					yAxis: { format: formatRate }
				}}
			>
				{#snippet tooltip()}
					<ChartTooltip valueFormatter={formatRate} />
				{/snippet}
			</LineChart>
		</ChartUI.Container>
	</div>
{:else}
	<div class="h-64 flex items-center justify-center text-muted-foreground">No data available</div>
{/if}
