import { writable, derived } from 'svelte/store';
import type { User, TimeRange, Theme } from '$lib/types';
import { getCurrentUser, updatePreferences } from '$lib/api';
import { logger } from '$lib/utils';

interface UserState {
	user: User | null;
	loading: boolean;
	error: string | null;
}

let mediaQuery: MediaQueryList | null = null;
let mediaListener: ((e: MediaQueryListEvent) => void) | null = null;

function applyTheme(theme: Theme): void {
	if (typeof document === 'undefined') return;

	// Clean up previous system listener
	if (mediaListener && mediaQuery) {
		mediaQuery.removeEventListener('change', mediaListener);
		mediaListener = null;
	}

	if (theme === 'system') {
		mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
		const apply = (dark: boolean) => {
			document.documentElement.classList.toggle('dark', dark);
		};
		apply(mediaQuery.matches);
		mediaListener = (e) => apply(e.matches);
		mediaQuery.addEventListener('change', mediaListener);
	} else {
		document.documentElement.classList.toggle('dark', theme === 'dark');
	}
}

// Standalone reactive store for theme — always in sync with user preferences
export const themeStore = writable<Theme>('system');

function createUserStore() {
	const { subscribe, set, update } = writable<UserState>({
		user: null,
		loading: false,
		error: null
	});

	return {
		subscribe,

		// Load current user from API
		async load(): Promise<void> {
			update(state => ({ ...state, loading: true, error: null }));

			try {
				const userData = await getCurrentUser();
				if (!userData || !userData.user) {
					throw new Error('No user data received');
				}

				const user = userData.user;
				const theme = user.theme || 'system';
				applyTheme(theme);
				themeStore.set(theme);

				update(state => ({
					...state,
					user,
					loading: false
				}));
			} catch (err) {
				const error = err instanceof Error ? err.message : 'Failed to load user';
				update(state => ({ ...state, loading: false, error }));
				throw err;
			}
		},

		// Update user preferences
		async updatePreferences(timeRange: TimeRange, theme: Theme): Promise<void> {
			// Optimistic update
			themeStore.set(theme);
			applyTheme(theme);

			update(state => {
				if (state.user) {
					return {
						...state,
						user: {
							...state.user,
							default_time_range: timeRange,
							theme
						}
					};
				}
				return state;
			});

			try {
				await updatePreferences(timeRange, theme);
			} catch (err) {
				logger.error('Failed to update preferences:', err);
				throw err;
			}
		},

		// Update theme only
		async updateTheme(theme: Theme): Promise<void> {
			applyTheme(theme);
			themeStore.set(theme);

			let timeRange = '24h';
			update(state => {
				if (state.user) {
					timeRange = state.user.default_time_range;
					return { ...state, user: { ...state.user, theme } };
				}
				return state;
			});

			try {
				await updatePreferences(timeRange, theme);
			} catch (err) {
				logger.error('Failed to update theme:', err);
			}
		},

		// Clear user data (logout)
		clear(): void {
			set({ user: null, loading: false, error: null });
			themeStore.set('system');
		}
	};
}

export const userStore = createUserStore();

// Derived stores for convenience
export const currentUser = derived(userStore, $store => $store.user);
export const userLoading = derived(userStore, $store => $store.loading);
