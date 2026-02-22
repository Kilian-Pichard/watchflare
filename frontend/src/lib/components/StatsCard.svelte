<script>
	const { title, value, trend, trendLabel, icon, compact = false } = $props();

	// Determine if trend is positive or negative
	const isPositive = $derived(trend >= 0);
	const trendColor = $derived(isPositive ? 'text-success' : 'text-destructive');
	const trendIcon = $derived(isPositive ? '↑' : '↓');
</script>

<div
	class="stats-card rounded-lg border bg-card"
	style="padding: {compact ? '0.75rem 1rem' : '1.5rem'};"
>
	<div class="flex items-center justify-between" style="align-items: {compact ? 'center' : 'flex-start'};">
		<div class="flex-1" style="display: {compact ? 'flex' : 'block'}; align-items: center; gap: {compact ? '0.75rem' : '0'};">
			<p class="text-sm text-muted-foreground" style="margin-bottom: {compact ? '0' : '0.25rem'};">{title}</p>
			<p class="font-semibold text-foreground" style="font-size: {compact ? '1.125rem' : '1.875rem'}; line-height: {compact ? '1.75rem' : '2.25rem'};">{value}</p>
			<div
				class="flex items-center gap-1 text-sm"
				style="margin-top: {compact ? '0' : '0.5rem'}; overflow: hidden; max-height: {compact ? '0' : '1.5rem'}; opacity: {compact ? '0' : '1'};"
			>
				{#if trend !== undefined}
					<span class="{trendColor} font-medium">
						{trendIcon}{Math.abs(trend).toFixed(1)}%
					</span>
					<span class="text-muted-foreground">{trendLabel || ''}</span>
				{/if}
			</div>
		</div>
		{#if icon}
			{@const Icon = icon}
			<div
				class="flex items-center justify-center rounded-lg bg-primary/10 text-primary"
				style="width: {compact ? '0' : '2.5rem'}; height: {compact ? '0' : '2.5rem'}; opacity: {compact ? '0' : '1'}; overflow: hidden;"
			>
				<Icon class="h-5 w-5" />
			</div>
		{/if}
	</div>
</div>

<style>
	.stats-card,
	.stats-card * {
		transition: all 250ms ease;
	}
</style>
