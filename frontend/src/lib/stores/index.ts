/**
 * Central store exports for Watchflare Frontend
 * All application state management is centralized here
 */

// User store
export { userStore, currentUser, userLoading } from './user';

// Servers store
export {
	serversStore,
	servers,
	onlineServers,
	offlineServers,
	serversLoading,
	alertCount
} from './servers';

// Metrics store
export { metricsStore, metricsData } from './metrics';

// Aggregated metrics store
export {
	aggregatedStore,
	aggregatedMetrics,
	aggregatedMetrics24h,
	currentTimeRange,
	dashboardStats
} from './aggregated';

// Alerts store
export { alertsStore } from './alerts';

// Auth actions
export { authActions } from './auth';

// UI store
export { uiStore } from './ui';

// Toast store (already existed)
export { toasts } from './toasts';

// Sidebar store (already existed)
export { sidebarCollapsed, mobileMenuOpen, sidebarTransitioning } from './sidebar';

// SSE store
export {
	sseStore,
	sseConnectionState,
	sseIsConnected,
	sseIsReconnecting,
	sseLastError
} from './sse';
