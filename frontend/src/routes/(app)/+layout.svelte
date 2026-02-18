<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { logout } from '$lib/api';
	import { countAlerts, logger } from '$lib/utils';
	import {
		userStore,
		serversStore,
		servers,
		metricsStore,
		aggregatedStore,
		alertsStore,
		uiStore,
		sidebarCollapsed,
		sidebarTransitioning
	} from '$lib/stores';
	import DesktopSidebar from '$lib/components/DesktopSidebar.svelte';
	import MobileSidebar from '$lib/components/MobileSidebar.svelte';
	import Header from '$lib/components/Header.svelte';
	import RightSidebar from '$lib/components/RightSidebar.svelte';

	const { children } = $props();

	let serversList = $derived($servers);
	let rightSidebarOpen = $derived($uiStore.rightSidebarOpen);
	let alertCount = $derived(countAlerts(serversList));
	let isDashboard = $derived($page.url.pathname === '/');

	const titleMap: Record<string, string> = {
		'/servers': 'Servers',
		'/servers/new': 'Add Server',
		'/settings': 'Settings',
	};

	let title = $derived.by(() => {
		const path = $page.url.pathname;
		if (path === '/') return '';
		if (titleMap[path]) return titleMap[path];
		if (path.match(/^\/servers\/[^/]+\/packages$/)) return 'Packages';
		if (path.match(/^\/servers\/[^/]+$/)) return 'Server Details';
		return '';
	});

	async function handleLogout() {
		try {
			await logout();
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
</script>

<div class="min-h-screen bg-background">
	<Header {title} showAlerts={isDashboard} {alertCount} />

	<DesktopSidebar onLogout={handleLogout} />
	<MobileSidebar onLogout={handleLogout} />

	{#if isDashboard}
		<RightSidebar servers={serversList} isOpen={rightSidebarOpen} onClose={() => uiStore.setRightSidebar(false)} />
	{/if}

	<main
		class="min-h-screen pt-[104px] p-4 md:p-8 md:pt-[112px] {$sidebarCollapsed
			? 'lg:ml-20'
			: 'lg:ml-64'} {$sidebarTransitioning ? 'transition-[margin] duration-300 ease-in-out' : ''}"
	>
		{@render children()}
	</main>
</div>
