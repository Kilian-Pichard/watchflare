<script>
	import { cn } from '$lib/utils';

	let {
		title,
		value,
		subtitle,
		icon,
		trend,
		trendValue,
		variant = 'default',
		class: className
	} = $props();
</script>

<div
	class={cn(
		'group rounded-2xl border bg-card p-6 shadow-sm transition-all duration-300',
		'hover:shadow-lg hover:scale-[1.02] hover:border-primary/20',
		'bg-gradient-to-br from-card to-card/80',
		className
	)}
>
	<div class="flex items-start justify-between">
		<div class="space-y-2 flex-1">
			<p class="text-sm font-medium text-muted-foreground transition-colors group-hover:text-foreground/80">
				{title}
			</p>
			<div class="flex items-baseline gap-2">
				<h3 class="text-3xl font-bold tracking-tight bg-gradient-to-br from-foreground to-foreground/70 bg-clip-text text-transparent">
					{value}
				</h3>
				{#if trend}
					<span
						class={cn(
							'text-xs font-semibold px-2 py-1 rounded-full',
							trend === 'up'
								? 'bg-green-500/10 text-green-600 dark:text-green-400'
								: 'bg-red-500/10 text-red-600 dark:text-red-400'
						)}
					>
						{trend === 'up' ? '↑' : '↓'} {trendValue}
					</span>
				{/if}
			</div>
			{#if subtitle}
				<p class="text-xs text-muted-foreground/80">{subtitle}</p>
			{/if}
		</div>
		{#if icon}
			<div class="text-3xl opacity-50 group-hover:opacity-70 transition-opacity">
				{icon}
			</div>
		{/if}
	</div>
</div>
