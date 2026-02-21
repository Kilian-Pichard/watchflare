import { clsx, type ClassValue } from 'clsx';
import { twMerge } from 'tailwind-merge';
import type { SSEEvent, TimeRange } from './types';
import { toasts } from './stores/toasts';

// Tailwind class name utility
export function cn(...inputs: ClassValue[]) {
	return twMerge(clsx(inputs));
}

// Type utilities
export type WithoutChild<T> = T extends { child?: unknown } ? Omit<T, 'child'> : T;
export type WithoutChildren<T> = T extends { children?: unknown } ? Omit<T, 'children'> : T;
export type WithoutChildrenOrChild<T> = WithoutChildren<WithoutChild<T>>;
export type WithElementRef<T, U extends HTMLElement = HTMLElement> = T & { ref?: U | null };

// Time range utilities
export interface TimeRangeOption {
	value: TimeRange;
	label: string;
	seconds: number;
}

export const TIME_RANGES: TimeRangeOption[] = [
	{ value: '1h', label: '1 Hour', seconds: 3600 },
	{ value: '12h', label: '12 Hours', seconds: 43200 },
	{ value: '24h', label: '24 Hours', seconds: 86400 },
	{ value: '7d', label: '7 Days', seconds: 604800 },
	{ value: '30d', label: '30 Days', seconds: 2592000 }
];

export interface TimeRangeTimestamps {
	start: string;
	end: string;
}

export function getTimeRangeTimestamps(timeRange: TimeRange): TimeRangeTimestamps | null {
	const range = TIME_RANGES.find((r) => r.value === timeRange);
	if (!range) return null;

	const end = new Date();
	const start = new Date(end.getTime() - range.seconds * 1000);

	return {
		start: start.toISOString(),
		end: end.toISOString()
	};
}

export function getIntervalForTimeRange(timeRange: TimeRange): string {
	const intervals: Record<TimeRange, string> = {
		'1h': '',      // Raw data (every 30s) - 120 points
		'12h': '10m',  // Continuous aggregate 10min - 72 points
		'24h': '15m',  // Continuous aggregate 15min - 96 points
		'7d': '2h',    // Continuous aggregate 2h - 84 points
		'30d': '8h'    // Continuous aggregate 8h - 90 points
	};
	return intervals[timeRange] || '';
}

// Format bytes to human-readable
export function formatBytes(bytes: number): string {
	if (bytes === 0) return '0 B';
	const k = 1024;
	const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
	const i = Math.floor(Math.log(bytes) / Math.log(k));
	return Math.round((bytes / Math.pow(k, i)) * 100) / 100 + ' ' + sizes[i];
}

// Format percentage
export function formatPercent(value: number): string {
	return Math.round(value * 10) / 10 + '%';
}

// Format uptime
export function formatUptime(seconds: number): string {
	const days = Math.floor(seconds / 86400);
	const hours = Math.floor((seconds % 86400) / 3600);
	const minutes = Math.floor((seconds % 3600) / 60);

	if (days > 0) return `${days}d ${hours}h`;
	if (hours > 0) return `${hours}h ${minutes}m`;
	return `${minutes}m`;
}

// Server status badge class
export function getStatusClass(status: string): string {
	switch (status) {
		case 'online':
			return 'bg-success/10 text-success border-success/20';
		case 'pending':
			return 'bg-warning/10 text-warning border-warning/20';
		case 'ip_mismatch':
			return 'bg-warning/10 text-warning border-warning/20';
		case 'offline':
		case 'expired':
		default:
			return 'bg-muted text-muted-foreground border-border';
	}
}

// Metric threshold class (CPU, memory, disk percentages)
export function getMetricClass(percent: number): string {
	if (percent >= 90) return 'text-danger font-semibold';
	if (percent >= 70) return 'text-warning font-medium';
	return 'text-foreground';
}

// Format timestamp as relative time ("5s ago", "3m ago", "2h ago", "1d ago")
export function formatRelativeTime(dateString: string | null | undefined): string {
	if (!dateString) return 'Never';
	const date = new Date(dateString);
	const now = new Date();
	const diff = Math.floor((now.getTime() - date.getTime()) / 1000);

	if (diff < 60) return `${diff}s ago`;
	if (diff < 3600) return `${Math.floor(diff / 60)}m ago`;
	if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`;
	return `${Math.floor(diff / 86400)}d ago`;
}

// Format date as locale string
export function formatDateTime(dateString: string | null | undefined): string {
	if (!dateString) return '-';
	return new Date(dateString).toLocaleString('fr-FR');
}

// Count active alerts from server list
export interface ServerWithLatestMetric {
	server: { status: string; [key: string]: unknown };
	latestMetric?: {
		cpu_usage_percent: number;
		memory_used_bytes: number;
		memory_total_bytes: number;
		[key: string]: unknown;
	} | null;
}

export function countAlerts(servers: ServerWithLatestMetric[]): number {
	let count = 0;
	for (const { server, latestMetric } of servers) {
		if (server.status === 'offline') count++;
		if (server.status === 'ip_mismatch') count++;
		if (latestMetric && latestMetric.cpu_usage_percent > 90) count++;
		if (latestMetric && latestMetric.memory_total_bytes > 0) {
			const memPct = (latestMetric.memory_used_bytes / latestMetric.memory_total_bytes) * 100;
			if (memPct > 90) count++;
		}
	}
	return count;
}

export interface Alert {
	type: 'critical' | 'warning';
	server: string;
	message: string;
	time: string;
}

export function generateAlerts(servers: ServerWithLatestMetric[]): Alert[] {
	const alerts: Alert[] = [];

	for (const { server, latestMetric } of servers) {
		if (server.status === 'offline') {
			alerts.push({ type: 'critical', server: server.name as string, message: 'Server is offline', time: 'Just now' });
		}
		if (server.status === 'ip_mismatch') {
			alerts.push({ type: 'warning', server: server.name as string, message: 'IP address mismatch detected', time: 'Just now' });
		}
		if (latestMetric && latestMetric.cpu_usage_percent > 90) {
			alerts.push({ type: 'warning', server: server.name as string, message: `High CPU usage: ${latestMetric.cpu_usage_percent.toFixed(1)}%`, time: 'Just now' });
		}
		if (latestMetric && latestMetric.memory_total_bytes > 0) {
			const memPercent = (latestMetric.memory_used_bytes / latestMetric.memory_total_bytes) * 100;
			if (memPercent > 90) {
				alerts.push({ type: 'warning', server: server.name as string, message: `High memory usage: ${memPercent.toFixed(1)}%`, time: 'Just now' });
			}
		}
	}

	return alerts.slice(0, 10);
}

// SSE reactivation toast (shared across pages)
export function handleSSEReactivation(event: SSEEvent): void {
	if (event.type === 'server_update' && event.data.reactivated && event.data.hostname) {
		toasts.add(
			`Agent "${event.data.hostname}" was reactivated (same physical server detected via UUID)`,
			'info',
			8000
		);
	}
}

// Dev-only logger (silenced in production builds)
export const logger = {
	error: (...args: unknown[]) => { if (import.meta.env.DEV) console.error(...args); },
	warn: (...args: unknown[]) => { if (import.meta.env.DEV) console.warn(...args); },
	log: (...args: unknown[]) => { if (import.meta.env.DEV) console.log(...args); },
};
