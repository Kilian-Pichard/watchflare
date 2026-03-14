<script lang="ts">
    import { cn } from "$lib/utils";

    interface StatusInfo {
        label: string;
        color: string;
        dot: string;
    }

    let {
        title,
        value,
        subtitle,
        icon,
        trend,
        trendValue,
        variant = "default",
        status = null,
        percentage = null,
        class: className,
    }: {
        title: string;
        value: string;
        subtitle?: string;
        icon?: string;
        trend?: "up" | "down";
        trendValue?: string;
        variant?: string;
        status?: StatusInfo | null;
        percentage?: number | null;
        class?: string;
    } = $props();

    // Determine status badge based on percentage
    let statusInfo = $derived.by(() => {
        if (percentage === null || percentage === undefined) return null;

        if (percentage >= 90) {
            return {
                label: "Critical",
                color: "bg-[var(--danger)] text-white",
                dot: "bg-[var(--danger)]",
            };
        } else if (percentage >= 70) {
            return {
                label: "Warning",
                color: "bg-[var(--warning)] text-white",
                dot: "bg-[var(--warning)]",
            };
        } else {
            return {
                label: "Healthy",
                color: "bg-[var(--success)] text-white",
                dot: "bg-[var(--success)]",
            };
        }
    });

    // Custom status override
    let displayStatus = $derived(status || statusInfo);
</script>

<div
    class={cn(
        "group relative rounded-2xl border bg-card p-6 shadow-sm transition-all duration-300",
        "bg-gradient-to-br from-card to-card/80",
        className,
    )}
>
    <div class="flex items-start justify-between">
        <div class="space-y-2 flex-1">
            <div class="flex items-center gap-2 flex-wrap">
                <p class="text-sm font-medium">
                    {title}
                </p>
                {#if displayStatus}
                    <div class="flex items-center gap-1.5">
                        <div
                            class={cn(
                                "h-2 w-2 rounded-full animate-pulse",
                                displayStatus.dot,
                            )}
                        ></div>
                        <span
                            class={cn(
                                "text-xs font-semibold px-2 py-0.5 rounded-full",
                                displayStatus.color,
                            )}
                        >
                            {displayStatus.label}
                        </span>
                    </div>
                {/if}
            </div>
            <div class="flex items-baseline gap-2">
                <h3
                    class="text-3xl font-bold tracking-tight bg-gradient-to-br from-foreground to-foreground/70 bg-clip-text text-transparent"
                >
                    {value}
                </h3>
                {#if trend}
                    <span
                        class={cn(
                            "text-xs font-semibold px-2 py-1 rounded-full",
                            trend === "up"
                                ? "bg-green-500/10 text-green-600 dark:text-green-400"
                                : "bg-red-500/10 text-red-600 dark:text-red-400",
                        )}
                    >
                        {trend === "up" ? "↑" : "↓"}
                        {trendValue}
                    </span>
                {/if}
            </div>
            {#if subtitle}
                <p class="text-xs text-muted-foreground/80">{subtitle}</p>
            {/if}
            {#if percentage !== null && percentage !== undefined}
                <div class="mt-3 space-y-1.5">
                    <div class="flex items-center justify-between text-xs">
                        <span class="text-muted-foreground">Usage</span>
                        <span class="font-semibold"
                            >{percentage.toFixed(1)}%</span
                        >
                    </div>
                    <div
                        class="h-2 w-full bg-muted rounded-full overflow-hidden"
                    >
                        <div
                            class={cn(
                                "h-full transition-all duration-500 rounded-full",
                                percentage >= 90
                                    ? "bg-[var(--danger)]"
                                    : percentage >= 70
                                      ? "bg-[var(--warning)]"
                                      : "bg-[var(--success)]",
                            )}
                            style="width: {Math.min(percentage, 100)}%"
                        ></div>
                    </div>
                </div>
            {/if}
        </div>
        {#if icon}
            <div class="text-3xl">
                {icon}
            </div>
        {/if}
    </div>
</div>
