<script lang="ts">
    import { onMount, onDestroy } from 'svelte';
    import { page } from '$app/stores';
    import * as api from '$lib/api.js';
    import { sseStore } from '$lib/stores/sse';
    import { logger } from '$lib/utils';
    import type { Metric, ContainerMetric, SSEEvent, TimeRange } from '$lib/types';
    import ServerMetricsCharts from '$lib/components/server/ServerMetricsCharts.svelte';

    const TIME_RANGE_SECONDS: Record<string, number> = {
        '1h': 3600, '12h': 43200, '24h': 86400, '7d': 604800, '30d': 2592000,
    };

    function pruneMetricsByTime(arr: Metric[], range: TimeRange): Metric[] {
        const cutoff = Date.now() / 1000 - (TIME_RANGE_SECONDS[range] || 3600) - 60;
        return arr.filter(m => new Date(m.timestamp).getTime() / 1000 >= cutoff);
    }

    const serverId = $derived($page.params.id);

    let metrics: Metric[] = $state([]);
    let containerMetrics: ContainerMetric[] = $state([]);
    let latestMetric: Metric | null = $state(null);
    let timeRange: TimeRange = $state('1h');
    let metricsLoadId = 0;
    let sseUnsubscribe: (() => void) | null = null;

    onMount(() => {
        sseUnsubscribe = sseStore.connect(handleSSEMessage);
        loadMetrics();
    });

    onDestroy(() => {
        if (sseUnsubscribe) sseUnsubscribe();
    });

    function handleSSEMessage(event: SSEEvent) {
        if (event.type === 'metrics_update') {
            const metric = event.data;
            if (metric.server_id === serverId) {
                latestMetric = metric;
                metrics = pruneMetricsByTime([...metrics, metric], timeRange);
            }
        }
        if (event.type === 'container_metrics_update') {
            const update = event.data as { server_id: string; metrics: ContainerMetric[] };
            if (update.server_id === serverId) {
                const cutoff = Date.now() / 1000 - (TIME_RANGE_SECONDS[timeRange] || 3600) - 60;
                containerMetrics = [...containerMetrics, ...update.metrics].filter(
                    m => new Date(m.timestamp).getTime() / 1000 >= cutoff,
                );
            }
        }
    }

    async function loadMetrics() {
        const thisLoadId = ++metricsLoadId;
        try {
            const data = await api.getServerMetrics(serverId, { time_range: timeRange });
            if (thisLoadId !== metricsLoadId) return;
            metrics = data.metrics || [];
            if (!latestMetric && metrics.length > 0) {
                latestMetric = metrics[metrics.length - 1];
            }
        } catch (err) {
            logger.error('Failed to load metrics:', err);
        }
        try {
            const containerData = await api.getContainerMetrics(serverId, timeRange);
            containerMetrics = containerData.metrics || [];
        } catch (err) {
            logger.error('Failed to load container metrics:', err);
            containerMetrics = [];
        }
    }

    function handleTimeRangeChange() {
        loadMetrics();
    }
</script>

<ServerMetricsCharts
    serverID={serverId}
    {metrics}
    {containerMetrics}
    bind:timeRange
    onTimeRangeChange={handleTimeRangeChange}
/>
