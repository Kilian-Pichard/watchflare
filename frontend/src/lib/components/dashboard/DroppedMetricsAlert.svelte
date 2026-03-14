<script lang="ts">
	import type { DroppedMetric } from '$lib/types';

	const { alerts }: { alerts: DroppedMetric[] } = $props();

	function formatDuration(nanoseconds: number): string {
		const seconds = nanoseconds / 1_000_000_000;
		const hours = Math.floor(seconds / 3600);
		const minutes = Math.floor((seconds % 3600) / 60);

		if (hours > 0) {
			return `${hours}h${minutes > 0 ? ` ${minutes}min` : ''}`;
		} else if (minutes > 0) {
			return `${minutes}min`;
		} else {
			return `${Math.floor(seconds)}s`;
		}
	}
</script>

{#if alerts.length > 0}
	<div class="mb-6 rounded-lg border border-warning bg-warning/5 p-4">
		<h3 class="mb-3 flex items-center gap-2 text-sm font-semibold text-warning">
			<svg class="h-4 w-4" fill="currentColor" viewBox="0 0 20 20">
				<path
					fill-rule="evenodd"
					d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z"
					clip-rule="evenodd"
				/>
			</svg>
			Dropped Metrics
		</h3>
		{#each alerts as alert}
			<div class="mb-2 last:mb-0 rounded-md bg-background p-3">
				<p class="text-sm font-medium">
					<strong>{alert.hostname}</strong> dropped <strong>{alert.total_dropped} metrics</strong>
				</p>
				<p class="text-xs text-muted-foreground mt-1">
					Backend unavailable for {formatDuration(alert.downtime_duration)}
					({new Date(alert.first_dropped_at).toLocaleString()} → {new Date(
						alert.last_dropped_at
					).toLocaleString()})
				</p>
			</div>
		{/each}
	</div>
{/if}
