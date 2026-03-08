<script lang="ts">
	import { Tooltip, getTooltipContext } from 'layerchart';
	import { formatTooltipDate } from '$lib/chart-utils';
	import { useChart } from '$lib/components/ui/chart/chart-utils';
	import { getPayloadConfigFromPayload } from '$lib/components/ui/chart/chart-utils';

	let { valueFormatter }: { valueFormatter: (value: number) => string } = $props();

	const chart = useChart();
	const tooltipCtx = getTooltipContext();
</script>

<Tooltip.Root variant="none">
	<div class="rounded-lg border border-border/50 bg-background px-2.5 py-1.5 text-xs shadow-xl" style="width: max-content;">
		{#if tooltipCtx.payload?.length}
			<div class="font-medium text-foreground mb-1.5">
				{formatTooltipDate(tooltipCtx.payload[0].label as Date)}
			</div>
			{#each tooltipCtx.payload as item}
				{@const key = `${item.key || item.name || 'value'}`}
				{@const itemConfig = getPayloadConfigFromPayload(chart.config, item, key)}
				<div class="flex items-center gap-2 py-0.5">
					<div class="w-1 shrink-0 self-stretch rounded-sm" style="background-color: {item.color}"></div>
					<span class="text-muted-foreground">{itemConfig?.label || item.name}</span>
					<span class="ml-auto font-mono font-medium text-foreground pl-4">{valueFormatter(item.value)}</span>
				</div>
			{/each}
		{/if}
	</div>
</Tooltip.Root>
