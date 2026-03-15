import { writable, derived } from 'svelte/store';
import type { AggregatedMetric, TimeRange } from '$lib/types';
import { getAggregatedMetrics } from '$lib/api';
import { serversStore } from './servers';
import { logger } from '$lib/utils';
import { MAX_AGGREGATED_POINTS } from '$lib/constants';

interface AggregatedState {
	// Current time range metrics (for charts)
	metrics: AggregatedMetric[];
	// 24h metrics for trend calculation
	metrics24h: AggregatedMetric[];
	// Latest real-time metric (for stats cards, independent of time range)
	latestMetric: AggregatedMetric | null;
	// Current time range
	timeRange: TimeRange;
	loading: boolean;
	error: string | null;
}

function createAggregatedStore() {
	const { subscribe, set, update } = writable<AggregatedState>({
		metrics: [],
		metrics24h: [],
		latestMetric: null,
		timeRange: '1h',
		loading: false,
		error: null
	});

	// Bucket sizes in ms for each time range (must match backend intervals)
	const bucketMs: Record<string, number> = {
		'12h': 10 * 60 * 1000,
		'24h': 15 * 60 * 1000,
		'7d': 2 * 60 * 60 * 1000,
		'30d': 8 * 60 * 60 * 1000
	};

	// Guard to prevent concurrent loads (race condition from rapid SSE updates)
	let loadInFlight = false;

	// Extracted as local function so addMetricPoint can call it
	async function load(timeRange: TimeRange): Promise<void> {
		if (loadInFlight) return;
		loadInFlight = true;
		update(state => ({ ...state, loading: true, error: null, timeRange }));

		try {
			const data = await getAggregatedMetrics(timeRange);
			const metricsArray = data.metrics || [];

			update(state => ({
				...state,
				metrics: metricsArray,
				// Initialize latestMetric from loaded data if not yet set
				latestMetric: state.latestMetric || (metricsArray.length > 0 ? metricsArray[metricsArray.length - 1] : null),
				loading: false
			}));
		} catch (err) {
			const error = err instanceof Error ? err.message : 'Failed to load aggregated metrics';
			update(state => ({ ...state, loading: false, error }));
			logger.error('Failed to load aggregated metrics:', err);
		} finally {
			loadInFlight = false;
		}
	}

	return {
		subscribe,
		load,

		// Load 24h metrics for trend calculation
		async load24h(): Promise<void> {
			try {
				const data = await getAggregatedMetrics('24h');

				update(state => ({
					...state,
					metrics24h: data.metrics || []
				}));
			} catch (err) {
				logger.error('Failed to load 24h aggregated metrics for trends:', err);
			}
		},

		// Update metrics (add new point from SSE)
		addMetricPoint(metric: AggregatedMetric): void {
			let shouldReload = false;
			let reloadTimeRange: TimeRange = '1h';

			update(state => {
				// Always update latestMetric for real-time stats cards
				const newState = { ...state, latestMetric: metric };

				if (newState.timeRange === '1h') {
					// 1h view: add real-time 30s points
					let updatedMetrics = [...newState.metrics, metric];
					if (updatedMetrics.length > MAX_AGGREGATED_POINTS) {
						updatedMetrics = updatedMetrics.slice(-MAX_AGGREGATED_POINTS);
					}
					return { ...newState, metrics: updatedMetrics };
				}

				// For non-1h ranges: check if a new completed bucket exists
				const bucket = bucketMs[newState.timeRange];
				if (bucket && !newState.loading) {
					const now = Date.now();
					// Last completed bucket end (= labels use bucket end)
					const lastCompleteBucketEnd = Math.floor(now / bucket) * bucket;
					const lastPoint = newState.metrics[newState.metrics.length - 1];
					const lastPointTime = lastPoint ? new Date(lastPoint.timestamp).getTime() : 0;

					if (lastCompleteBucketEnd > lastPointTime) {
						// New completed bucket available - reload from API
						shouldReload = true;
						reloadTimeRange = newState.timeRange;
					}
				}

				return newState;
			});

			if (shouldReload) {
				load(reloadTimeRange);
			}
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
				latestMetric: null,
				timeRange: '1h',
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

// Derived store for computed stats (memoized to avoid recalculation on irrelevant store changes)
let cachedStats: ReturnType<typeof computeStats> | null = null;
let cachedLastPoint: AggregatedMetric | null = null;
let cachedFirstPoint24h: AggregatedMetric | null = null;
let cachedOnlineCount = -1;
let cachedTotalCount = -1;

function computeStats(
	lastPoint: AggregatedMetric | null,
	firstPoint24h: AggregatedMetric | null,
	totalServers: number,
	onlineServers: number
) {
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

export const dashboardStats = derived(
	[aggregatedStore, serversStore],
	([$aggregated, $servers]) => {
		// Use latestMetric (real-time SSE) for stats cards, independent of time range
		const lastPoint = $aggregated.latestMetric;
		const firstPoint24h =
			$aggregated.metrics24h.length > 0 ? $aggregated.metrics24h[0] : null;

		const activeServers = $servers.servers.filter(s => s.server.status !== 'pending');
		const onlineCount = activeServers.filter(s => s.server.status === 'online').length;
		const totalCount = activeServers.length;

		// Skip recalculation if inputs haven't changed (compare by value for SSE objects)
		if (
			cachedStats &&
			lastPoint?.timestamp === cachedLastPoint?.timestamp &&
			lastPoint?.cpu_usage_percent === cachedLastPoint?.cpu_usage_percent &&
			lastPoint?.memory_used_bytes === cachedLastPoint?.memory_used_bytes &&
			firstPoint24h?.timestamp === cachedFirstPoint24h?.timestamp &&
			onlineCount === cachedOnlineCount &&
			totalCount === cachedTotalCount
		) {
			return cachedStats;
		}

		cachedLastPoint = lastPoint;
		cachedFirstPoint24h = firstPoint24h;
		cachedOnlineCount = onlineCount;
		cachedTotalCount = totalCount;
		cachedStats = computeStats(lastPoint, firstPoint24h, totalCount, onlineCount);
		return cachedStats;
	}
);
