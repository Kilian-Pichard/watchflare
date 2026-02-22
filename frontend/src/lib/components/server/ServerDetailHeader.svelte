<script lang="ts">
	import { getStatusClass, formatRelativeTime } from '$lib/utils';
	import type { Server, PackageStats } from '$lib/types';

	const {
		server,
		packageStats,
		onDelete,
		onRegenerateToken,
		onChangeIP,
	}: {
		server: Server;
		packageStats: PackageStats | null;
		onDelete: () => void;
		onRegenerateToken: () => void;
		onChangeIP: () => void;
	} = $props();
</script>

<div class="mb-6 rounded-lg border bg-card p-4 md:p-6">
	<!-- Top: Name, status, actions -->
	<div class="flex flex-col gap-4 md:flex-row md:items-start md:justify-between">
		<div class="flex items-center gap-3 flex-wrap">
			<h1 class="text-xl sm:text-2xl font-semibold text-foreground">
				{server.name}
			</h1>
			<span
				class="inline-flex items-center gap-1.5 rounded-full border px-2.5 py-0.5 text-xs font-medium {getStatusClass(server.status)}"
			>
				<span
					class="h-1.5 w-1.5 rounded-full {server.status === 'online' ? 'bg-success' : 'bg-muted-foreground'}"
				></span>
				{server.status}
			</span>
		</div>
		<div class="flex flex-wrap gap-2">
			<button
				onclick={onChangeIP}
				class="rounded-lg border bg-background px-3 py-1.5 text-sm font-medium text-foreground transition-colors hover:bg-muted whitespace-nowrap"
			>
				Change IP
			</button>
			<button
				onclick={onRegenerateToken}
				class="rounded-lg border bg-background px-3 py-1.5 text-sm font-medium text-foreground transition-colors hover:bg-muted whitespace-nowrap"
			>
				Regenerate Token
			</button>
			<button
				onclick={onDelete}
				class="rounded-lg border border-destructive bg-destructive/10 px-3 py-1.5 text-sm font-medium text-destructive transition-colors hover:bg-destructive/20 whitespace-nowrap"
			>
				Delete
			</button>
		</div>
	</div>

	<!-- Server details grid -->
	<div class="mt-6 grid grid-cols-2 md:grid-cols-4 gap-x-6 gap-y-3">
		{#if server.hostname}
			<div>
				<p class="text-xs text-muted-foreground">Hostname</p>
				<p class="text-sm font-medium text-foreground">{server.hostname}</p>
			</div>
		{/if}
		{#if server.platform}
			<div>
				<p class="text-xs text-muted-foreground">OS</p>
				<p class="text-sm font-medium text-foreground">
					{server.platform_version ? `${server.platform} ${server.platform_version}` : server.platform}
				</p>
			</div>
		{/if}
		{#if server.architecture}
			<div>
				<p class="text-xs text-muted-foreground">Architecture</p>
				<p class="text-sm font-medium text-foreground">{server.architecture}</p>
			</div>
		{/if}
		{#if server.ip_address_v4 || server.configured_ip}
			<div>
				<p class="text-xs text-muted-foreground">IP Address</p>
				<p class="text-sm font-medium text-foreground">{server.ip_address_v4 || server.configured_ip}</p>
			</div>
		{/if}
		{#if server.last_seen}
			<div>
				<p class="text-xs text-muted-foreground">Last Seen</p>
				<p class="text-sm font-medium text-foreground">{formatRelativeTime(server.last_seen)}</p>
			</div>
		{/if}
	</div>

	<!-- Packages -->
	{#if packageStats}
		<div class="mt-4 pt-4 border-t flex items-center justify-between">
			<div class="flex items-center gap-4">
				<div>
					<p class="text-xs text-muted-foreground">Packages</p>
					<p class="text-sm font-medium text-foreground">
						{packageStats.total_packages || 0} installed
					</p>
				</div>
				{#if packageStats.recent_changes}
					<div>
						<p class="text-xs text-muted-foreground">Recent Changes</p>
						<p class="text-sm font-medium text-foreground">
							{packageStats.recent_changes} in the last 30d
						</p>
					</div>
				{/if}
			</div>
			<a
				href="/servers/{server.id}/packages"
				class="rounded-lg border bg-background px-3 py-1.5 text-sm font-medium text-foreground transition-colors hover:bg-muted whitespace-nowrap"
			>
				View Packages →
			</a>
		</div>
	{/if}
</div>
