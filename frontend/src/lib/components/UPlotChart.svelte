<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import uPlot from 'uplot';
	import 'uplot/dist/uPlot.min.css';
	import { formatTooltipDate } from '$lib/chart-utils';

	let {
		data,
		series,
		axes,
		scales,
	}: {
		data: uPlot.AlignedData;
		series: uPlot.Series[];
		axes?: uPlot.Axis[];
		scales?: uPlot.Scales;
	} = $props();

	let container: HTMLDivElement;
	let chart: uPlot | null = null;
	let mounted = false;
	let rawMouseTop = 0;

	// Resolve CSS variable to a Canvas-compatible hex color
	function resolveColor(color: string): string {
		// Step 1: resolve CSS variable (var(--x)) to computed value via DOM
		const el = document.createElement('div');
		el.style.color = color;
		document.body.appendChild(el);
		const computed = getComputedStyle(el).color;
		document.body.removeChild(el);

		// Step 2: convert to hex via Canvas (handles oklch, hsl, etc.)
		const ctx = document.createElement('canvas').getContext('2d')!;
		ctx.fillStyle = computed;
		return ctx.fillStyle;
	}

	function tooltipPlugin(): uPlot.Plugin {
		let tooltip: HTMLDivElement;

		function init(u: uPlot) {
			tooltip = document.createElement('div');
			tooltip.style.cssText = `
				display: none;
				position: absolute;
				z-index: 50;
				pointer-events: none;
				background: var(--color-popover);
				color: var(--color-popover-foreground);
				border: 1px solid var(--color-border);
				border-radius: 0.5rem;
				padding: 0.5rem 0.75rem;
				font-size: 0.75rem;
				line-height: 1.25rem;
				box-shadow: 0 4px 6px -1px rgb(0 0 0 / 0.1);
				white-space: nowrap;
			`;
			u.over.parentElement!.appendChild(tooltip);

			// Touch support: horizontal movement controls cursor, vertical scrolls page
			const over = u.over;
			let touchStartX = 0;
			let touchStartY = 0;
			let isHorizontal: boolean | null = null;

			over.addEventListener('touchstart', (e) => {
				const t = e.touches[0];
				touchStartX = t.clientX;
				touchStartY = t.clientY;
				isHorizontal = null;
			}, { passive: true });

			over.addEventListener('touchmove', (e) => {
				const t = e.touches[0];
				// Determine direction on first significant move
				if (isHorizontal === null) {
					const dx = Math.abs(t.clientX - touchStartX);
					const dy = Math.abs(t.clientY - touchStartY);
					if (dx < 5 && dy < 5) return;
					isHorizontal = dx > dy;
				}
				if (!isHorizontal) return; // vertical = let page scroll
				e.preventDefault();
				const rect = over.getBoundingClientRect();
				const left = t.clientX - rect.left;
				const top = t.clientY - rect.top;
				u.setCursor({ left, top });
			}, { passive: false });

			over.addEventListener('touchend', () => {
				isHorizontal = null;
				u.setCursor({ left: -10, top: -10 });
			}, { passive: true });
		}

		function setCursor(u: uPlot) {
			const idx = u.cursor.idx;
			if (idx == null || idx < 0) {
				tooltip.style.display = 'none';
				return;
			}

			const ts = u.data[0][idx];
			let html = `<div style="margin-bottom:4px;font-weight:500;">${formatTooltipDate(new Date(ts * 1000))}</div>`;

			let hasValue = false;
			for (let i = 1; i < u.series.length; i++) {
				const s = u.series[i];
				if (!s.show) continue;
				const val = u.data[i][idx];
				if (val == null) continue;
				hasValue = true;
				const color = (s as any)._stroke ?? s.stroke ?? '#888';
				const formatted = s.value
					? (s.value as (u: uPlot, v: number | null, si: number, i: number | null) => string)(u, val, i, idx)
					: String(val);
				html += `<div style="display:flex;align-items:center;gap:6px;">
					<span style="width:8px;height:8px;border-radius:2px;background:${color};display:inline-block;"></span>
					<span style="color:var(--color-muted-foreground);">${s.label}:</span>
					<span style="font-weight:500;">${formatted}</span>
				</div>`;
			}

			if (!hasValue) {
				tooltip.style.display = 'none';
				return;
			}

			tooltip.innerHTML = html;
			tooltip.style.display = 'block';

			const left = u.cursor.left ?? 0;
			const plotLeft = u.bbox.left / devicePixelRatio;
			const plotWidth = u.bbox.width / devicePixelRatio;

			let tipX = left + plotLeft + 12;
			let tipY = rawMouseTop - tooltip.offsetHeight / 2;

			if (tipX + tooltip.offsetWidth > plotLeft + plotWidth) {
				tipX = left + plotLeft - tooltip.offsetWidth - 12;
			}

			tooltip.style.left = tipX + 'px';
			tooltip.style.top = Math.max(0, tipY) + 'px';
		}

		return {
			hooks: {
				init: [init],
				setCursor: [setCursor],
			}
		};
	}

	function buildOpts(width: number, chartHeight: number): uPlot.Options {
		const resolvedSeries: uPlot.Series[] = series.map(s => {
			const resolved: uPlot.Series = { ...s };
			if (s.stroke) resolved.stroke = resolveColor(s.stroke as string);
			if (s.fill) {
				const hex = resolveColor(s.fill as string); // returns #rrggbb
				const r = parseInt(hex.slice(1, 3), 16);
				const g = parseInt(hex.slice(3, 5), 16);
				const b = parseInt(hex.slice(5, 7), 16);
				resolved.fill = `rgba(${r},${g},${b},0.2)`;
			}
			return resolved;
		});

		const gridStroke = resolveColor('var(--border)');
		const textColor = resolveColor('var(--muted-foreground)');

		const defaultAxes: uPlot.Axis[] = [
			{
				stroke: textColor,
				grid: { stroke: gridStroke, width: 1 },
				ticks: { stroke: gridStroke, width: 1 },
				font: '11px system-ui',
				values: (_u: uPlot, ticks: number[]) => ticks.map(t => {
					const d = new Date(t * 1000);
					return `${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`;
				}),
			},
			{
				stroke: textColor,
				grid: { stroke: gridStroke, width: 1 },
				ticks: { stroke: gridStroke, width: 1 },
				font: '11px system-ui',
				size: 70,
				splits: (u: uPlot) => {
					const min = (u.scales.y as any).min ?? 0;
					const max = (u.scales.y as any).max ?? 100;
					if (max <= min) return [min];
					const step = (max - min) / 4;
					return [
						min,
						min + step,
						min + step * 2,
						min + step * 3,
						max,
					];
				},
			}
		];

		const mergedAxes = axes
			? defaultAxes.map((def, i) => axes[i] ? { ...def, ...axes[i] } : def)
			: defaultAxes;

		return {
			width,
			height: chartHeight,
			cursor: {
				drag: { x: false, y: false },
				y: false,
				move(u: uPlot, left: number, top: number) {
					rawMouseTop = top;
					// Keep negative values (mouseleave hides cursor at -10)
					if (left < 0) return [left, top];
					// Snap vertical line to nearest data point
					const idx = u.posToIdx(left);
					if (idx >= 0 && idx < u.data[0].length) {
						left = Math.round(u.valToPos(u.data[0][idx], 'x'));
					}
					return [left, top];
				},
			},
			legend: { show: false },
			series: [{}, ...resolvedSeries],
			axes: mergedAxes,
			scales: scales || { y: { range: (_u: uPlot, _min: number, max: number) => {
					if (max <= 0) return [0, 1] as uPlot.Range.MinMax;
					// Find the display unit (1024-based: B, KB, MB, GB)
					const units = [1, 1024, 1024**2, 1024**3];
					let unit = 1;
					for (const u of units) {
						if (max >= u) unit = u;
					}
					const displayed = max / unit;
					// Round up to next nice number in that unit
					const mag = Math.pow(10, Math.floor(Math.log10(displayed)));
					const normalized = displayed / mag;
					const niceMaxes = [1, 2, 4, 5, 8, 10];
					const niceNorm = niceMaxes.find(n => n >= normalized) || 10;
					return [0, niceNorm * mag * unit] as uPlot.Range.MinMax;
				}}},
			plugins: [tooltipPlugin()],
		};
	}

	let lastWidth = 0;
	let lastHeight = 0;

	function createChart() {
		destroyChart();
		if (!container) return;
		if (!data || data[0].length === 0) return;
		const width = container.clientWidth;
		const height = container.clientHeight;
		if (width === 0 || height === 0) return;
		lastWidth = width;
		lastHeight = height;

		chart = new uPlot(
			buildOpts(width, height),
			data,
			container
		);
	}

	function destroyChart() {
		if (chart) {
			chart.destroy();
			chart = null;
		}
	}

	// Track series identity for recreation
	let seriesKey = $derived(series.map(s => s.label).join(','));

	// Track scales for recreation when range changes
	let scalesKey = $derived(JSON.stringify(scales || {}));
	let prevScalesKey = '';

	// When data, series, or scales change, update or recreate the chart
	$effect(() => {
		// Access reactive deps
		const _data = data;
		const _key = seriesKey;
		const _scales = scalesKey;

		if (!mounted || !container) return;

		const scalesChanged = _scales !== prevScalesKey;
		prevScalesKey = _scales;

		if (chart && _data && _data[0].length > 0 && _data.length === chart.series.length && !scalesChanged) {
			// Same series structure and scales — just update data
			chart.setData(_data);
		} else if (_data && _data[0].length > 0) {
			// Series changed or no chart yet — recreate
			createChart();
		}
	});

	onMount(() => {
		mounted = true;
		createChart();

		function onResize() {
			if (!chart || !container) return;
			const width = container.clientWidth;
			const height = container.clientHeight;
			if (width > 0 && height > 0 && (width !== lastWidth || height !== lastHeight)) {
				lastWidth = width;
				lastHeight = height;
				chart.setSize({ width, height });
			}
		}
		window.addEventListener('resize', onResize);

		// Recreate chart on theme change (colors are baked into canvas)
		const themeObserver = new MutationObserver(() => {
			createChart();
		});
		themeObserver.observe(document.documentElement, {
			attributes: true,
			attributeFilter: ['class']
		});

		return () => {
			window.removeEventListener('resize', onResize);
			themeObserver.disconnect();
		};
	});

	onDestroy(destroyChart);
</script>

<div class="h-48 sm:h-64 relative">
	<div class="absolute inset-0" bind:this={container}></div>
</div>
