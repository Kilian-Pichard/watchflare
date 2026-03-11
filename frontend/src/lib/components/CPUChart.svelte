<script lang="ts">
	import UPlotChart from '$lib/components/UPlotChart.svelte';
	import type { Metric, AggregatedMetric, TimeRange } from '$lib/types';
	import type uPlot from 'uplot';

	let { data = [], timeRange }: { data: (Metric | AggregatedMetric)[]; timeRange?: TimeRange } =
		$props();

	let chartData = $derived.by(() => {
		if (data.length === 0) return [[], []] as uPlot.AlignedData;
		const timestamps: number[] = [];
		const cpu: (number | null)[] = [];
		for (const d of data) {
			timestamps.push(new Date(d.timestamp).getTime() / 1000);
			cpu.push(d.cpu_usage_percent);
		}
		return [timestamps, cpu] as uPlot.AlignedData;
	});

	const series: uPlot.Series[] = [
		{
			label: 'CPU Usage',
			stroke: 'var(--chart-1)',
			width: 2,
			value: (_u: uPlot, v: number | null) => v != null ? v.toFixed(1) + '%' : '—',
		}
	];

	const scales: uPlot.Scales = { y: { range: [0, 100] } };

	const axes: uPlot.Axis[] = [
		{},
		{ values: (_u: uPlot, ticks: number[]) => ticks.map(v => v + '%') }
	];
</script>

{#if data.length > 0}
	<UPlotChart data={chartData} {series} {axes} {scales} />
{:else}
	<div class="h-48 sm:h-64 flex items-center justify-center text-muted-foreground">No data available</div>
{/if}
