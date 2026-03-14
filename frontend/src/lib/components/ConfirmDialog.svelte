<script lang="ts">
	import type { Snippet } from 'svelte';
	import Modal from './Modal.svelte';

	const {
		open,
		title,
		onConfirm,
		onClose,
		confirmLabel = 'Confirm',
		confirmVariant = 'primary',
		children,
	}: {
		open: boolean;
		title: string;
		onConfirm: () => void;
		onClose: () => void;
		confirmLabel?: string;
		confirmVariant?: 'destructive' | 'primary';
		children: Snippet;
	} = $props();

	const confirmClass = $derived(
		confirmVariant === 'destructive'
			? 'bg-destructive text-destructive-foreground hover:bg-destructive/90'
			: 'bg-primary text-primary-foreground hover:bg-primary/90'
	);
</script>

<Modal {open} {onClose}>
	<h3 class="text-lg font-semibold text-foreground mb-3">{title}</h3>
	{@render children()}
	<div class="flex gap-3 justify-end mt-6">
		<button
			onclick={onClose}
			class="rounded-lg border bg-background px-4 py-2 text-sm font-medium text-foreground transition-colors hover:bg-muted"
		>
			Cancel
		</button>
		<button
			onclick={onConfirm}
			class="rounded-lg px-4 py-2 text-sm font-medium transition-colors {confirmClass}"
		>
			{confirmLabel}
		</button>
	</div>
</Modal>
