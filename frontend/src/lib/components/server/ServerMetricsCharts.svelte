<script lang="ts">
	import { formatBytes, formatPercent } from '$lib/utils';
	import CPUChart from '$lib/components/CPUChart.svelte';
	import MemoryChart from '$lib/components/MemoryChart.svelte';
	import DiskChart from '$lib/components/DiskChart.svelte';
	import TimeRangeSelector from '$lib/components/TimeRangeSelector.svelte';
	import type { Metric, TimeRange } from '$lib/types';

	let {
		metrics,
		timeRange = $bindable(),
		onTimeRangeChange,
	}: {
		metrics: Metric[];
		timeRange: TimeRange;
		onTimeRangeChange: (range: TimeRange) => void;
	} = $props();

	const latestMetric = $derived(metrics.length > 0 ? metrics[metrics.length - 1] : null);
</script>

<div class="mb-6">
	<div class="mb-4 flex items-center justify-between">
		<h2 class="text-lg font-semibold text-foreground">Metrics</h2>
		<TimeRangeSelector bind:value={timeRange} onValueChange={onTimeRangeChange} />
	</div>

	<div class="grid gap-4 xl:grid-cols-2">
		<div class="rounded-lg border bg-card p-4">
			<div class="mb-3 flex items-center justify-between">
				<h3 class="text-sm font-medium">CPU Usage</h3>
				{#if latestMetric}
					<span class="text-xs text-muted-foreground">{latestMetric.cpu_usage_percent.toFixed(1)}%</span>
				{/if}
			</div>
			<CPUChart data={metrics} {timeRange} />
		</div>
		<div class="rounded-lg border bg-card p-4">
			<div class="mb-3 flex items-center justify-between">
				<h3 class="text-sm font-medium">Memory Usage</h3>
				{#if latestMetric}
					<span class="text-xs text-muted-foreground">
						<span class="sm:hidden">{formatPercent(latestMetric.memory_total_bytes > 0 ? (latestMetric.memory_used_bytes / latestMetric.memory_total_bytes) * 100 : 0)}</span>
						<span class="hidden sm:inline">{formatBytes(latestMetric.memory_used_bytes)} / {formatBytes(latestMetric.memory_total_bytes)} ({formatPercent(latestMetric.memory_total_bytes > 0 ? (latestMetric.memory_used_bytes / latestMetric.memory_total_bytes) * 100 : 0)})</span>
					</span>
				{/if}
			</div>
			<MemoryChart data={metrics} {timeRange} />
		</div>
		<div class="rounded-lg border bg-card p-4">
			<div class="mb-3 flex items-center justify-between">
				<h3 class="text-sm font-medium">Disk Usage</h3>
				{#if latestMetric}
					<span class="text-xs text-muted-foreground">
						<span class="sm:hidden">{formatPercent(latestMetric.disk_total_bytes > 0 ? (latestMetric.disk_used_bytes / latestMetric.disk_total_bytes) * 100 : 0)}</span>
						<span class="hidden sm:inline">{formatBytes(latestMetric.disk_used_bytes)} / {formatBytes(latestMetric.disk_total_bytes)} ({formatPercent(latestMetric.disk_total_bytes > 0 ? (latestMetric.disk_used_bytes / latestMetric.disk_total_bytes) * 100 : 0)})</span>
					</span>
				{/if}
			</div>
			<DiskChart data={metrics} {timeRange} />
		</div>
	</div>
</div>
