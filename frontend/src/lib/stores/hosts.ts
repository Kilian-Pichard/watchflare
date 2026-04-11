import { writable, derived } from 'svelte/store';
import type { Host, HostWithMetrics, HostStatus } from '$lib/types';
import { listHosts } from '$lib/api';

interface HostsState {
	hosts: HostWithMetrics[];
	loading: boolean;
	error: string | null;
}

function createHostsStore() {
	const { subscribe, set, update } = writable<HostsState>({
		hosts: [],
		loading: false,
		error: null
	});

	return {
		subscribe,

		// Load hosts from API
		async load(): Promise<void> {
			update(state => ({ ...state, loading: true, error: null }));

			try {
				const data = await listHosts();
				const hosts = data.hosts.map(host => ({
					host
				}));

				update(state => ({
					...state,
					hosts,
					loading: false
				}));
			} catch (err) {
				const error = err instanceof Error ? err.message : 'Failed to load hosts';
				update(state => ({ ...state, loading: false, error }));
				throw err;
			}
		},

		// Update a single host (from SSE events)
		updateHost(hostId: string, updates: Partial<Host>): void {
			update(state => ({
				...state,
				hosts: state.hosts.map(item =>
					item.host.id === hostId
						? { ...item, host: { ...item.host, ...updates } }
						: item
				)
			}));
		},

		// Update host status
		updateStatus(hostId: string, status: HostStatus, lastSeen: string): void {
			update(state => ({
				...state,
				hosts: state.hosts.map(item =>
					item.host.id === hostId
						? {
								...item,
								host: {
									...item.host,
									status,
									last_seen: lastSeen
								}
						  }
						: item
				)
			}));
		},

		// Add a new host
		addHost(host: Host): void {
			update(state => ({
				...state,
				hosts: [...state.hosts, { host }]
			}));
		},

		// Remove a host
		removeHost(hostId: string): void {
			update(state => ({
				...state,
				hosts: state.hosts.filter(item => item.host.id !== hostId)
			}));
		},

		// Clear all hosts
		clear(): void {
			set({ hosts: [], loading: false, error: null });
		}
	};
}

export const hostsStore = createHostsStore();

// Derived stores for convenience
export const hosts = derived(hostsStore, $store => $store.hosts);
export const onlineHosts = derived(hostsStore, $store =>
	$store.hosts.filter(item => item.host.status === 'online')
);
export const offlineHosts = derived(hostsStore, $store =>
	$store.hosts.filter(item => item.host.status === 'offline')
);
export const hostsLoading = derived(hostsStore, $store => $store.loading);
