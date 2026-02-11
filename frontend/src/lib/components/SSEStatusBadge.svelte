<script lang="ts">
	import { sseConnectionState, sseIsReconnecting } from '$lib/stores/sse';

	const { collapsed = false }: { collapsed?: boolean } = $props();

	const connectionState = $derived($sseConnectionState);
	const isReconnecting = $derived($sseIsReconnecting);

	function getStateColor(state: string): string {
		switch (state) {
			case 'connected':
				return 'bg-green-500';
			case 'connecting':
				return 'bg-yellow-500';
			case 'reconnecting':
				return 'bg-orange-500';
			case 'error':
				return 'bg-red-500';
			default:
				return 'bg-gray-500';
		}
	}

	function getStateLabel(state: string): string {
		switch (state) {
			case 'connected':
				return 'SSE Connected';
			case 'connecting':
				return 'SSE Connecting...';
			case 'reconnecting':
				return 'SSE Reconnecting...';
			case 'error':
				return 'SSE Error';
			default:
				return 'SSE Disconnected';
		}
	}
</script>

<div
	class="flex items-center rounded-lg px-3 py-2 text-xs {collapsed ? 'justify-center' : 'gap-2'}"
	title={collapsed ? getStateLabel(connectionState) : ''}
>
	<span class="relative flex h-2 w-2">
		<span
			class="absolute inline-flex h-full w-full rounded-full opacity-75 {isReconnecting
				? 'animate-ping'
				: ''} {getStateColor(connectionState)}"
		></span>
		<span class="relative inline-flex h-2 w-2 rounded-full {getStateColor(connectionState)}">
		</span>
	</span>
	{#if !collapsed}
		<span class="text-muted-foreground">{getStateLabel(connectionState)}</span>
	{/if}
</div>
