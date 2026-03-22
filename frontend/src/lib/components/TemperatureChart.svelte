<script lang="ts">
	import UPlotChart from '$lib/components/UPlotChart.svelte';
	import type { Metric, TimeRange } from '$lib/types';
	import type uPlot from 'uplot';

	let { data = [], timeRange }: { data: Metric[]; timeRange?: TimeRange } = $props();

	const CHART_COLORS = [
		'var(--chart-1)',
		'var(--chart-2)',
		'var(--chart-3)',
		'var(--chart-4)',
		'var(--chart-5)',
	];

	// CPU-related sensor patterns (sorted first in the legend)
	const CPU_PATTERNS = ['tdie', 'coretemp', 'k10temp', 'cpu', 'package', 'tctl'];

	function isCPUKey(key: string): boolean {
		const lower = key.toLowerCase();
		return CPU_PATTERNS.some(p => lower.includes(p));
	}

	// Sorted unique sensor keys: CPU sensors first, then alphabetical
	const sensorKeys = $derived.by(() => {
		const keys = new Set<string>();
		for (const d of data) {
			if (d.sensor_readings) {
				for (const sr of d.sensor_readings) keys.add(sr.key);
			}
		}
		return [...keys].sort((a, b) => {
			const aCPU = isCPUKey(a);
			const bCPU = isCPUKey(b);
			if (aCPU !== bCPU) return aCPU ? -1 : 1;
			return a.localeCompare(b);
		});
	});

	const hasSensorReadings = $derived(sensorKeys.length > 0);

	const chartData = $derived.by((): uPlot.AlignedData => {
		if (data.length === 0) return [[], []] as uPlot.AlignedData;

		const timestamps: number[] = [];

		if (hasSensorReadings) {
			const seriesArrays: (number | null)[][] = sensorKeys.map(() => []);
			for (const d of data) {
				timestamps.push(new Date(d.timestamp).getTime() / 1000);
				if (d.sensor_readings && d.sensor_readings.length > 0) {
					const readingMap = new Map(d.sensor_readings.map(sr => [sr.key, sr.temperature_celsius]));
					for (let i = 0; i < sensorKeys.length; i++) {
						const val = readingMap.get(sensorKeys[i]);
						seriesArrays[i].push(val != null ? val : null);
					}
				} else {
					// Old metric: use cpu_temperature_celsius for the primary (first) CPU sensor
					for (let i = 0; i < sensorKeys.length; i++) {
						seriesArrays[i].push(i === 0 && d.cpu_temperature_celsius > 0 ? d.cpu_temperature_celsius : null);
					}
				}
			}
			return [timestamps, ...seriesArrays] as uPlot.AlignedData;
		}

		// Fallback: single cpu_temperature_celsius curve
		const temp: (number | null)[] = [];
		for (const d of data) {
			if (d.cpu_temperature_celsius > 0) {
				timestamps.push(new Date(d.timestamp).getTime() / 1000);
				temp.push(d.cpu_temperature_celsius);
			}
		}
		return [timestamps, temp] as uPlot.AlignedData;
	});

	const series = $derived(
		hasSensorReadings
			? sensorKeys.map((key, i): uPlot.Series => ({
				label: key,
				stroke: CHART_COLORS[i % CHART_COLORS.length],
				width: 2,
				value: (_u: uPlot, v: number | null) => v != null ? v.toFixed(1) + '°C' : '—',
			}))
			: [{
				label: 'CPU Temp',
				stroke: CHART_COLORS[0],
				fill: CHART_COLORS[0],
				width: 2,
				value: (_u: uPlot, v: number | null) => v != null ? v.toFixed(1) + '°C' : '—',
			}] as uPlot.Series[]
	);

	const axes: uPlot.Axis[] = [
		{},
		{ values: (_u: uPlot, ticks: number[]) => ticks.map(v => v + '°C') }
	];
</script>

<UPlotChart data={chartData} {series} {axes} {timeRange} />
