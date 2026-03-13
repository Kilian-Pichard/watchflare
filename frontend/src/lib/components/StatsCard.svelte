<script>
    const { title, value, trend, trendLabel, icon, compact = false } = $props();

    // Determine if trend is positive or negative
    const isPositive = $derived(trend >= 0);
    const trendColor = $derived(
        isPositive ? "text-success" : "text-destructive",
    );
    const trendIcon = $derived(isPositive ? "↑" : "↓");
</script>

<div
    class="stats-card rounded-lg border bg-card"
    style="padding: {compact ? '0.75rem 1rem' : '1.5rem'};"
>
    <div
        class="flex items-center justify-between gap-3"
        style="align-items: {compact ? 'center' : 'flex-start'};"
    >
        <div
            class="flex-1 max-w-fit"
            style="display: {compact
                ? 'flex'
                : 'block'}; align-items: center; gap: {compact
                ? '0.75rem'
                : '0'};"
        >
            <p
                class="text-sm text-muted-foreground min-w-30"
                style="margin-bottom: {compact ? '0' : '0.25rem'};"
            >
                {title}
            </p>
            <p
                class="font-semibold text-foreground"
                style="font-size: {compact
                    ? '1.125rem'
                    : '1.5rem'}; line-height: {compact ? '1.75rem' : '2rem'};"
            >
                {value}
            </p>
            <div
                class={`flex items-center gap-1 text-sm overflow-hidden ${compact ? "hidden mt-0 max-h-0" : "mt-2 max-h-6"}`}
            >
                {#if trend !== undefined}
                    <span class="{trendColor} font-medium">
                        {trendIcon}{Math.abs(trend).toFixed(1)}%
                    </span>
                    <span class="text-muted-foreground">{trendLabel || ""}</span
                    >
                {/if}
            </div>
        </div>
        {#if icon}
            {@const Icon = icon}
            <div
                class="flex items-center justify-center rounded-lg bg-primary/10 text-primary"
                style="width: 2.5rem; height: 2.5rem;"
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
