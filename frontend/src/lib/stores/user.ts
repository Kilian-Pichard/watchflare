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
				applyTheme(user.theme || 'system');

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
			try {
				await updatePreferences(timeRange, theme);

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
			} catch (err) {
				logger.error('Failed to update preferences:', err);
				throw err;
			}
		},

		// Update theme only
		async updateTheme(theme: Theme): Promise<void> {
			applyTheme(theme);

			let currentUser: User | null = null;
			const unsubscribe = subscribe(state => { currentUser = state.user; });
			unsubscribe();

			if (currentUser) {
				try {
					await updatePreferences(currentUser.default_time_range, theme);
					update(state => {
						if (state.user) {
							return { ...state, user: { ...state.user, theme } };
						}
						return state;
					});
				} catch (err) {
					logger.error('Failed to update theme:', err);
				}
			}
		},

		// Clear user data (logout)
		clear(): void {
			set({ user: null, loading: false, error: null });
		}
	};
}

export const userStore = createUserStore();

// Derived stores for convenience
export const currentUser = derived(userStore, $store => $store.user);
export const userLoading = derived(userStore, $store => $store.loading);
