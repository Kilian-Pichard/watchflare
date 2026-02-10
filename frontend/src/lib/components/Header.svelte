<script>
	import { mobileMenuOpen, sidebarCollapsed } from '$lib/stores/sidebar';

	const { title } = $props();

	function toggleMenu() {
		mobileMenuOpen.update(val => !val);
	}
</script>

<header class="fixed left-0 right-0 top-0 z-30 h-14 border-b bg-background {$sidebarCollapsed ? 'lg:left-16' : 'lg:left-64'}">
	<div class="flex h-full items-center justify-between px-4">
		<!-- Burger button (visible only on mobile) -->
		<button
			onclick={toggleMenu}
			class="flex h-9 w-9 items-center justify-center rounded-lg text-foreground transition-colors hover:bg-muted lg:hidden"
			aria-label="Toggle menu"
		>
			{#if $mobileMenuOpen}
				<!-- X icon -->
				<svg class="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
				</svg>
			{:else}
				<!-- Burger icon -->
				<svg class="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h16"/>
				</svg>
			{/if}
		</button>

		<!-- Spacer on desktop (where burger would be) -->
		<div class="hidden lg:block w-9"></div>

		<!-- Title / Logo -->
		{#if title}
			<h1 class="text-base font-semibold text-foreground">{title}</h1>
		{:else}
			<span class="text-lg font-bold text-primary">Watchflare</span>
		{/if}

		<!-- Spacer pour centrer le titre -->
		<div class="w-9"></div>
	</div>
</header>
