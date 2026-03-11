<script lang="ts">
	import UPlotChart from '$lib/components/UPlotChart.svelte';
	import { formatRate } from '$lib/chart-utils';
	import type { Metric, AggregatedMetric, TimeRange } from '$lib/types';
	import type uPlot from 'uplot';

	let { data = [], timeRange }: { data: (Metric | AggregatedMetric)[]; timeRange?: TimeRange } =
		$props();

	let chartData = $derived.by(() => {
		if (data.length === 0) return [[], [], []] as uPlot.AlignedData;
		const timestamps: number[] = [];
		const read: (number | null)[] = [];
		const write: (number | null)[] = [];
		for (const d of data) {
			timestamps.push(new Date(d.timestamp).getTime() / 1000);
			read.push(d.disk_read_bytes_per_sec);
			write.push(d.disk_write_bytes_per_sec);
		}
		return [timestamps, read, write] as uPlot.AlignedData;
	});

	const series: uPlot.Series[] = [
		{
			label: 'Read',
			stroke: 'var(--chart-1)',
			width: 2,
			value: (_u: uPlot, v: number | null) => v != null ? formatRate(v) : '—',
		},
		{
			label: 'Write',
			stroke: 'var(--chart-2)',
			width: 2,
			value: (_u: uPlot, v: number | null) => v != null ? formatRate(v) : '—',
		}
	];

	const axes: uPlot.Axis[] = [
		{},
		{
			values: (_u: uPlot, ticks: number[]) => ticks.map(v => formatRate(v)),
			size: 70,
		}
	];
</script>

{#if data.length > 0}
	<UPlotChart data={chartData} {series} {axes} />
{:else}
	<div class="h-48 sm:h-64 flex items-center justify-center text-muted-foreground">No data available</div>
{/if}
