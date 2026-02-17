import { writable, derived } from 'svelte/store';
import type { User, TimeRange } from '$lib/types';
import { getCurrentUser, updatePreferences } from '$lib/api';
import { logger } from '$lib/utils';

interface UserState {
	user: User | null;
	loading: boolean;
	error: string | null;
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

				update(state => ({
					...state,
					user: userData.user,
					loading: false
				}));
			} catch (err) {
				const error = err instanceof Error ? err.message : 'Failed to load user';
				update(state => ({ ...state, loading: false, error }));
				throw err;
			}
		},

		// Update user preferences
		async updatePreferences(timeRange: TimeRange, theme: string): Promise<void> {
			try {
				await updatePreferences(timeRange, theme);

				update(state => {
					if (state.user) {
						return {
							...state,
							user: {
								...state.user,
								default_time_range: timeRange
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
