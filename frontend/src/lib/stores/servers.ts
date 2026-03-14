import { writable, derived } from 'svelte/store';
import type { Server, ServerWithMetrics, ServerStatus } from '$lib/types';
import { listServers } from '$lib/api';
import { countAlerts } from '$lib/utils';

interface ServersState {
	servers: ServerWithMetrics[];
	loading: boolean;
	error: string | null;
}

function createServersStore() {
	const { subscribe, set, update } = writable<ServersState>({
		servers: [],
		loading: false,
		error: null
	});

	return {
		subscribe,

		// Load servers from API
		async load(): Promise<void> {
			update(state => ({ ...state, loading: true, error: null }));

			try {
				const data = await listServers();
				const servers = data.servers.map(server => ({
					server
				}));

				update(state => ({
					...state,
					servers,
					loading: false
				}));
			} catch (err) {
				const error = err instanceof Error ? err.message : 'Failed to load servers';
				update(state => ({ ...state, loading: false, error }));
				throw err;
			}
		},

		// Update a single server (from SSE events)
		updateServer(serverId: string, updates: Partial<Server>): void {
			update(state => ({
				...state,
				servers: state.servers.map(item =>
					item.server.id === serverId
						? { ...item, server: { ...item.server, ...updates } }
						: item
				)
			}));
		},

		// Update server status
		updateStatus(serverId: string, status: ServerStatus, lastSeen: string): void {
			update(state => ({
				...state,
				servers: state.servers.map(item =>
					item.server.id === serverId
						? {
								...item,
								server: {
									...item.server,
									status,
									last_seen: lastSeen
								}
						  }
						: item
				)
			}));
		},

		// Add a new server
		addServer(server: Server): void {
			update(state => ({
				...state,
				servers: [...state.servers, { server }]
			}));
		},

		// Remove a server
		removeServer(serverId: string): void {
			update(state => ({
				...state,
				servers: state.servers.filter(item => item.server.id !== serverId)
			}));
		},

		// Clear all servers
		clear(): void {
			set({ servers: [], loading: false, error: null });
		}
	};
}

export const serversStore = createServersStore();

// Derived stores for convenience
export const servers = derived(serversStore, $store => $store.servers);
export const onlineServers = derived(serversStore, $store =>
	$store.servers.filter(item => item.server.status === 'online')
);
export const offlineServers = derived(serversStore, $store =>
	$store.servers.filter(item => item.server.status === 'offline')
);
export const serversLoading = derived(serversStore, $store => $store.loading);
export const alertCount = derived(servers, ($servers) => countAlerts($servers));
