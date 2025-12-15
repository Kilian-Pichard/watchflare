<script>
	import { Chart, Svg, Axis, Spline, Highlight } from 'layerchart';
	import { scaleTime, scaleLinear } from 'd3-scale';

	let { data = [] } = $props();

	// Transform data for layerchart
	let chartData = $derived(
		data.map((d) => ({
			date: new Date(d.timestamp),
			value: d.cpu_usage_percent
		}))
	);
</script>

{#if chartData.length > 0}
	<div class="h-64">
		<Chart
			data={chartData}
			x="date"
			xScale={scaleTime()}
			y="value"
			yScale={scaleLinear()}
			yDomain={[0, 100]}
			yNice
			padding={{ left: 40, bottom: 24, top: 8, right: 8 }}
		>
			<Svg>
				<Axis placement="left" grid rule />
				<Axis
					placement="bottom"
					format={(d) => d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
					rule
				/>
				<Spline class="stroke-2 stroke-primary" />
				<Highlight area class="fill-primary/10" />
			</Svg>
		</Chart>
	</div>
{:else}
	<div class="h-64 flex items-center justify-center text-muted-foreground">
		No data available
	</div>
{/if}
