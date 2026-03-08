<script lang="ts">
	import { goto } from '$app/navigation';
	import { getStatusClass } from '$lib/utils';
	import type { Server } from '$lib/types';

	const {
		servers,
		sortColumn,
		sortOrder,
		onSort,
		onDelete,
		onDismissReactivation,
	}: {
		servers: Server[];
		sortColumn: string;
		sortOrder: 'asc' | 'desc';
		onSort: (column: string) => void;
		onDelete: (server: Server, e: Event) => void;
		onDismissReactivation: (serverId: string) => void;
	} = $props();

	function hasIPMismatch(server: Server) {
		return (
			server.configured_ip &&
			server.ip_address_v4 &&
			server.configured_ip !== server.ip_address_v4 &&
			!server.ignore_ip_mismatch
		);
	}
</script>

{#snippet sortIcon(column)}
	{#if sortColumn === column}
		<svg class="h-3 w-3" viewBox="0 0 12 12" fill="currentColor">
			{#if sortOrder === 'asc'}
				<path d="M6 2l4 5H2z" />
			{:else}
				<path d="M6 10l4-5H2z" />
			{/if}
		</svg>
	{:else}
		<svg class="h-3 w-3 opacity-0 group-hover:opacity-50 transition-opacity" viewBox="0 0 12 12" fill="currentColor">
			<path d="M6 10l4-5H2z" />
		</svg>
	{/if}
{/snippet}

<table class="w-full min-w-[640px]">
	<thead>
		<tr class="border-b bg-muted/30">
			<th scope="col" class="w-2/5 px-4 py-2 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider" onclick={() => onSort('name')}>
				<span class="group inline-flex items-center gap-1 h-8 rounded-md px-2.5 cursor-pointer select-none transition-colors hover:bg-muted hover:text-foreground">
					Name
					{@render sortIcon('name')}
				</span>
			</th>
			<th scope="col" class="w-1/5 px-4 py-2 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider" onclick={() => onSort('status')}>
				<span class="group inline-flex items-center gap-1 h-8 rounded-md px-2.5 cursor-pointer select-none transition-colors hover:bg-muted hover:text-foreground">
					Status
					{@render sortIcon('status')}
				</span>
			</th>
			<th scope="col" class="w-1/4 px-4 py-2 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider" onclick={() => onSort('ip')}>
				<span class="group inline-flex items-center gap-1 h-8 rounded-md px-2.5 cursor-pointer select-none transition-colors hover:bg-muted hover:text-foreground">
					IP Address
					{@render sortIcon('ip')}
				</span>
			</th>
			<th scope="col" class="px-4 py-2 text-right text-xs font-medium text-muted-foreground uppercase tracking-wider">
				Actions
			</th>
		</tr>
	</thead>
	<tbody class="divide-y divide-border">
		{#each servers as server}
			<tr
				onclick={() => goto(`/servers/${server.id}`)}
				class="hover:bg-muted/20 transition-colors cursor-pointer"
			>
				<td class="px-4 py-3.5">
					<div class="flex flex-col">
						<span class="font-medium text-foreground">{server.name}</span>
						{#if server.hostname}
							<span class="text-xs text-muted-foreground">{server.hostname}</span>
						{/if}
					</div>
				</td>
				<td class="px-4 py-3.5">
					<div class="flex items-center gap-2">
						<span
							class="inline-flex items-center gap-1.5 rounded-full border px-2.5 py-0.5 text-xs font-medium {getStatusClass(server.status)}"
						>
							<span
								class="h-1.5 w-1.5 rounded-full {server.status === 'online' ? 'bg-success' : 'bg-muted-foreground'}"
							></span>
							{server.status}
						</span>
						{#if hasIPMismatch(server)}
							<span class="inline-flex items-center text-warning" title="IP mismatch detected">
								<svg class="h-4 w-4" fill="currentColor" viewBox="0 0 20 20">
									<path
										fill-rule="evenodd"
										d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z"
										clip-rule="evenodd"
									/>
								</svg>
							</span>
						{/if}
						{#if server.reactivated_at}
							<span
								class="inline-flex items-center gap-1 rounded-full border border-primary/20 bg-primary/10 px-2 py-0.5 text-xs font-medium text-primary"
								title="Agent was reactivated (same physical server via UUID)"
							>
								<svg class="h-3 w-3" fill="currentColor" viewBox="0 0 20 20">
									<path
										fill-rule="evenodd"
										d="M4 2a1 1 0 011 1v2.101a7.002 7.002 0 0111.601 2.566 1 1 0 11-1.885.666A5.002 5.002 0 005.999 7H9a1 1 0 010 2H4a1 1 0 01-1-1V3a1 1 0 011-1zm.008 9.057a1 1 0 011.276.61A5.002 5.002 0 0014.001 13H11a1 1 0 110-2h5a1 1 0 011 1v5a1 1 0 11-2 0v-2.101a7.002 7.002 0 01-11.601-2.566 1 1 0 01.61-1.276z"
										clip-rule="evenodd"
									/>
								</svg>
								Reactivated
								<button
									onclick={(e) => {
										e.stopPropagation();
										onDismissReactivation(server.id);
									}}
									class="ml-0.5 text-primary hover:text-primary/80"
									title="Dismiss"
								>
									<svg class="h-3 w-3" fill="currentColor" viewBox="0 0 20 20">
										<path
											fill-rule="evenodd"
											d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z"
											clip-rule="evenodd"
										/>
									</svg>
								</button>
							</span>
						{/if}
					</div>
				</td>
				<td class="px-4 py-3.5 text-sm text-foreground">
					{server.ip_address_v4 || server.configured_ip || '-'}
				</td>
					<td class="px-4 py-3.5 text-right">
					<div class="flex items-center justify-end gap-3">
						<button
							onclick={(e) => {
								e.stopPropagation();
								goto(`/servers/${server.id}`);
							}}
							class="text-sm font-medium text-primary hover:text-primary/80 transition-colors"
						>
							View
						</button>
						<button
							onclick={(e) => onDelete(server, e)}
							class="text-sm font-medium text-destructive hover:text-destructive/80 transition-colors"
						>
							Delete
						</button>
					</div>
				</td>
			</tr>
		{/each}
	</tbody>
</table>
