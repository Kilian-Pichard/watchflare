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
		const rx: (number | null)[] = [];
		const tx: (number | null)[] = [];
		for (const d of data) {
			timestamps.push(new Date(d.timestamp).getTime() / 1000);
			rx.push(d.network_rx_bytes_per_sec);
			tx.push(d.network_tx_bytes_per_sec);
		}
		return [timestamps, rx, tx] as uPlot.AlignedData;
	});

	const series: uPlot.Series[] = [
		{
			label: 'Download (RX)',
			stroke: 'var(--chart-1)',
			fill: 'var(--chart-1)',
			width: 2,
			value: (_u: uPlot, v: number | null) => v != null ? formatRate(v) : '—',
		},
		{
			label: 'Upload (TX)',
			stroke: 'var(--chart-2)',
			fill: 'var(--chart-2)',
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

<UPlotChart data={chartData} {series} {axes} {timeRange} />
