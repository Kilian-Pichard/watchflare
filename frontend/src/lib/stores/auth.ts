import { goto } from '$app/navigation';
import { get } from 'svelte/store';
import { logout as apiLogout } from '$lib/api';
import { userStore, hostsStore, metricsStore, aggregatedStore, alertsStore } from '.';
import { logger } from '$lib/utils';
import { authTheme } from '$lib/stores/auth-theme';

export const authActions = {
	async logout() {
		try {
			// Persist current theme so the login page uses the same one
			const currentUser = get(userStore).user;
			if (currentUser?.theme) {
				authTheme.set(currentUser.theme);
				localStorage.setItem('wf_auth_theme', currentUser.theme);
			}
			await apiLogout();
			userStore.clear();
			hostsStore.clear();
			metricsStore.clear();
			aggregatedStore.clear();
			alertsStore.clear();
			goto('/login');
		} catch (err) {
			logger.error('Logout failed:', err);
			goto('/login');
		}
	}
};
