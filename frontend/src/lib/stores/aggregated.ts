import { writable, derived } from 'svelte/store';
import type { AggregatedMetric, TimeRange } from '$lib/types';
import { getAggregatedMetrics } from '$lib/api';
import { serversStore } from './servers';

interface AggregatedState {
	// Current time range metrics
	metrics: AggregatedMetric[];
	// 24h metrics for trend calculation
	metrics24h: AggregatedMetric[];
	// Current time range
	timeRange: TimeRange;
	loading: boolean;
	error: string | null;
}

function createAggregatedStore() {
	const { subscribe, set, update } = writable<AggregatedState>({
		metrics: [],
		metrics24h: [],
		timeRange: '24h',
		loading: false,
		error: null
	});

	return {
		subscribe,

		// Load aggregated metrics for current time range
		async load(timeRange: TimeRange): Promise<void> {
			update(state => ({ ...state, loading: true, error: null, timeRange }));

			try {
				const data = await getAggregatedMetrics(timeRange);

				update(state => ({
					...state,
					metrics: data.metrics || [],
					loading: false
				}));
			} catch (err) {
				const error = err instanceof Error ? err.message : 'Failed to load aggregated metrics';
				update(state => ({ ...state, loading: false, error }));
				console.error('Failed to load aggregated metrics:', err);
			}
		},

		// Load 24h metrics for trend calculation
		async load24h(): Promise<void> {
			try {
				const data = await getAggregatedMetrics('24h');

				update(state => ({
					...state,
					metrics24h: data.metrics || []
				}));
			} catch (err) {
				console.error('Failed to load 24h aggregated metrics for trends:', err);
			}
		},

		// Update metrics (add new point from SSE)
		addMetricPoint(metric: AggregatedMetric): void {
			update(state => {
				// Determine bucket size in ms based on current time range
				const bucketMs: Record<TimeRange, number> = {
					'1h': 30 * 1000,        // 30s
					'12h': 10 * 60 * 1000,  // 10min
					'24h': 15 * 60 * 1000,  // 15min
					'7d': 2 * 60 * 60 * 1000, // 2h
					'30d': 8 * 60 * 60 * 1000  // 8h
				};
				const bucket = bucketMs[state.timeRange] || 30 * 1000;
				const metricTime = new Date(metric.timestamp).getTime();
				const snappedTime = new Date(Math.floor(metricTime / bucket) * bucket).toISOString();
				const snappedMetric = { ...metric, timestamp: snappedTime };

				let updatedMetrics = [...state.metrics];
				const lastPoint = updatedMetrics[updatedMetrics.length - 1];

				if (lastPoint && lastPoint.timestamp === snappedTime) {
					// Same bucket: replace last point with fresher data
					updatedMetrics[updatedMetrics.length - 1] = snappedMetric;
				} else {
					// New bucket: add new point
					updatedMetrics.push(snappedMetric);
				}

				// Keep only last 200 points to avoid memory issues
				if (updatedMetrics.length > 200) {
					updatedMetrics = updatedMetrics.slice(-200);
				}

				return {
					...state,
					metrics: updatedMetrics
				};
			});
		},

		// Change time range
		setTimeRange(timeRange: TimeRange): void {
			update(state => ({ ...state, timeRange }));
		},

		// Clear all data
		clear(): void {
			set({
				metrics: [],
				metrics24h: [],
				timeRange: '24h',
				loading: false,
				error: null
			});
		}
	};
}

export const aggregatedStore = createAggregatedStore();

// Derived stores for convenience
export const aggregatedMetrics = derived(aggregatedStore, $store => $store.metrics);
export const aggregatedMetrics24h = derived(aggregatedStore, $store => $store.metrics24h);
export const currentTimeRange = derived(aggregatedStore, $store => $store.timeRange);

// Derived store for computed stats
export const dashboardStats = derived(
	[aggregatedStore, serversStore],
	([$aggregated, $servers]) => {
		const lastPoint =
			$aggregated.metrics.length > 0
				? $aggregated.metrics[$aggregated.metrics.length - 1]
				: null;

		const firstPoint24h =
			$aggregated.metrics24h.length > 0 ? $aggregated.metrics24h[0] : null;

		const totalServers = $servers.servers.length;
		const onlineServers = $servers.servers.filter(
			s => s.server.status === 'online'
		).length;
		const avgCPU = lastPoint?.cpu_usage_percent || 0;
		const totalMemory = lastPoint?.memory_total_bytes || 0;
		const usedMemory = lastPoint?.memory_used_bytes || 0;
		const totalDisk = lastPoint?.disk_total_bytes || 0;
		const usedDisk = lastPoint?.disk_used_bytes || 0;

		// Calculate trends (comparing current to 24h ago)
		const cpuTrend =
			firstPoint24h && firstPoint24h.cpu_usage_percent > 0
				? ((avgCPU - firstPoint24h.cpu_usage_percent) / firstPoint24h.cpu_usage_percent) * 100
				: 0;

		const memoryPercent = totalMemory > 0 ? (usedMemory / totalMemory) * 100 : 0;
		const compareMemoryPercent =
			firstPoint24h && firstPoint24h.memory_total_bytes > 0
				? (firstPoint24h.memory_used_bytes / firstPoint24h.memory_total_bytes) * 100
				: memoryPercent;
		const memoryTrend =
			compareMemoryPercent > 0
				? ((memoryPercent - compareMemoryPercent) / compareMemoryPercent) * 100
				: 0;

		const diskPercent = totalDisk > 0 ? (usedDisk / totalDisk) * 100 : 0;
		const compareDiskPercent =
			firstPoint24h && firstPoint24h.disk_total_bytes > 0
				? (firstPoint24h.disk_used_bytes / firstPoint24h.disk_total_bytes) * 100
				: diskPercent;
		const diskTrend =
			compareDiskPercent > 0
				? ((diskPercent - compareDiskPercent) / compareDiskPercent) * 100
				: 0;

		return {
			totalServers,
			onlineServers,
			offlineServers: totalServers - onlineServers,
			avgCPU,
			avgMemory: memoryPercent,
			avgDisk: diskPercent,
			totalMemory,
			usedMemory,
			totalDisk,
			usedDisk,
			cpuTrend: isFinite(cpuTrend) ? cpuTrend : 0,
			memoryTrend: isFinite(memoryTrend) ? memoryTrend : 0,
			diskTrend: isFinite(diskTrend) ? diskTrend : 0
		};
	}
);
