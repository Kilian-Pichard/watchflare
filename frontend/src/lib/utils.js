import { clsx } from 'clsx';
import { twMerge } from 'tailwind-merge';

export function cn(...inputs) {
	return twMerge(clsx(inputs));
}

// Time range utilities
export const TIME_RANGES = [
	{ value: '1h', label: '1 Hour', seconds: 3600 },
	{ value: '6h', label: '6 Hours', seconds: 21600 },
	{ value: '24h', label: '24 Hours', seconds: 86400 },
	{ value: '7d', label: '7 Days', seconds: 604800 },
	{ value: '30d', label: '30 Days', seconds: 2592000 }
];

export function getTimeRangeTimestamps(timeRange) {
	const range = TIME_RANGES.find((r) => r.value === timeRange);
	if (!range) return null;

	const end = new Date();
	const start = new Date(end.getTime() - range.seconds * 1000);

	return {
		start: start.toISOString(),
		end: end.toISOString()
	};
}

export function getIntervalForTimeRange(timeRange) {
	const intervals = {
		'1h': '', // No aggregation - raw data
		'6h': '5m',
		'24h': '15m',
		'7d': '1h',
		'30d': '6h'
	};
	return intervals[timeRange] || '';
}

// Format bytes to human-readable
export function formatBytes(bytes) {
	if (bytes === 0) return '0 B';
	const k = 1024;
	const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
	const i = Math.floor(Math.log(bytes) / Math.log(k));
	return Math.round((bytes / Math.pow(k, i)) * 100) / 100 + ' ' + sizes[i];
}

// Format percentage
export function formatPercent(value) {
	return Math.round(value * 10) / 10 + '%';
}

// Format uptime
export function formatUptime(seconds) {
	const days = Math.floor(seconds / 86400);
	const hours = Math.floor((seconds % 86400) / 3600);
	const minutes = Math.floor((seconds % 3600) / 60);

	if (days > 0) return `${days}d ${hours}h`;
	if (hours > 0) return `${hours}h ${minutes}m`;
	return `${minutes}m`;
}

// Get status color
export function getStatusColor(percentage) {
	if (percentage >= 90) return 'text-red-500';
	if (percentage >= 70) return 'text-orange-500';
	return 'text-green-500';
}

export function getStatusBgColor(percentage) {
	if (percentage >= 90) return 'bg-red-500';
	if (percentage >= 70) return 'bg-orange-500';
	return 'bg-green-500';
}
