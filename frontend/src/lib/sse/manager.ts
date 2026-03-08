import type { SSEEvent, Metric } from '../types';
import { API_BASE_URL } from '../api';
import { logger } from '../utils';

/**
 * Connection states for SSE
 */
export type ConnectionState = 'disconnected' | 'connecting' | 'connected' | 'reconnecting' | 'error';

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
	dr: number;      // disk_read_bytes_per_sec
	dw: number;      // disk_write_bytes_per_sec
	nr: number;      // network_rx_bytes_per_sec
	nt: number;      // network_tx_bytes_per_sec
	tmp: number;     // cpu_temperature_celsius
	u: number;       // uptime_seconds
}

/**
 * Decode minified SSE metrics format to full format
 */
function decodeMinifiedMetrics(minified: MinifiedMetrics): Metric {
	return {
		id: 0,
		server_id: minified.s,
		timestamp: new Date(minified.t * 1000).toISOString(),
		cpu_usage_percent: minified.c,
		memory_used_bytes: minified.mu,
		memory_total_bytes: minified.mt,
		memory_available_bytes: minified.ma,
		disk_used_bytes: minified.du,
		disk_total_bytes: minified.dt,
		load_avg_1min: minified.l1,
		load_avg_5min: minified.l5,
		load_avg_15min: minified.l15,
		disk_read_bytes_per_sec: minified.dr ?? 0,
		disk_write_bytes_per_sec: minified.dw ?? 0,
		network_rx_bytes_per_sec: minified.nr ?? 0,
		network_tx_bytes_per_sec: minified.nt ?? 0,
		cpu_temperature_celsius: minified.tmp ?? 0,
		uptime_seconds: minified.u
	};
}

/**
 * Configuration for SSE Manager
 */
interface SSEManagerConfig {
	/** Initial retry delay in ms (default: 1000) */
	initialRetryDelay?: number;
	/** Maximum retry delay in ms (default: 30000) */
	maxRetryDelay?: number;
	/** Maximum number of retry attempts (default: Infinity) */
	maxRetries?: number;
	/** Buffer delay in ms to batch events (default: 100) */
	bufferDelay?: number;
}

/**
 * Advanced SSE Manager with reconnection and buffering
 */
export class SSEManager {
	private eventSource: EventSource | null = null;
	private state: ConnectionState = 'disconnected';
	private retryCount = 0;
	private retryDelay: number;
	private retryTimer: ReturnType<typeof setTimeout> | null = null;
	private eventBuffer: SSEEvent[] = [];
	private bufferTimer: ReturnType<typeof setTimeout> | null = null;
	private shouldReconnect = true;

	private config: Required<SSEManagerConfig>;
	private onMessageCallback?: (event: SSEEvent) => void;
	private onBatchCallback?: (events: SSEEvent[]) => void;
	private onStateChangeCallback?: (state: ConnectionState) => void;
	private onErrorCallback?: (error: Event | Error) => void;

	constructor(config: SSEManagerConfig = {}) {
		this.config = {
			initialRetryDelay: config.initialRetryDelay ?? 1000,
			maxRetryDelay: config.maxRetryDelay ?? 30000,
			maxRetries: config.maxRetries ?? Infinity,
			bufferDelay: config.bufferDelay ?? 100
		};
		this.retryDelay = this.config.initialRetryDelay;
	}

	/**
	 * Connect to SSE endpoint
	 */
	connect(): void {
		if (this.eventSource) {
			return; // Already connected
		}

		this.shouldReconnect = true;
		this.setState('connecting');

		try {
			this.eventSource = new EventSource(`${API_BASE_URL}/servers/events`, {
				withCredentials: true
			});

			this.setupEventListeners();
		} catch (err) {
			logger.error('Failed to create EventSource:', err);
			this.handleError(err instanceof Error ? err : new Error('Failed to connect'));
		}
	}

	/**
	 * Disconnect from SSE endpoint
	 */
	disconnect(): void {
		this.shouldReconnect = false;
		this.cleanup();
		this.setState('disconnected');
	}

	/**
	 * Register message callback
	 */
	onMessage(callback: (event: SSEEvent) => void): void {
		this.onMessageCallback = callback;
	}

	/**
	 * Register batch callback (receives buffered events)
	 */
	onBatch(callback: (events: SSEEvent[]) => void): void {
		this.onBatchCallback = callback;
	}

	/**
	 * Register state change callback
	 */
	onStateChange(callback: (state: ConnectionState) => void): void {
		this.onStateChangeCallback = callback;
	}

