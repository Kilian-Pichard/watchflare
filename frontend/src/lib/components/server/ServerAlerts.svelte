<script lang="ts">
	import { formatDateTime } from '$lib/utils';
	import type { Server } from '$lib/types';

	const {
		server,
		showIPMismatchWarning,
		onUpdateIP,
		onIgnoreIP,
		onDismissReactivation,
	}: {
		server: Server;
		showIPMismatchWarning: boolean;
		onUpdateIP: () => void;
		onIgnoreIP: () => void;
		onDismissReactivation: () => void;
	} = $props();
</script>

{#if showIPMismatchWarning}
	<div class="mb-6 rounded-lg border border-warning bg-warning/10 p-4">
		<div class="flex items-start gap-3">
			<svg class="h-5 w-5 text-warning mt-0.5" fill="currentColor" viewBox="0 0 20 20">
				<path
					fill-rule="evenodd"
					d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z"
					clip-rule="evenodd"
				/>
			</svg>
			<div class="flex-1">
				<p class="text-sm font-medium text-foreground">IP Address Mismatch</p>
				<p class="text-sm text-muted-foreground mt-1">
					Configured IP: {server.configured_ip} • Actual IP: {server.ip_address_v4}
				</p>
				<div class="mt-3 flex gap-2">
					<button
						onclick={onUpdateIP}
						class="rounded-lg bg-primary px-3 py-1.5 text-xs font-medium text-primary-foreground transition-colors hover:bg-primary/90"
					>
						Update to {server.ip_address_v4}
					</button>
					<button
						onclick={onIgnoreIP}
						class="rounded-lg border bg-background px-3 py-1.5 text-xs font-medium text-foreground transition-colors hover:bg-muted"
					>
						Ignore
					</button>
				</div>
			</div>
		</div>
	</div>
{/if}

{#if server.reactivated_at}
	<div class="mb-6 rounded-lg border border-primary bg-primary/10 p-4">
		<div class="flex items-start justify-between gap-3">
			<div class="flex items-start gap-3">
				<svg class="h-5 w-5 text-primary mt-0.5" fill="currentColor" viewBox="0 0 20 20">
					<path
						fill-rule="evenodd"
						d="M4 2a1 1 0 011 1v2.101a7.002 7.002 0 0111.601 2.566 1 1 0 11-1.885.666A5.002 5.002 0 005.999 7H9a1 1 0 010 2H4a1 1 0 01-1-1V3a1 1 0 011-1zm.008 9.057a1 1 0 011.276.61A5.002 5.002 0 0014.001 13H11a1 1 0 110-2h5a1 1 0 011 1v5a1 1 0 11-2 0v-2.101a7.002 7.002 0 01-11.601-2.566 1 1 0 01.61-1.276z"
						clip-rule="evenodd"
					/>
				</svg>
				<div>
					<p class="text-sm font-medium text-foreground">Agent Reactivated</p>
					<p class="text-sm text-muted-foreground mt-1">
						Same physical server detected via UUID at {formatDateTime(server.reactivated_at)}
					</p>
				</div>
			</div>
			<button onclick={onDismissReactivation} class="text-primary hover:text-primary/80" aria-label="Dismiss reactivation notice">
				<svg class="h-5 w-5" fill="currentColor" viewBox="0 0 20 20">
					<path
						fill-rule="evenodd"
						d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z"
						clip-rule="evenodd"
					/>
				</svg>
			</button>
		</div>
	</div>
{/if}
