import { writable } from 'svelte/store';
import type { DroppedMetric } from '$lib/types';
import { getDroppedMetrics } from '$lib/api';
import { logger } from '$lib/utils';

interface AlertsState {
	droppedMetrics: DroppedMetric[];
	loading: boolean;
	error: string | null;
}

function createAlertsStore() {
	const { subscribe, set, update } = writable<AlertsState>({
		droppedMetrics: [],
		loading: false,
		error: null
	});

	return {
		subscribe,

		// Load dropped metrics alerts
		async load(): Promise<void> {
			update(state => ({ ...state, loading: true, error: null }));

			try {
				const data = await getDroppedMetrics();

				update(state => ({
					...state,
					droppedMetrics: data.dropped_metrics || [],
					loading: false
				}));
			} catch (err) {
				logger.error('Failed to load dropped metrics:', err);

				update(state => ({
					...state,
					loading: false,
					error: err instanceof Error ? err.message : 'Failed to load alerts'
				}));
			}
		},

		// Clear alerts
		clear(): void {
			set({ droppedMetrics: [], loading: false, error: null });
		}
	};
}

export const alertsStore = createAlertsStore();