	/**
	 * Register error callback
	 */
	onError(callback: (error: Event | Error) => void): void {
		this.onErrorCallback = callback;
	}

	/**
	 * Get current connection state
	 */
	getState(): ConnectionState {
		return this.state;
	}

	/**
	 * Setup event listeners on EventSource
	 */
	private setupEventListeners(): void {
		if (!this.eventSource) return;

		this.eventSource.addEventListener('open', () => {
			logger.log('SSE connection opened');
			this.setState('connected');
			this.retryCount = 0;
			this.retryDelay = this.config.initialRetryDelay;
		});

		this.eventSource.addEventListener('connected', (e: MessageEvent) => {
			const data = JSON.parse(e.data) as { client_id: string };
			logger.log('SSE connected:', data.client_id);
		});

		this.eventSource.addEventListener('server_update', (e: MessageEvent) => {
			const data = JSON.parse(e.data);
			this.bufferEvent({ type: 'server_update', data });
		});

		this.eventSource.addEventListener('metrics_update', (e: MessageEvent) => {
			const minified = JSON.parse(e.data) as MinifiedMetrics;
			const data = decodeMinifiedMetrics(minified);
			this.bufferEvent({ type: 'metrics_update', data });
		});

		this.eventSource.addEventListener('aggregated_metrics_update', (e: MessageEvent) => {
			const data = JSON.parse(e.data);
			this.bufferEvent({ type: 'aggregated_metrics_update', data });
		});

		this.eventSource.onerror = (error: Event) => {
			logger.error('SSE error:', error);
			this.handleError(error);
		};
	}

	/**
	 * Buffer event and flush after delay
	 */
	private bufferEvent(event: SSEEvent): void {
		this.eventBuffer.push(event);

		// Emit immediately if no buffer callback
		if (!this.onBatchCallback && this.onMessageCallback) {
			this.onMessageCallback(event);
			return;
		}

		// Clear existing timer
		if (this.bufferTimer) {
			clearTimeout(this.bufferTimer);
		}

		// Set new timer to flush buffer
		this.bufferTimer = setTimeout(() => {
			this.flushBuffer();
		}, this.config.bufferDelay);
	}

	/**
	 * Flush buffered events
	 */
	private flushBuffer(): void {
		if (this.eventBuffer.length === 0) return;

		const events = [...this.eventBuffer];
		this.eventBuffer = [];

		// Call batch callback if registered
		if (this.onBatchCallback) {
			this.onBatchCallback(events);
		}

		// Also call individual message callbacks
		if (this.onMessageCallback) {
			events.forEach(event => this.onMessageCallback!(event));
		}
	}

	/**
	 * Handle error and trigger reconnection
	 */
	private handleError(error: Event | Error): void {
		if (this.onErrorCallback) {
			this.onErrorCallback(error);
		}

		// Check if we should reconnect
		if (!this.shouldReconnect) {
			this.setState('disconnected');
			return;
		}

		if (this.retryCount >= this.config.maxRetries) {
			logger.error('Max retry attempts reached');
			this.setState('error');
			this.shouldReconnect = false;
			return;
		}

		this.setState('reconnecting');
		this.scheduleReconnect();
	}

	/**
	 * Schedule reconnection with exponential backoff
	 */
	private scheduleReconnect(): void {
		if (this.retryTimer) {
			clearTimeout(this.retryTimer);
		}

		logger.log(`Reconnecting in ${this.retryDelay}ms... (attempt ${this.retryCount + 1})`);

		this.retryTimer = setTimeout(() => {
			this.retryCount++;
			this.cleanup();
			this.connect();

			// Exponential backoff: double the delay, cap at maxRetryDelay
			this.retryDelay = Math.min(this.retryDelay * 2, this.config.maxRetryDelay);
		}, this.retryDelay);
	}

	/**
	 * Set connection state and notify listeners
	 */
	private setState(state: ConnectionState): void {
		if (this.state === state) return;

		this.state = state;
		logger.log(`SSE state: ${state}`);

		if (this.onStateChangeCallback) {
			this.onStateChangeCallback(state);
		}
	}

	/**
	 * Cleanup resources
	 */
	private cleanup(): void {
		// Flush any remaining buffered events
		this.flushBuffer();

		if (this.bufferTimer) {
			clearTimeout(this.bufferTimer);
			this.bufferTimer = null;
		}

		if (this.retryTimer) {
			clearTimeout(this.retryTimer);
			this.retryTimer = null;
		}

		if (this.eventSource) {
			this.eventSource.close();
			this.eventSource = null;
		}
	}
}
