<script lang="ts">
    import { onMount, onDestroy, getContext } from 'svelte';
    import { page } from '$app/stores';
    import * as api from '$lib/api.js';
    import { sseStore } from '$lib/stores/sse';
    import { logger, TIME_RANGES } from '$lib/utils';
    import type { Metric, ContainerMetric, SSEEvent, TimeRange } from '$lib/types';
    import ServerMetricsCharts from '$lib/components/server/ServerMetricsCharts.svelte';

    function rangeSeconds(range: TimeRange): number {
        return TIME_RANGES.find(r => r.value === range)?.seconds ?? 3600;
    }

    function pruneMetricsByTime(arr: Metric[], range: TimeRange): Metric[] {
        const cutoff = Date.now() / 1000 - rangeSeconds(range) - 60;
        return arr.filter(m => new Date(m.timestamp).getTime() / 1000 >= cutoff);
    }

    const serverId = $derived($page.params.id);

    type OverviewCache = { metrics: Metric[]; containerMetrics: ContainerMetric[]; timeRange: TimeRange };
    const ctx = getContext<{
        overviewCache: OverviewCache | null;
        setOverviewCache: (data: OverviewCache) => void;
        setLatestMetric: (m: Metric | null) => void;
    }>('serverDetail');

    const cached = ctx.overviewCache;
    let metrics: Metric[] = $state(cached?.metrics ?? []);
    let containerMetrics: ContainerMetric[] = $state(cached?.containerMetrics ?? []);
    let timeRange: TimeRange = $state(cached?.timeRange ?? '1h');
    let loading = $state(!cached);
    let metricsError = $state('');
    let metricsLoadId = 0;
    let lastCacheUpdate = 0;
    let sseUnsubscribe: (() => void) | null = null;

    onMount(() => {
        sseUnsubscribe = sseStore.connect(handleSSEMessage);
    });

    onDestroy(() => {
        if (sseUnsubscribe) sseUnsubscribe();
    });

    $effect(() => {
        const range = timeRange;
        const id = serverId;
        loadMetrics(range, id);
    });

    function handleSSEMessage(event: SSEEvent) {
        if (event.type === 'metrics_update') {
            const metric = event.data;
            if (metric.server_id === serverId) {
                metrics = pruneMetricsByTime([...metrics, metric], timeRange);
                ctx.setLatestMetric(metric);
                const now = Date.now();
                if (now - lastCacheUpdate >= 5000) {
                    lastCacheUpdate = now;
                    ctx.setOverviewCache({ metrics, containerMetrics, timeRange });
                }
            }
        }
        if (event.type === 'container_metrics_update') {
            const update = event.data as { server_id: string; metrics: ContainerMetric[] };
            if (update.server_id === serverId) {
                const cutoff = Date.now() / 1000 - rangeSeconds(timeRange) - 60;
                containerMetrics = [...containerMetrics, ...update.metrics].filter(
                    m => new Date(m.timestamp).getTime() / 1000 >= cutoff,
                );
                const now = Date.now();
                if (now - lastCacheUpdate >= 5000) {
                    lastCacheUpdate = now;
                    ctx.setOverviewCache({ metrics, containerMetrics, timeRange });
                }
            }
        }
    }

    async function loadMetrics(range: TimeRange, id: string) {
        const thisLoadId = ++metricsLoadId;
        loading = true;
        metricsError = '';
        lastCacheUpdate = 0;
        try {
            const data = await api.getServerMetrics(id, { time_range: range });
            if (thisLoadId !== metricsLoadId) return;
            metrics = data.metrics || [];
            if (metrics.length > 0) ctx.setLatestMetric(metrics[metrics.length - 1]);
        } catch (err) {
            logger.error('Failed to load metrics:', err);
            if (thisLoadId === metricsLoadId) metricsError = 'Failed to load metrics';
        }
        try {
            const containerData = await api.getContainerMetrics(id, range);
            if (thisLoadId !== metricsLoadId) return;
            containerMetrics = containerData.metrics || [];
        } catch (err) {
            logger.error('Failed to load container metrics:', err);
            containerMetrics = [];
        }
        loading = false;
        ctx.setOverviewCache({ metrics, containerMetrics, timeRange: range });
    }
</script>

{#if loading}
    <div class="flex items-center justify-center py-20">
        <p class="text-sm text-muted-foreground">Loading metrics...</p>
    </div>
{:else if metricsError}
    <div class="flex items-center justify-center py-20">
        <p class="text-sm text-destructive">{metricsError}</p>
    </div>
{:else}
    <ServerMetricsCharts
        serverID={serverId}
        {metrics}
        {containerMetrics}
        bind:timeRange
    />
{/if}
