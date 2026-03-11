<script lang="ts">
	import UPlotChart from '$lib/components/UPlotChart.svelte';
	import { formatRate } from '$lib/chart-utils';
	import type { TimeRange } from '$lib/types';
	import type uPlot from 'uplot';

	const CHART_COLORS = ['var(--chart-1)', 'var(--chart-2)', 'var(--chart-3)', 'var(--chart-4)', 'var(--chart-5)'];

	let { pivotedData = [], seriesKeys = [], timeRange }: {
		pivotedData: Record<string, unknown>[];
		seriesKeys: string[];
		timeRange?: TimeRange;
	} = $props();

	let chartData = $derived.by(() => {
		if (pivotedData.length === 0 || seriesKeys.length === 0) return [[]] as uPlot.AlignedData;
		const timestamps: number[] = [];
		const columns: (number | null)[][] = seriesKeys.map(() => []);
		for (const row of pivotedData) {
			timestamps.push((row.date as Date).getTime() / 1000);
			for (let i = 0; i < seriesKeys.length; i++) {
				const val = row[seriesKeys[i]];
				columns[i].push(val != null ? val as number : null);
			}
		}
		return [timestamps, ...columns] as uPlot.AlignedData;
	});

	let series = $derived(
		seriesKeys.map((key, i): uPlot.Series => ({
			label: key,
			stroke: CHART_COLORS[Math.floor(i / 2) % CHART_COLORS.length],
			width: 2,
			value: (_u: uPlot, v: number | null) => v != null ? formatRate(v) : '—',
		}))
	);

	const axes: uPlot.Axis[] = [
		{},
		{
			values: (_u: uPlot, ticks: number[]) => ticks.map(v => formatRate(v)),
			size: 70,
		}
	];

	let hasData = $derived(pivotedData.length > 0 && seriesKeys.length > 0);
</script>

{#if hasData}
	<UPlotChart data={chartData} {series} {axes} />
{:else}
	<div class="h-48 sm:h-64 flex items-center justify-center text-muted-foreground">No data available</div>
{/if}
