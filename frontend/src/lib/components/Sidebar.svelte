<script>
	import { page } from '$app/stores';
	import { sidebarCollapsed } from '$lib/stores/sidebar';

	const { onLogout } = $props();

	const navItems = [
		{ href: '/', label: 'Dashboard', icon: 'M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6' },
		{ href: '/servers', label: 'Servers', icon: 'M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2m-2-4h.01M17 16h.01' },
		{ href: '/settings', label: 'Settings', icon: 'M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z' }
	];

	function isActive(href) {
		if (href === '/') {
			return $page.url.pathname === '/';
		}
		return $page.url.pathname.startsWith(href);
	}

	function toggleCollapse() {
		sidebarCollapsed.update(val => !val);
	}
</script>

<aside class="fixed left-0 top-0 z-40 h-screen border-r bg-sidebar transition-all duration-300 {$sidebarCollapsed ? 'w-16' : 'w-64'}">
	<div class="flex h-full flex-col">
		<!-- Logo + Toggle -->
		<div class="flex h-16 items-center border-b {$sidebarCollapsed ? 'justify-center px-2' : 'justify-between px-6'}">
			<h1 class="text-xl font-semibold text-foreground {$sidebarCollapsed ? 'hidden' : 'block'}">
				Watchflare
			</h1>
			{#if $sidebarCollapsed}
				<span class="text-xl font-bold text-primary">W</span>
			{/if}
			<button
				onclick={toggleCollapse}
				class="flex h-8 w-8 items-center justify-center rounded-lg text-muted-foreground transition-colors hover:bg-sidebar-accent hover:text-foreground {$sidebarCollapsed ? 'hidden' : 'block'}"
				aria-label={$sidebarCollapsed ? 'Expand sidebar' : 'Collapse sidebar'}
			>
				{#if $sidebarCollapsed}
					<!-- Chevron Right (expand) -->
					<svg class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7"/>
					</svg>
				{:else}
					<!-- Chevron Left (collapse) -->
					<svg class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7"/>
					</svg>
				{/if}
			</button>
		</div>

		<!-- Navigation -->
		<nav class="flex-1 space-y-1 p-4">
			{#each navItems as item}
				<a
					href={item.href}
					class="flex items-center rounded-lg px-3 py-2.5 text-sm font-medium transition-colors {isActive(item.href)
						? 'bg-primary text-primary-foreground'
						: 'text-sidebar-foreground hover:bg-sidebar-accent'} {$sidebarCollapsed ? 'justify-center' : 'gap-3'}"
					title={$sidebarCollapsed ? item.label : ''}
				>
					<svg
						class="h-5 w-5 flex-shrink-0"
						fill="none"
						stroke="currentColor"
						viewBox="0 0 24 24"
					>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="2"
							d={item.icon}
						/>
					</svg>
					{#if !$sidebarCollapsed}
						<span>{item.label}</span>
					{/if}
				</a>
			{/each}
		</nav>

		<!-- Logout Button -->
		<div class="border-t p-4">
			<button
				onclick={onLogout}
				class="flex w-full items-center rounded-lg px-3 py-2.5 text-sm font-medium text-destructive transition-colors hover:bg-destructive/10 {$sidebarCollapsed ? 'justify-center' : 'gap-3'}"
				title={$sidebarCollapsed ? 'Logout' : ''}
			>
				<svg
					class="h-5 w-5 flex-shrink-0"
					fill="none"
					stroke="currentColor"
					viewBox="0 0 24 24"
				>
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						stroke-width="2"
						d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1"
					/>
				</svg>
				{#if !$sidebarCollapsed}
					<span>Logout</span>
				{/if}
			</button>
		</div>

		<!-- Collapse button when collapsed (at bottom) -->
		{#if $sidebarCollapsed}
			<div class="border-t p-2">
				<button
					onclick={toggleCollapse}
					class="flex w-full items-center justify-center rounded-lg p-2 text-muted-foreground transition-colors hover:bg-sidebar-accent hover:text-foreground"
					aria-label="Expand sidebar"
				>
					<!-- Chevron Right (expand) -->
					<svg class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7"/>
					</svg>
				</button>
			</div>
		{/if}
	</div>
</aside>
