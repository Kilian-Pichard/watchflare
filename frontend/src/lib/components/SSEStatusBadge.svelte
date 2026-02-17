<script lang="ts">
	import { sseConnectionState, sseIsReconnecting } from '$lib/stores/sse';

	const { collapsed = false }: { collapsed?: boolean } = $props();

	const connectionState = $derived($sseConnectionState);
	const isReconnecting = $derived($sseIsReconnecting);

	function getStateColor(state: string): string {
		switch (state) {
			case 'connected':
				return 'bg-success';
			case 'connecting':
				return 'bg-warning';
			case 'reconnecting':
				return 'bg-warning';
			case 'error':
				return 'bg-destructive';
			default:
				return 'bg-muted-foreground';
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
	class="flex items-center gap-2 rounded-lg px-3 py-2 text-xs overflow-hidden"
	title={collapsed ? getStateLabel(connectionState) : ''}
>
	<span class="relative flex h-2 w-2 flex-shrink-0">
		<span
			class="absolute inline-flex h-full w-full rounded-full opacity-75 {isReconnecting
				? 'animate-ping'
				: ''} {getStateColor(connectionState)}"
		></span>
		<span class="relative inline-flex h-2 w-2 rounded-full {getStateColor(connectionState)}">
		</span>
	</span>
	<span class="text-muted-foreground whitespace-nowrap">{getStateLabel(connectionState)}</span>
</div>
