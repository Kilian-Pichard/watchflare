<script>
	const { title, value, trend, trendLabel, icon } = $props();

	// Determine if trend is positive or negative
	const isPositive = $derived(trend >= 0);
	const trendColor = $derived(isPositive ? 'text-success' : 'text-destructive');
	const trendIcon = $derived(isPositive ? '↑' : '↓');
</script>

<div class="rounded-lg border bg-card p-6">
	<div class="flex items-start justify-between">
		<div class="flex-1">
			<p class="text-sm text-muted-foreground mb-1">{title}</p>
			<p class="text-3xl font-semibold text-foreground">{value}</p>
			{#if trend !== undefined}
				<div class="mt-2 flex items-center gap-1 text-sm">
					<span class="{trendColor} font-medium">
						{trendIcon}{Math.abs(trend).toFixed(1)}%
					</span>
					<span class="text-muted-foreground">{trendLabel || ''}</span>
				</div>
			{/if}
		</div>
		{#if icon}
			<div class="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10 text-primary">
				{#if icon}
					{@const Icon = icon}
					<Icon class="h-5 w-5" />
				{/if}
			</div>
		{/if}
	</div>
</div>
