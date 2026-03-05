<script lang="ts">
	import * as ChartUI from '$lib/components/ui/chart';
	import { formatTooltipDate } from '$lib/chart-utils';

	let { valueFormatter }: { valueFormatter: (value: number) => string } = $props();
</script>

<ChartUI.Tooltip indicator="line">
	{#snippet formatter({ value, name, item })}
		<div class="flex text-xs">
			<div class="w-1 rounded-sm mr-2" style="background-color: {item.color}"></div>
			<div class="flex flex-col">
				<div class="text-foreground font-medium">
					{formatTooltipDate(item.label)}
				</div>
				<div class="flex items-center gap-2 mt-1">
					<div class="text-muted-foreground">
						{name}
					</div>
					<div class="font-mono font-medium text-foreground">
						{valueFormatter(value)}
					</div>
				</div>
			</div>
		</div>
	{/snippet}
</ChartUI.Tooltip>
