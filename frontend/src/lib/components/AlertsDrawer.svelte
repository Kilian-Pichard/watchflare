<script lang="ts">
    import { X, XCircle, AlertTriangle } from "lucide-svelte";
    import { servers } from "$lib/stores";
    import { generateAlerts } from "$lib/utils";
    import RightSidebar from "$lib/components/RightSidebar.svelte";

    const {
        open,
        onClose,
    }: {
        open: boolean;
        onClose: () => void;
    } = $props();

    const alerts = $derived(generateAlerts($servers));

    function getAlertColor(type: string): string {
        if (type === "critical") return "bg-destructive/10 text-destructive border-destructive/20";
        return "bg-warning/10 text-warning border-warning/20";
    }
</script>

<RightSidebar {open} {onClose}>
    <!-- Header -->
    <div class="flex items-center justify-between border-b px-6 py-4 shrink-0">
        <h2 class="text-sm font-semibold text-foreground">Active Alerts</h2>
        <div class="flex items-center gap-2">
            {#if alerts.length > 0}
                <span class="flex h-5 w-5 items-center justify-center rounded-full bg-destructive text-xs font-medium text-primary-foreground">
                    {alerts.length}
                </span>
            {/if}
            <button
                onclick={onClose}
                class="flex h-8 w-8 items-center justify-center rounded-lg text-muted-foreground hover:bg-muted hover:text-foreground transition-colors"
                aria-label="Close alerts"
            >
                <X class="h-4 w-4" />
            </button>
        </div>
    </div>

    <!-- Content -->
    <div class="flex-1 overflow-y-auto p-6">
        {#if alerts.length === 0}
            <div class="rounded-lg border border-dashed bg-muted/20 p-6 text-center">
                <svg class="mx-auto h-8 w-8 text-muted-foreground/50 mb-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
                <p class="text-xs text-muted-foreground">No alerts</p>
            </div>
        {:else}
            <div class="space-y-2">
                {#each alerts as alert}
                    <div class="rounded-lg border p-3 {getAlertColor(alert.type)}">
                        <div class="flex items-start gap-2">
                            <div class="mt-0.5">
                                {#if alert.type === "critical"}
                                    <XCircle class="h-4 w-4" />
                                {:else}
                                    <AlertTriangle class="h-4 w-4" />
                                {/if}
                            </div>
                            <div class="flex-1 min-w-0">
                                <p class="text-xs font-medium mb-0.5">{alert.server}</p>
                                <p class="text-xs opacity-90">{alert.message}</p>
                                <p class="text-xs opacity-60 mt-1">{alert.time}</p>
                            </div>
                        </div>
                    </div>
                {/each}
            </div>
        {/if}
    </div>
</RightSidebar>
