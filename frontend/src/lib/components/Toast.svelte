<script>
	import { toasts } from '$lib/stores/toasts';
	import { fade, fly } from 'svelte/transition';

	// Icon for each toast type
	function getIcon(type) {
		switch (type) {
			case 'success':
				return '✓';
			case 'warning':
				return '⚠';
			case 'error':
				return '✕';
			case 'info':
			default:
				return 'ℹ';
		}
	}

	// Color classes for each type
	function getColorClasses(type) {
		switch (type) {
			case 'success':
				return 'bg-success/10 border-success/20 text-success';
			case 'warning':
				return 'bg-warning/10 border-warning/20 text-warning';
			case 'error':
				return 'bg-destructive/10 border-destructive/20 text-destructive';
			case 'info':
			default:
				return 'bg-primary/10 border-primary/20 text-primary';
		}
	}
</script>

<div class="fixed top-4 right-4 z-50 flex flex-col gap-2 max-w-md">
	{#each $toasts as toast (toast.id)}
		<div
			transition:fly={{ x: 300, duration: 300 }}
			class="flex items-start gap-3 p-4 rounded-lg border shadow-lg {getColorClasses(toast.type)}"
		>
			<span class="text-xl font-bold flex-shrink-0">{getIcon(toast.type)}</span>
			<p class="flex-1 text-sm">{toast.message}</p>
			<button
				on:click={() => toasts.remove(toast.id)}
				class="text-current opacity-50 hover:opacity-100 transition-opacity flex-shrink-0"
				aria-label="Close notification"
			>
				✕
			</button>
		</div>
	{/each}
</div>
