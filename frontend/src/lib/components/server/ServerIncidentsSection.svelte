<script lang="ts">
	import { onMount } from 'svelte';
	import * as api from '$lib/api';
	import type { ServerIncident, IncidentStatusFilter } from '$lib/types';
	import { ALERT_METRIC_LABELS } from '$lib/types';
	import { formatDateTime, formatRelativeTime } from '$lib/utils';
	import { userStore } from '$lib/stores/user';

	const { serverId }: { serverId: string } = $props();

	const timeFormat = $derived(($userStore.user?.time_format ?? '24h') as '12h' | '24h');

	const LIMIT = 20;

	let incidents: ServerIncident[] = $state([]);
	let totalCount = $state(0);
	let offset = $state(0);
	let loading = $state(true);
	let loadingMore = $state(false);
	let statusFilter: IncidentStatusFilter = $state('all');

	onMount(() => {
		loadIncidents(true);
	});

	async function loadIncidents(reset = false) {
		if (reset) {
			offset = 0;
			incidents = [];
			loading = true;
		} else {
			loadingMore = true;
		}
		try {
			const data = await api.getServerIncidents(serverId, {
				status: statusFilter,
				limit: LIMIT,
				offset: reset ? 0 : offset,
			});
			if (reset) {
				incidents = data.incidents;
			} else {
				incidents = [...incidents, ...data.incidents];
			}
			totalCount = data.total_count;
			offset = incidents.length;
		} catch {
			// silent — non-critical section
		} finally {
			loading = false;
			loadingMore = false;
		}
	}

	function handleFilterChange(filter: IncidentStatusFilter) {
		statusFilter = filter;
		loadIncidents(true);
	}

	function incidentDuration(incident: ServerIncident): string {
		const start = new Date(incident.started_at).getTime();
		const end = incident.resolved_at ? new Date(incident.resolved_at).getTime() : Date.now();
		const secs = Math.floor((end - start) / 1000);
		if (secs < 60) return `${secs}s`;
		if (secs < 3600) return `${Math.floor(secs / 60)}m ${secs % 60}s`;
		const h = Math.floor(secs / 3600);
		const m = Math.floor((secs % 3600) / 60);
		return m > 0 ? `${h}h ${m}m` : `${h}h`;
	}

	function formatIncidentValue(incident: ServerIncident): string {
		const { metric_type, current_value, threshold_value } = incident;
		if (metric_type === 'server_down') return '—';
		const isPercent = ['cpu_usage', 'memory_usage', 'disk_usage'].includes(metric_type);
		const isLoad = metric_type.startsWith('load_avg');
		const isTemp = metric_type === 'temperature';
		if (isPercent) return `${current_value.toFixed(1)}% / ${threshold_value.toFixed(0)}%`;
		if (isLoad) return `${current_value.toFixed(2)} / ${threshold_value.toFixed(2)}`;
		if (isTemp) return `${current_value.toFixed(1)}°C / ${threshold_value.toFixed(0)}°C`;
		return `${current_value.toFixed(2)} / ${threshold_value.toFixed(2)}`;
	}
</script>

<div class="mb-8">
	<div class="mb-4 flex items-center justify-between">
		<h2 class="text-base font-semibold text-foreground">Incident History</h2>
		<!-- Filter tabs -->
		<div class="flex rounded-lg border bg-muted/40 p-0.5 text-xs font-medium">
			{#each (['all', 'active', 'resolved'] as IncidentStatusFilter[]) as filter}
				<button
					onclick={() => handleFilterChange(filter)}
					class="rounded-md px-3 py-1 capitalize transition-colors {statusFilter === filter
						? 'bg-background text-foreground shadow-sm'
						: 'text-muted-foreground hover:text-foreground'}"
				>
					{filter}
				</button>
			{/each}
		</div>
	</div>

	{#if loading}
		<div class="rounded-xl border bg-card p-6 text-center">
			<p class="text-xs text-muted-foreground">Loading incidents...</p>
		</div>
	{:else if incidents.length === 0}
		<div class="rounded-xl border border-dashed bg-muted/20 p-6 text-center">
			<p class="text-xs text-muted-foreground">No incidents found</p>
		</div>
	{:else}
		<div class="rounded-xl border bg-card overflow-hidden">
			<table class="w-full text-xs">
				<thead>
					<tr class="border-b bg-muted/30">
						<th class="px-4 py-2.5 text-left font-medium text-muted-foreground uppercase tracking-wide text-[10px]">Status</th>
						<th class="px-4 py-2.5 text-left font-medium text-muted-foreground uppercase tracking-wide text-[10px]">Metric</th>
						<th class="px-4 py-2.5 text-left font-medium text-muted-foreground uppercase tracking-wide text-[10px]">Value / Threshold</th>
						<th class="px-4 py-2.5 text-left font-medium text-muted-foreground uppercase tracking-wide text-[10px]">Started</th>
						<th class="px-4 py-2.5 text-left font-medium text-muted-foreground uppercase tracking-wide text-[10px]">Duration</th>
						<th class="px-4 py-2.5 text-left font-medium text-muted-foreground uppercase tracking-wide text-[10px]">Resolved</th>
					</tr>
				</thead>
				<tbody class="divide-y">
					{#each incidents as incident (incident.id)}
						<tr class="hover:bg-muted/20 transition-colors">
							<td class="px-4 py-2.5">
								{#if incident.resolved_at}
									<span class="inline-flex items-center rounded-full px-2 py-0.5 text-[10px] font-medium bg-success/10 text-success border border-success/20">
										Resolved
									</span>
								{:else}
									<span class="inline-flex items-center rounded-full px-2 py-0.5 text-[10px] font-medium bg-destructive/10 text-destructive border border-destructive/20">
										Active
									</span>
								{/if}
							</td>
							<td class="px-4 py-2.5 text-foreground font-medium">
								{ALERT_METRIC_LABELS[incident.metric_type] ?? incident.metric_type}
							</td>
							<td class="px-4 py-2.5 text-muted-foreground font-mono">
								{formatIncidentValue(incident)}
							</td>
							<td class="px-4 py-2.5 text-muted-foreground" title={formatDateTime(incident.started_at, timeFormat)}>
								{formatRelativeTime(incident.started_at)}
							</td>
							<td class="px-4 py-2.5 text-muted-foreground">
								{incidentDuration(incident)}
							</td>
							<td class="px-4 py-2.5 text-muted-foreground">
								{incident.resolved_at ? formatDateTime(incident.resolved_at, timeFormat) : '—'}
							</td>
						</tr>
					{/each}
				</tbody>
			</table>

			{#if incidents.length < totalCount}
				<div class="border-t px-4 py-3 text-center">
					<button
						onclick={() => loadIncidents(false)}
						disabled={loadingMore}
						class="text-xs font-medium text-primary hover:text-primary/80 transition-colors disabled:opacity-40"
					>
						{loadingMore ? 'Loading...' : `Load more (${totalCount - incidents.length} remaining)`}
					</button>
				</div>
			{/if}
		</div>
	{/if}
</div>
