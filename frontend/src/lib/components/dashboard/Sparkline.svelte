<script lang="ts">
	import type { TimeRange } from '$lib/types';

	const {
		values,
		timestamps,
		timeRange,
		yMin,
		yMax,
		class: className = "",
	}: {
		values: number[];
		timestamps?: number[];
		timeRange?: TimeRange;
		yMin?: number;
		yMax?: number;
		class?: string;
	} = $props();

	const W = 100;
	const H = 36;
	const TENSION = 0.4;

	// 1.5× bucket interval per time range, in seconds — mirrors UPlotChart GAP_THRESHOLDS
	const GAP_THRESHOLDS_S: Record<string, number> = {
		"1h":  45,
		"12h": 900,
		"24h": 1350,
		"7d":  10800,
		"30d": 43200,
	};

	// Duration of each time range in seconds
	const TIME_RANGE_S: Record<string, number> = {
		"1h":  3600,
		"12h": 43200,
		"24h": 86400,
		"7d":  604800,
		"30d": 2592000,
	};

	// Stable unique ID per instance for the SVG gradient reference
	const id = `sg-${Math.random().toString(36).slice(2, 7)}`;

	function buildPath(vals: number[], ts?: number[], tr?: TimeRange, fixedMin?: number, fixedMax?: number): { line: string; fill: string; dots: { x: number; y: number }[] } {
		if (vals.length < 2) return { line: '', fill: '', dots: [] };

		const min = fixedMin ?? Math.min(...vals);
		const max = fixedMax ?? Math.max(...vals);
		const range = max - min || 1;

		// X positioning: time-based when timestamps + timeRange are available,
		// anchored to [lastTs - duration, lastTs] to match the large charts.
		// Fallback: index-based.
		const duration = tr ? TIME_RANGE_S[tr] : null;
		const useTime = ts && ts.length === vals.length && duration != null;
		const xStart = useTime ? ts![ts!.length - 1] - duration! : 0;

		const pts = vals.map((v, i) => ({
			x: useTime
				? Math.max(0, Math.min(W, ((ts![i] - xStart) / duration!) * W))
				: (i / (vals.length - 1)) * W,
			y: H - ((v - min) / range) * (H * 0.85) - H * 0.075,
		}));

		// Gap detection: use time-range-aware threshold when available,
		// otherwise fall back to 2× average interval heuristic.
		const gapAfter = new Set<number>();
		if (ts && ts.length === vals.length && ts.length > 1) {
			const threshold = tr && GAP_THRESHOLDS_S[tr]
				? GAP_THRESHOLDS_S[tr]
				: (ts[ts.length - 1] - ts[0]) / (ts.length - 1) * 2;
			for (let i = 0; i < ts.length - 1; i++) {
				if (ts[i + 1] - ts[i] > threshold) gapAfter.add(i);
			}
		}

		// Split into continuous segments; collect isolated single points as dots
		const segments: { x: number; y: number }[][] = [];
		const dots: { x: number; y: number }[] = [];
		let seg: { x: number; y: number }[] = [];
		for (let i = 0; i < pts.length; i++) {
			seg.push(pts[i]);
			if (gapAfter.has(i)) {
				if (seg.length >= 2) segments.push(seg);
				else if (seg.length === 1) dots.push(seg[0]);
				seg = [];
			}
		}
		if (seg.length >= 2) segments.push(seg);
		else if (seg.length === 1) dots.push(seg[0]);

		// Catmull-Rom → cubic bezier smooth path for one segment
		function smoothSegment(s: { x: number; y: number }[]): string {
			let d = `M ${s[0].x.toFixed(2)},${s[0].y.toFixed(2)}`;
			for (let i = 0; i < s.length - 1; i++) {
				const p0 = s[Math.max(0, i - 1)];
				const p1 = s[i];
				const p2 = s[i + 1];
				const p3 = s[Math.min(s.length - 1, i + 2)];
				const cp1x = p1.x + (p2.x - p0.x) * TENSION;
				const cp1y = p1.y + (p2.y - p0.y) * TENSION;
				const cp2x = p2.x - (p3.x - p1.x) * TENSION;
				const cp2y = p2.y - (p3.y - p1.y) * TENSION;
				d += ` C ${cp1x.toFixed(2)},${cp1y.toFixed(2)} ${cp2x.toFixed(2)},${cp2y.toFixed(2)} ${p2.x.toFixed(2)},${p2.y.toFixed(2)}`;
			}
			return d;
		}

		const line = segments.map(smoothSegment).join(' ');

		const fill = segments
			.map((s) => {
				const last = s[s.length - 1];
				const first = s[0];
				return `${smoothSegment(s)} L ${last.x.toFixed(2)},${H} L ${first.x.toFixed(2)},${H} Z`;
			})
			.join(' ');

		return { line, fill, dots };
	}

	const path = $derived(buildPath(values, timestamps, timeRange, yMin, yMax));
</script>

<div class="relative w-full h-8 {className}" aria-hidden="true">
	<svg
		viewBox="0 0 {W} {H}"
		preserveAspectRatio="none"
		class="w-full h-full"
		overflow="hidden"
	>
		<defs>
			<linearGradient id={id} x1="0" y1="0" x2="0" y2="1">
				<stop offset="0%" stop-color="currentColor" stop-opacity="0.15" />
				<stop offset="100%" stop-color="currentColor" stop-opacity="0" />
			</linearGradient>
		</defs>
		{#if path.fill}
			<path d={path.fill} fill="url(#{id})" stroke="none" />
		{/if}
		{#if path.line}
			<path
				d={path.line}
				fill="none"
				stroke="currentColor"
				stroke-width="1"
				stroke-linecap="round"
				stroke-linejoin="round"
				vector-effect="non-scaling-stroke"
			/>
		{/if}
	</svg>
	{#each path.dots as dot}
		<div
			class="absolute rounded-full bg-current pointer-events-none"
			style="width: 2px; height: 2px; left: {(dot.x / W) * 100}%; top: {(dot.y / H) * 100}%; transform: translate(-50%, -50%); box-shadow: 0 0 0 1px hsl(var(--card))"
		></div>
	{/each}
</div>
