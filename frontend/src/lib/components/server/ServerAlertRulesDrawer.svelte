<script lang="ts">
    import { Check, X } from "lucide-svelte";
    import { getServerAlertRules, upsertServerAlertRule } from "$lib/api";
    import type { AlertMetricType, EffectiveAlertRule } from "$lib/types";
    import { ALERT_METRIC_LABELS } from "$lib/types";
    import RightSidebar from "$lib/components/RightSidebar.svelte";
    import Toggle from "$lib/components/ui/Toggle.svelte";
    import Slider from "$lib/components/ui/Slider.svelte";

    const {
        serverId,
        open,
        onClose,
    }: {
        serverId: string;
        open: boolean;
        onClose: () => void;
    } = $props();

    type EditableRule = {
        metric_type: AlertMetricType;
        enabled: boolean;
        threshold: number;
        duration_minutes: number;
        is_override: boolean;
        dirty: boolean;
    };

    let rules = $state<EditableRule[]>([]);
    let loaded = $state(false);
    let loadError = $state(false);
    let saving = $state(false);
    let saveSuccess = $state(false);
    let saveError = $state("");
    let saveTimer: ReturnType<typeof setTimeout> | undefined;

    let hasLoaded = false;
    $effect(() => {
        if (open && !hasLoaded) {
            hasLoaded = true;
            loadRules();
        }
    });

    async function loadRules() {
        loaded = false;
        loadError = false;
        try {
            const res = await getServerAlertRules(serverId);
            rules = res.rules.map((r: EffectiveAlertRule) => ({
                metric_type: r.metric_type,
                enabled: r.enabled,
                threshold: r.threshold,
                duration_minutes: r.duration_minutes,
                is_override: r.is_override,
                dirty: false,
            }));
        } catch {
            loadError = true;
        }
        loaded = true;
    }

    function markDirty(metricType: AlertMetricType) {
        const rule = rules.find((r) => r.metric_type === metricType);
        if (rule) {
            rule.dirty = true;
            rule.is_override = true;
        }
    }

    async function handleSave() {
        saveError = "";
        saveSuccess = false;
        clearTimeout(saveTimer);

        const invalid = rules.find(
            (r) =>
                r.dirty &&
                r.enabled &&
                r.metric_type !== "server_down" &&
                (isNaN(r.threshold) || r.threshold === null),
        );
        if (invalid) {
            saveError = `Enter a threshold for ${ALERT_METRIC_LABELS[invalid.metric_type]}.`;
            return;
        }
        const invalidDuration = rules.find(
            (r) =>
                r.dirty &&
                r.enabled &&
                (isNaN(r.duration_minutes) || r.duration_minutes === null),
        );
        if (invalidDuration) {
            saveError = `Enter a duration for ${ALERT_METRIC_LABELS[invalidDuration.metric_type]}.`;
            return;
        }

        saving = true;
        try {
            const dirtyRules = rules.filter((r) => r.dirty);
            await Promise.all(
                dirtyRules.map((r) =>
                    upsertServerAlertRule(serverId, r.metric_type, {
                        enabled: r.enabled,
                        threshold: r.threshold,
                        duration_minutes: r.duration_minutes,
                    }),
                ),
            );
            await loadRules();
            saveSuccess = true;
            saveTimer = setTimeout(() => {
                saveSuccess = false;
            }, 3000);
        } catch (err) {
            saveError = err instanceof Error ? err.message : "Failed to save.";
        } finally {
            saving = false;
        }
    }

    const DESCRIPTIONS: Record<AlertMetricType, string> = {
        server_down: "Alert when the server stops sending heartbeats.",
        cpu_usage: "Alert when CPU usage stays above the threshold.",
        memory_usage: "Alert when RAM usage stays above the threshold.",
        disk_usage: "Alert when disk usage stays above the threshold.",
        load_avg: "Alert when the 1-min load average exceeds the threshold.",
        load_avg_5: "Alert when the 5-min load average exceeds the threshold.",
        load_avg_15:
            "Alert when the 15-min load average exceeds the threshold.",
        temperature: "Alert when CPU temperature exceeds the threshold.",
    };

    const GAUGE_MAX: Partial<Record<AlertMetricType, number>> = {
        cpu_usage: 100,
        memory_usage: 100,
        disk_usage: 100,
        temperature: 120,
    };

    const DURATION_MAX = 60;

    function thresholdUnit(metricType: AlertMetricType): string {
        if (
            metricType === "cpu_usage" ||
            metricType === "memory_usage" ||
            metricType === "disk_usage"
        )
            return "%";
        if (metricType === "temperature") return "°C";
        return "";
    }

    const hasDirty = $derived(rules.some((r) => r.dirty));
</script>

