import { goto } from '$app/navigation';
import { logout as apiLogout } from '$lib/api';
import { userStore, serversStore, metricsStore, aggregatedStore, alertsStore } from '.';
import { logger } from '$lib/utils';

export const authActions = {
	async logout() {
		try {
			await apiLogout();
			userStore.clear();
			serversStore.clear();
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
