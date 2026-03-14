import { writable, derived, get } from 'svelte/store';
import type { Metric } from '$lib/types';
import { getServerMetrics } from '$lib/api';
import { logger } from '$lib/utils';
import { MAX_METRICS_POINTS_DASHBOARD } from '$lib/constants';

interface MetricsState {
	// Map of server ID to array of metrics (for charts, time-range dependent)
	data: Record<string, Metric[]>;
	// Latest real-time metric per server (for table display, independent of time range)
	latest: Record<string, Metric>;
	loading: Record<string, boolean>;
	error: string | null;
}

function createMetricsStore() {
	const { subscribe, set, update } = writable<MetricsState>({
		data: {},
		latest: {},
		loading: {},
		error: null
	});

	return {
		subscribe,

		// Load metrics for a specific server
		async loadForServer(serverId: string, timeRange: string = '1h'): Promise<void> {
			update(state => ({
				...state,
				loading: { ...state.loading, [serverId]: true },
				error: null
			}));

			try {
				const data = await getServerMetrics(serverId, { time_range: timeRange });
				const metricsArray = data.metrics || [];

				update(state => {
					const lastPoint = metricsArray.length > 0 ? metricsArray[metricsArray.length - 1] : null;
					return {
						...state,
						data: { ...state.data, [serverId]: metricsArray },
						// Initialize latest if not yet set
						latest: lastPoint && !state.latest[serverId]
							? { ...state.latest, [serverId]: lastPoint }
							: state.latest,
						loading: { ...state.loading, [serverId]: false }
					};
				});
			} catch (err) {
				logger.error(`Failed to load metrics for server ${serverId}:`, err);

				update(state => ({
					...state,
					data: { ...state.data, [serverId]: [] },
					loading: { ...state.loading, [serverId]: false },
					error: err instanceof Error ? err.message : 'Failed to load metrics'
				}));
			}
		},

		// Load metrics for multiple servers
		async loadForServers(serverIds: string[], timeRange: string = '1h'): Promise<void> {
			const promises = serverIds.map(id => this.loadForServer(id, timeRange));
			await Promise.all(promises);
		},

		// Update metrics for a server (add new metric point from SSE)
		updateServerMetrics(serverId: string, metric: Metric): void {
			update(state => {
				const existingMetrics = state.data[serverId] || [];
				let updatedMetrics = [...existingMetrics, metric];

				// Keep only last N points per server
				if (updatedMetrics.length > MAX_METRICS_POINTS_DASHBOARD) {
					updatedMetrics = updatedMetrics.slice(-MAX_METRICS_POINTS_DASHBOARD);
				}

				return {
					...state,
					data: { ...state.data, [serverId]: updatedMetrics },
					// Always update latest for real-time display
					latest: { ...state.latest, [serverId]: metric }
				};
			});
		},

		// Get metrics for a specific server
		getForServer(serverId: string): Metric[] {
			return get({ subscribe }).data[serverId] || [];
		},

		// Clear metrics for a specific server
		clearForServer(serverId: string): void {
			update(state => {
				const newData = { ...state.data };
				delete newData[serverId];
				const newLoading = { ...state.loading };
				delete newLoading[serverId];

				return {
					...state,
					data: newData,
					loading: newLoading
				};
			});
		},

		// Clear all metrics
		clear(): void {
			set({ data: {}, latest: {}, loading: {}, error: null });
		}
	};
}

export const metricsStore = createMetricsStore();

// Derived store to get all metrics data (for charts, time-range dependent)
export const metricsData = derived(metricsStore, $store => $store.data);

// Derived store for latest real-time metric per server (for table display)
export const latestMetrics = derived(metricsStore, $store => $store.latest);
