<script lang="ts">
	const {
		currentPage,
		totalPages,
		totalItems,
		pageSize,
		itemLabel = 'items',
		onPageChange,
	}: {
		currentPage: number;
		totalPages: number;
		totalItems: number;
		pageSize: number;
		itemLabel?: string;
		onPageChange: (page: number) => void;
	} = $props();

	const start = $derived((currentPage - 1) * pageSize + 1);
	const end = $derived(Math.min(currentPage * pageSize, totalItems));
</script>

{#if totalPages > 1}
	<div class="flex items-center justify-between border-t px-4 py-3">
		<p class="text-sm text-muted-foreground">
			{start}-{end} of {totalItems} {itemLabel}
		</p>
		<div class="flex items-center gap-2">
			<button
				onclick={() => onPageChange(currentPage - 1)}
				disabled={currentPage <= 1}
				class="rounded-lg border bg-background px-3 py-1.5 text-sm font-medium text-foreground transition-colors hover:bg-muted disabled:opacity-50 disabled:cursor-not-allowed"
			>
				Previous
			</button>
			<span class="text-sm text-muted-foreground">
				{currentPage} / {totalPages}
			</span>
			<button
				onclick={() => onPageChange(currentPage + 1)}
				disabled={currentPage >= totalPages}
				class="rounded-lg border bg-background px-3 py-1.5 text-sm font-medium text-foreground transition-colors hover:bg-muted disabled:opacity-50 disabled:cursor-not-allowed"
			>
				Next
			</button>
		</div>
	</div>
{/if}
