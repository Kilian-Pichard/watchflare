<script>
	import { fade, fly } from 'svelte/transition';
	import { generateAlerts } from '$lib/utils';
	import { XCircle, AlertTriangle } from 'lucide-svelte';

	const { servers, isOpen, onClose } = $props();

	const alerts = $derived(generateAlerts(servers));

	function getAlertColor(type) {
		if (type === 'critical') {
			return 'bg-destructive/10 text-destructive border-destructive/20';
		}
		return 'bg-warning/10 text-warning border-warning/20';
	}
</script>

<svelte:window onkeydown={e => e.key === 'Escape' && isOpen && onClose()} />

{#if isOpen}
	<!-- Backdrop -->
	<!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
	<div
		transition:fade={{ duration: 200 }}
		class="fixed inset-0 z-40 bg-black/50"
		role="presentation"
		onclick={onClose}
	></div>

	<!-- Panel -->
	<div
		transition:fly={{ x: 320, duration: 300 }}
		class="fixed right-0 top-0 z-50 h-screen w-80 max-w-[85vw] py-4 pr-4 bg-transparent"
		role="dialog"
		aria-modal="true"
		tabindex="-1"
	>
		<div class="flex h-full flex-col overflow-y-auto bg-sidebar rounded-2xl border">
		<!-- Header -->
		<div class="flex items-center justify-between border-b px-6 py-4">
			<h2 class="text-sm font-semibold text-foreground">Active Alerts</h2>
			<div class="flex items-center gap-2">
				{#if alerts.length > 0}
					<span class="flex h-5 w-5 items-center justify-center rounded-full bg-destructive text-xs font-medium text-primary-foreground">
						{alerts.length}
					</span>
				{/if}
				<button
					onclick={onClose}
					class="flex h-8 w-8 items-center justify-center rounded-lg text-muted-foreground hover:bg-muted hover:text-foreground transition-colors"
					aria-label="Close alerts"
				>
					<svg class="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
					</svg>
				</button>
			</div>
		</div>

		<!-- Alerts list -->
		<div class="p-6">
			{#if alerts.length === 0}
				<div class="rounded-lg border border-dashed bg-muted/20 p-6 text-center">
					<svg class="mx-auto h-8 w-8 text-muted-foreground/50 mb-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"/>
					</svg>
					<p class="text-xs text-muted-foreground">No alerts</p>
				</div>
			{:else}
				<div class="space-y-2">
					{#each alerts as alert}
						<div class="rounded-lg border p-3 {getAlertColor(alert.type)}">
							<div class="flex items-start gap-2">
								<div class="mt-0.5">
									{#if alert.type === 'critical'}
										<XCircle class="h-4 w-4" />
									{:else}
										<AlertTriangle class="h-4 w-4" />
									{/if}
								</div>
								<div class="flex-1 min-w-0">
									<p class="text-xs font-medium mb-0.5">{alert.server}</p>
									<p class="text-xs opacity-90">{alert.message}</p>
									<p class="text-xs opacity-60 mt-1">{alert.time}</p>
								</div>
							</div>
						</div>
					{/each}
				</div>
			{/if}
		</div>
		</div>
	</div>
{/if}
