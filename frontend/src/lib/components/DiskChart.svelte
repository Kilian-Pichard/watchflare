<script>
	import { Chart, Svg, Axis, Area, Highlight } from 'layerchart';
	import { scaleTime, scaleLinear } from 'd3-scale';
	import { formatBytes } from '$lib/utils';

	let { data = [] } = $props();

	// Transform data for layerchart
	let chartData = $derived(
		data.map((d) => ({
			date: new Date(d.timestamp),
			used: d.disk_used_bytes,
			total: d.disk_total_bytes
		}))
	);

	// Calculate max for y-axis
	let maxDisk = $derived(
		chartData.length > 0 ? Math.max(...chartData.map((d) => d.total)) : 0
	);
</script>

{#if chartData.length > 0}
	<div class="h-64">
		<Chart
			data={chartData}
			x="date"
			xScale={scaleTime()}
			y="used"
			yScale={scaleLinear()}
			yDomain={[0, maxDisk]}
			padding={{ left: 70, bottom: 24, top: 8, right: 8 }}
		>
			<Svg>
				<Axis placement="left" grid rule format={(d) => formatBytes(d)} />
				<Axis
					placement="bottom"
					format={(d) => d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
					rule
				/>
				<Area class="fill-chart-2/20 stroke-chart-2 stroke-2" />
				<Highlight area class="fill-chart-2/30" />
			</Svg>
		</Chart>
	</div>
{:else}
	<div class="h-64 flex items-center justify-center text-muted-foreground">
		No data available
	</div>
{/if}
