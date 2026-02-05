const API_BASE_URL = 'http://localhost:8080';

/**
 * Decode minified SSE metrics format to full format
 * Minified format: {s, t, c, mu, mt, ma, du, dt, l1, l5, l15, u}
 * Full format: {server_id, timestamp, cpu_usage_percent, memory_used_bytes, ...}
 */
function decodeMinifiedMetrics(minified) {
	return {
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

export function connectSSE(onMessage, onError) {
	const eventSource = new EventSource(`${API_BASE_URL}/servers/events`, {
		withCredentials: true
	});

	// Handle different event types
	eventSource.addEventListener('connected', (e) => {
		const data = JSON.parse(e.data);
		console.log('SSE connected:', data.client_id);
	});

	eventSource.addEventListener('server_update', (e) => {
		const data = JSON.parse(e.data);
		if (onMessage) onMessage({ type: 'server_update', data });
	});

	eventSource.addEventListener('metrics_update', (e) => {
		const minified = JSON.parse(e.data);
		// Decode minified format to full format
		const data = decodeMinifiedMetrics(minified);
		if (onMessage) onMessage({ type: 'metrics_update', data });
	});

	eventSource.addEventListener('aggregated_metrics_update', (e) => {
		const data = JSON.parse(e.data);
		if (onMessage) onMessage({ type: 'aggregated_metrics_update', data });
	});

	eventSource.onerror = (error) => {
		console.error('SSE error:', error);
		if (onError) onError(error);
	};

	// Return disconnect function
	return () => {
		eventSource.close();
	};
}
