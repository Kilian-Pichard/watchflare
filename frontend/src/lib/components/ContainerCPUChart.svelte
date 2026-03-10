<script lang="ts">
	import { LineChart } from 'layerchart';
	import { scaleTime } from 'd3-scale';
	import * as ChartUI from '$lib/components/ui/chart';
	import ChartTooltip from '$lib/components/ChartTooltip.svelte';
	import { computeXDomain, filterByDomain, formatXAxis, CHART_PADDING_PERCENT } from '$lib/chart-utils';
	import type { ContainerMetric, TimeRange } from '$lib/types';

	const CHART_COLORS = ['var(--chart-1)', 'var(--chart-2)', 'var(--chart-3)', 'var(--chart-4)', 'var(--chart-5)'];

	let { data = [], timeRange }: { data: ContainerMetric[]; timeRange?: TimeRange } = $props();

	// Get unique container names in order of first appearance
	let containerNames = $derived(
		[...new Set(data.map((d) => d.container_name))]
	);

	// Pivot data: group by timestamp, one key per container
	let chartData = $derived(
		(() => {
			const byTimestamp = new Map<string, Record<string, unknown>>();
			for (const d of data) {
				const ts = d.timestamp;
				if (!byTimestamp.has(ts)) {
					byTimestamp.set(ts, { date: new Date(ts) });
				}
				const row = byTimestamp.get(ts)!;
				row[d.container_name] = d.cpu_percent;
			}
			return [...byTimestamp.values()].sort((a, b) => (a.date as Date).getTime() - (b.date as Date).getTime());
		})()
	);

	let xDomain = $derived(computeXDomain(chartData as { date: Date }[], timeRange));
	let visibleData = $derived(filterByDomain(chartData as { date: Date }[], xDomain));

	let chartConfig = $derived(
		Object.fromEntries(
			containerNames.map((name, i) => [
				name,
				{ label: name, color: CHART_COLORS[i % CHART_COLORS.length] }
			])
		)
	);

	let series = $derived(
		containerNames.map((name, i) => ({
			key: name,
			label: name,
			color: CHART_COLORS[i % CHART_COLORS.length]
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
				yDomain={[0, undefined]}
				padding={CHART_PADDING_PERCENT}
				{series}
				props={{
					line: { class: 'stroke-2' },
					xAxis: { format: formatXAxis },
					yAxis: { format: (v: number) => v.toFixed(0) + '%' }
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
