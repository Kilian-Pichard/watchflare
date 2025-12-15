const API_BASE_URL = 'http://localhost:8080';

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
		const data = JSON.parse(e.data);
		if (onMessage) onMessage({ type: 'metrics_update', data });
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
