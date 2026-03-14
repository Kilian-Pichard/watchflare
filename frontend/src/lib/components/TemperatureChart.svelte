<script lang="ts">
	import UPlotChart from '$lib/components/UPlotChart.svelte';
	import type { Metric, AggregatedMetric, TimeRange } from '$lib/types';
	import type uPlot from 'uplot';

	let { data = [], timeRange }: { data: (Metric | AggregatedMetric)[]; timeRange?: TimeRange } =
		$props();

	let chartData = $derived.by(() => {
		const filtered = data.filter(d => d.cpu_temperature_celsius > 0);
		if (filtered.length === 0) return [[], []] as uPlot.AlignedData;
		const timestamps: number[] = [];
		const temp: (number | null)[] = [];
		for (const d of filtered) {
			timestamps.push(new Date(d.timestamp).getTime() / 1000);
			temp.push(d.cpu_temperature_celsius);
		}
		return [timestamps, temp] as uPlot.AlignedData;
	});

	const series: uPlot.Series[] = [
		{
			label: 'CPU Temp',
			stroke: 'var(--chart-1)',
			fill: 'var(--chart-1)',
			width: 2,
			value: (_u: uPlot, v: number | null) => v != null ? v.toFixed(1) + '°C' : '—',
		}
	];

	const scales: uPlot.Scales = { y: { range: [0, 120] } };

	const axes: uPlot.Axis[] = [
		{},
		{ values: (_u: uPlot, ticks: number[]) => ticks.map(v => v + '°C') }
	];

</script>

<UPlotChart data={chartData} {series} {axes} {scales} {timeRange} />
