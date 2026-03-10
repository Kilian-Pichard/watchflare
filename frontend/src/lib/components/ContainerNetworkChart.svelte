<script lang="ts">
	import { LineChart } from 'layerchart';
	import { scaleTime } from 'd3-scale';
	import * as ChartUI from '$lib/components/ui/chart';
	import ChartTooltip from '$lib/components/ChartTooltip.svelte';
	import { computeXDomain, filterByDomain, formatXAxis, formatRate, CHART_PADDING_RATE } from '$lib/chart-utils';
	import type { TimeRange } from '$lib/types';

	const CHART_COLORS = ['var(--chart-1)', 'var(--chart-2)', 'var(--chart-3)', 'var(--chart-4)', 'var(--chart-5)'];

	let { pivotedData = [], seriesKeys = [], containerNames = [], timeRange }: {
		pivotedData: Record<string, unknown>[];
		seriesKeys: string[];
		containerNames: string[];
		timeRange?: TimeRange;
	} = $props();

	let xDomain = $derived(computeXDomain(pivotedData as { date: Date }[], timeRange));
	let visibleData = $derived(filterByDomain(pivotedData as { date: Date }[], xDomain));

	let chartConfig = $derived(
		Object.fromEntries(
			seriesKeys.map((key, i) => [
				key,
				{ label: key, color: CHART_COLORS[Math.floor(i / 2) % CHART_COLORS.length] }
			])
		)
	);

	let series = $derived(
		seriesKeys.map((key, i) => ({
			key,
			label: key,
			color: CHART_COLORS[Math.floor(i / 2) % CHART_COLORS.length]
		}))
	);
</script>

{#if visibleData.length > 0 && series.length > 0}
	<div class="h-48 sm:h-64">
		<ChartUI.Container config={chartConfig} class="h-full w-full">
			<LineChart
				data={visibleData}
				x="date"
				xScale={scaleTime()}
				{xDomain}
				padding={CHART_PADDING_RATE}
				{series}
				props={{
					line: { class: 'stroke-2' },
					xAxis: { format: formatXAxis },
					yAxis: { format: formatRate }
				}}
			>
				{#snippet tooltip()}
					<ChartTooltip valueFormatter={(v) => formatRate(v)} />
				{/snippet}
			</LineChart>
		</ChartUI.Container>
	</div>
{:else}
	<div class="h-64 flex items-center justify-center text-muted-foreground">No data available</div>
{/if}
