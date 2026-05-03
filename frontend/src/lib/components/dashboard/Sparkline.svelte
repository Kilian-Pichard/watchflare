<script lang="ts">
	const {
		values,
		timestamps,
		class: className = "",
	}: {
		values: number[];
		timestamps?: number[];
		class?: string;
	} = $props();

	const W = 100;
	const H = 36;
	const TENSION = 0.4;

	// Stable unique ID per instance for the SVG gradient reference
	const id = `sg-${Math.random().toString(36).slice(2, 7)}`;

	function buildPath(vals: number[], ts?: number[]): { line: string; fill: string } {
		if (vals.length < 2) return { line: '', fill: '' };

		const min = Math.min(...vals);
		const max = Math.max(...vals);
		const range = max - min || 1;

		const pts = vals.map((v, i) => ({
			x: (i / (vals.length - 1)) * W,
			y: H - ((v - min) / range) * (H * 0.85) - H * 0.075,
		}));

		// Detect gaps: flag index i if the interval to i+1 is > 2× average
		const gapAfter = new Set<number>();
		if (ts && ts.length === vals.length && ts.length > 1) {
			const avgInterval = (ts[ts.length - 1] - ts[0]) / (ts.length - 1);
			for (let i = 0; i < ts.length - 1; i++) {
				if (ts[i + 1] - ts[i] > avgInterval * 2) gapAfter.add(i);
			}
		}

		// Split into continuous segments
		const segments: { x: number; y: number }[][] = [];
		let seg: { x: number; y: number }[] = [];
		for (let i = 0; i < pts.length; i++) {
			seg.push(pts[i]);
			if (gapAfter.has(i)) {
				if (seg.length >= 2) segments.push(seg);
				seg = [];
			}
		}
		if (seg.length >= 2) segments.push(seg);
		if (segments.length === 0 && pts.length >= 2) segments.push(pts);

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

		return { line, fill };
	}

	const path = $derived(buildPath(values, timestamps));
</script>

<svg
	viewBox="0 0 {W} {H}"
	preserveAspectRatio="none"
	class="w-full h-8 {className}"
	aria-hidden="true"
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
