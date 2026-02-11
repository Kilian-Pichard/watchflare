import type { SSEEvent, Metric } from './types';

const API_BASE_URL = 'http://localhost:8080';

/**
 * Minified metrics format from backend SSE
 */
interface MinifiedMetrics {
	s: string;       // server_id
	t: number;       // timestamp (Unix epoch)
	c: number;       // cpu_usage_percent
	mu: number;      // memory_used_bytes
	mt: number;      // memory_total_bytes
	ma: number;      // memory_available_bytes
	du: number;      // disk_used_bytes
	dt: number;      // disk_total_bytes
	l1: number;      // load_avg_1min
	l5: number;      // load_avg_5min
	l15: number;     // load_avg_15min
	u: number;       // uptime_seconds
}

/**
 * Decode minified SSE metrics format to full format
 * Minified format: {s, t, c, mu, mt, ma, du, dt, l1, l5, l15, u}
 * Full format: {server_id, timestamp, cpu_usage_percent, memory_used_bytes, ...}
 */
function decodeMinifiedMetrics(minified: MinifiedMetrics): Metric {
	return {
		id: 0, // Not provided in SSE, set to 0
		server_id: minified.s,
		timestamp: new Date(minified.t * 1000).toISOString(), // Unix timestamp to ISO string
		cpu_usage_percent: minified.c,
		memory_used_bytes: minified.mu,
		memory_total_bytes: minified.mt,
		memory_available_bytes: minified.ma,
		disk_used_bytes: minified.du,
		disk_total_bytes: minified.dt,
		load_avg_1min: minified.l1,
		load_avg_5min: minified.l5,
		load_avg_15min: minified.l15,
		uptime_seconds: minified.u
	};
}

/**
 * Connect to SSE endpoint for real-time updates
 * @param onMessage Callback for SSE messages
 * @param onError Callback for SSE errors
 * @returns Disconnect function to close the connection
 */
export function connectSSE(
	onMessage?: (event: SSEEvent) => void,
	onError?: (error: Event) => void
): () => void {
	const eventSource = new EventSource(`${API_BASE_URL}/servers/events`, {
		withCredentials: true
	});

	// Handle different event types
	eventSource.addEventListener('connected', (e: MessageEvent) => {
		const data = JSON.parse(e.data) as { client_id: string };
		console.log('SSE connected:', data.client_id);
	});

	eventSource.addEventListener('server_update', (e: MessageEvent) => {
		const data = JSON.parse(e.data);
		if (onMessage) onMessage({ type: 'server_update', data });
	});

	eventSource.addEventListener('metrics_update', (e: MessageEvent) => {
		const minified = JSON.parse(e.data) as MinifiedMetrics;
		// Decode minified format to full format
		const data = decodeMinifiedMetrics(minified);
		if (onMessage) onMessage({ type: 'metrics_update', data });
	});

	eventSource.addEventListener('aggregated_metrics_update', (e: MessageEvent) => {
		const data = JSON.parse(e.data);
		if (onMessage) onMessage({ type: 'aggregated_metrics_update', data });
	});

	eventSource.onerror = (error: Event) => {
		console.error('SSE error:', error);
		if (onError) onError(error);
	};

	// Return disconnect function
	return () => {
		eventSource.close();
	};
}
