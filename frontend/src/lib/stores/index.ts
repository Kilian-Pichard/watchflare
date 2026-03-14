/**
 * Central store exports for Watchflare Frontend
 *
 * Store categories:
 *
 * DATA STORES — { subscribe, load(), clear(), ...methods }
 *   user, servers, metrics, aggregated, alerts
 *
 * UTILITY STORES — specialized patterns
 *   sidebar: raw writables + toggleSidebarWithTransition + resetSidebar
 *   toasts:  imperative add/remove/clear
 *   sse:     connection manager (connect/disconnect)
 *   ui:      UI state (right sidebar open/closed)
 *   auth:    actions (logout)
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
export { metricsStore, metricsData, latestMetrics } from './metrics';

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

// Toast store
export { toasts } from './toasts';

// Sidebar store
export {
	sidebarCollapsed,
	mobileMenuOpen,
	sidebarTransitioning,
	toggleSidebarWithTransition,
	resetSidebar
} from './sidebar';

// SSE store
export {
	sseStore,
	sseConnectionState,
	sseIsConnected,
	sseIsReconnecting,
	sseLastError
} from './sse';
