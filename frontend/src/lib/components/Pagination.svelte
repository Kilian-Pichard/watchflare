<script lang="ts">
	import * as Select from '$lib/components/ui/select';
	import { Button } from '$lib/components/ui/button';
	import { ChevronLeft, ChevronRight } from 'lucide-svelte';

	const {
		currentPage,
		totalPages,
		totalItems,
		pageSize,
		itemLabel = 'items',
		onPageChange,
		onPageSizeChange,
		pageSizeOptions,
	}: {
		currentPage: number;
		totalPages: number;
		totalItems: number;
		pageSize: number;
		itemLabel?: string;
		onPageChange: (page: number) => void;
		onPageSizeChange?: (size: number) => void;
		pageSizeOptions?: number[];
	} = $props();

	const start = $derived((currentPage - 1) * pageSize + 1);
	const end = $derived(Math.min(currentPage * pageSize, totalItems));
	const visible = $derived(totalPages > 1 || !!onPageSizeChange);
</script>

{#if visible}
	<!-- Mobile: stack vertical -->
	<div class="sm:hidden flex flex-col gap-2 border-t px-4 py-3">
		<div class="flex items-center justify-between">
			<p class="text-sm text-muted-foreground">
				{#if totalPages > 1}{start}-{end} of {totalItems}{:else}{totalItems}{/if}
			</p>
			{#if onPageSizeChange && pageSizeOptions}
				<Select.Root
					type="single"
					value={String(pageSize)}
					onValueChange={(v) => onPageSizeChange(Number(v))}
				>
					<Select.Trigger class="py-1.5 text-sm" items={pageSizeOptions.map(String)}>
						<span>{pageSize} / page</span>
					</Select.Trigger>
					<Select.Content>
						{#each pageSizeOptions as size}
							<Select.Item value={String(size)} label={`${size} / page`}>
								{size} / page
							</Select.Item>
						{/each}
					</Select.Content>
				</Select.Root>
			{/if}
		</div>
		{#if totalPages > 1}
			<div class="flex items-center justify-center gap-2">
				<Button
					variant="outline"
					size="icon-sm"
					onclick={() => onPageChange(currentPage - 1)}
					disabled={currentPage <= 1}
				>
					<ChevronLeft class="h-4 w-4" />
				</Button>
				<span class="text-sm text-muted-foreground">{currentPage} / {totalPages}</span>
				<Button
					variant="outline"
					size="icon-sm"
					onclick={() => onPageChange(currentPage + 1)}
					disabled={currentPage >= totalPages}
				>
					<ChevronRight class="h-4 w-4" />
				</Button>
			</div>
		{/if}
	</div>

	<!-- Desktop: single row -->
	<div class="hidden sm:flex items-center justify-between border-t px-4 py-3">
		<div class="flex items-center gap-3">
			<p class="text-sm text-muted-foreground">
				{#if totalPages > 1}{start}-{end} of {totalItems}{:else}{totalItems}{/if}
			</p>
			{#if onPageSizeChange && pageSizeOptions}
				<Select.Root
					type="single"
					value={String(pageSize)}
					onValueChange={(v) => onPageSizeChange(Number(v))}
				>
					<Select.Trigger class="py-1.5 text-sm" items={pageSizeOptions.map(String)}>
						<span>{pageSize} / page</span>
					</Select.Trigger>
					<Select.Content>
						{#each pageSizeOptions as size}
							<Select.Item value={String(size)} label={`${size} / page`}>
								{size} / page
							</Select.Item>
						{/each}
					</Select.Content>
				</Select.Root>
			{/if}
		</div>
		{#if totalPages > 1}
			<div class="flex items-center gap-2">
				<Button
					variant="outline"
					size="sm"
					onclick={() => onPageChange(currentPage - 1)}
					disabled={currentPage <= 1}
				>
					Previous
				</Button>
				<span class="text-sm text-muted-foreground">{currentPage} / {totalPages}</span>
				<Button
					variant="outline"
					size="sm"
					onclick={() => onPageChange(currentPage + 1)}
					disabled={currentPage >= totalPages}
				>
					Next
				</Button>
			</div>
		{/if}
	</div>
{/if}
