import { writable } from 'svelte/store';

interface UIState {
	loading: boolean;
	rightSidebarOpen: boolean;
	metricsCollapsed: boolean;
}

function loadMetricsCollapsed(): boolean {
	if (typeof localStorage === 'undefined') return false;
	return localStorage.getItem('metricsCollapsed') === 'true';
}

function saveMetricsCollapsed(value: boolean): void {
	if (typeof localStorage === 'undefined') return;
	localStorage.setItem('metricsCollapsed', String(value));
}

function createUIStore() {
	const { subscribe, set, update } = writable<UIState>({
		loading: false,
		rightSidebarOpen: false,
		metricsCollapsed: loadMetricsCollapsed()
	});

	return {
		subscribe,

		// Set loading state
		setLoading(loading: boolean): void {
			update(state => ({ ...state, loading }));
		},

		// Toggle right sidebar
		toggleRightSidebar(): void {
			update(state => ({ ...state, rightSidebarOpen: !state.rightSidebarOpen }));
		},

		// Set right sidebar state
		setRightSidebar(open: boolean): void {
			update(state => ({ ...state, rightSidebarOpen: open }));
		},

		// Toggle metrics collapsed
		toggleMetricsCollapsed(): void {
			update(state => {
				const next = !state.metricsCollapsed;
				saveMetricsCollapsed(next);
				return { ...state, metricsCollapsed: next };
			});
		},

		// Reset UI state
		reset(): void {
			set({ loading: false, rightSidebarOpen: false, metricsCollapsed: false });
		}
	};
}

export const uiStore = createUIStore();