<RightSidebar {open} {onClose} size="wide">
    <!-- Header -->
    <div class="flex items-center justify-between px-6 py-5 shrink-0">
        <h2 class="text-sm font-semibold text-foreground">Alert Rules</h2>
        <button
            onclick={onClose}
            class="flex h-7 w-7 items-center justify-center rounded-lg text-muted-foreground hover:text-foreground transition-colors"
            aria-label="Close"
        >
            <X class="h-4 w-4" />
        </button>
    </div>

    <!-- Content -->
    <div class="flex-1 overflow-y-auto px-6 pb-4">
        {#if !loaded}
            <div class="flex items-center justify-center py-16">
                <p class="text-sm text-muted-foreground">Loading…</p>
            </div>
        {:else if loadError}
            <div class="flex items-center justify-center py-16">
                <p class="text-sm text-destructive">Failed to load alert rules.</p>
            </div>
        {:else}
            <div>
                {#each rules as rule (rule.metric_type)}
                    <div class="py-4 border-b border-border/50 last:border-0">
                        <!-- Title + toggle -->
                        <div class="flex items-center justify-between gap-4">
                            <span class="text-sm font-medium text-foreground"
                                >{ALERT_METRIC_LABELS[rule.metric_type]}</span
                            >
                            <Toggle
                                bind:checked={rule.enabled}
                                onchange={() => markDirty(rule.metric_type)}
                            />
                        </div>

                        <!-- Description -->
                        <p class="text-xs text-muted-foreground mt-1">
                            {DESCRIPTIONS[rule.metric_type]}
                        </p>

                        <!-- Controls -->
                        {#if rule.enabled}
                            <div
                                class="mt-4 rounded-xl bg-muted/40 px-4 py-3 space-y-4"
                            >
                                {#if rule.metric_type !== "server_down"}
                                    <!-- Threshold -->
                                    <div>
                                        <p
                                            class="text-[11px] font-medium text-muted-foreground uppercase tracking-wide mb-2"
                                        >
                                            Threshold
                                        </p>
                                        <div class="flex items-center gap-3">
                                            {#if GAUGE_MAX[rule.metric_type]}
                                                <Slider
                                                    bind:value={rule.threshold}
                                                    min={0}
                                                    max={GAUGE_MAX[rule.metric_type]}
                                                    step={1}
                                                    oninput={() => markDirty(rule.metric_type)}
                                                />
                                            {:else}
                                                <div class="flex-1"></div>
                                            {/if}
                                            <div
                                                class="flex items-center gap-1 shrink-0"
                                            >
                                                <input
                                                    type="number"
                                                    min="0"
                                                    step={rule.metric_type ===
                                                        "load_avg" ||
                                                    rule.metric_type ===
                                                        "load_avg_5" ||
                                                    rule.metric_type ===
                                                        "load_avg_15"
                                                        ? "0.1"
                                                        : "1"}
                                                    bind:value={rule.threshold}
                                                    oninput={() =>
                                                        markDirty(
                                                            rule.metric_type,
                                                        )}
                                                    class="w-14 rounded-lg border bg-background px-2 py-1 text-xs text-foreground text-right focus:outline-none focus-visible:ring-2 focus-visible:ring-primary [appearance:textfield] [&::-webkit-outer-spin-button]:appearance-none [&::-webkit-inner-spin-button]:appearance-none"
                                                />
                                                {#if thresholdUnit(rule.metric_type)}
                                                    <span
                                                        class="text-xs text-muted-foreground w-5"
                                                        >{thresholdUnit(
                                                            rule.metric_type,
                                                        )}</span
                                                    >
                                                {/if}
                                            </div>
                                        </div>
                                    </div>
                                {/if}

                                <!-- Duration -->
                                <div>
                                    <p
                                        class="text-[11px] font-medium text-muted-foreground uppercase tracking-wide mb-2"
                                    >
                                        Duration
                                    </p>
                                    <div class="flex items-center gap-3">
                                        <Slider
                                            bind:value={rule.duration_minutes}
                                            min={1}
                                            max={DURATION_MAX}
                                            step={1}
                                            oninput={() => markDirty(rule.metric_type)}
                                        />
                                        <div
                                            class="flex items-center gap-1 shrink-0"
                                        >
                                            <input
                                                type="number"
                                                min="1"
                                                max={DURATION_MAX}
                                                step="1"
                                                bind:value={
                                                    rule.duration_minutes
                                                }
                                                oninput={() =>
                                                    markDirty(rule.metric_type)}
                                                class="w-14 rounded-lg border bg-background px-2 py-1 text-xs text-foreground text-right focus:outline-none focus-visible:ring-2 focus-visible:ring-primary [appearance:textfield] [&::-webkit-outer-spin-button]:appearance-none [&::-webkit-inner-spin-button]:appearance-none"
                                            />
                                            <span
                                                class="text-xs text-muted-foreground w-5"
                                                >min</span
                                            >
                                        </div>
                                    </div>
                                </div>
                            </div>
                        {/if}
                    </div>
                {/each}
            </div>
        {/if}
    </div>

    <!-- Footer -->
    <div class="px-6 py-4 shrink-0">
        {#if saveError}
            <p class="mb-3 text-xs text-destructive">{saveError}</p>
        {/if}
        <button
            type="button"
            onclick={handleSave}
            disabled={saving || saveSuccess || !hasDirty}
            class="w-full flex items-center justify-center gap-2 rounded-xl bg-primary py-2.5 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90 disabled:opacity-40 disabled:cursor-not-allowed"
        >
            {#if saveSuccess}
                <Check class="h-4 w-4" />
                Saved
            {:else}
                {saving ? "Saving…" : "Save changes"}
            {/if}
        </button>
    </div>
</RightSidebar>
