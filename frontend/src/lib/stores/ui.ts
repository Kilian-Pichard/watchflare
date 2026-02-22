import { writable } from 'svelte/store';

interface UIState {
	loading: boolean;
	rightSidebarOpen: boolean;
	metricsCollapsed: boolean;
}

function createUIStore() {
	const { subscribe, set, update } = writable<UIState>({
		loading: false,
		rightSidebarOpen: false,
		metricsCollapsed: false
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
			update(state => ({ ...state, metricsCollapsed: !state.metricsCollapsed }));
		},

		// Reset UI state
		reset(): void {
			set({ loading: false, rightSidebarOpen: false, metricsCollapsed: false });
		}
	};
}

export const uiStore = createUIStore();
