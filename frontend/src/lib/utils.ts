import { clsx, type ClassValue } from 'clsx';
import { twMerge } from 'tailwind-merge';
import type { TimeRange } from './types';

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

// Get status color classes
export function getStatusColor(percentage: number): string {
	if (percentage >= 90) return 'text-red-500';
	if (percentage >= 70) return 'text-orange-500';
	return 'text-green-500';
}

export function getStatusBgColor(percentage: number): string {
	if (percentage >= 90) return 'bg-red-500';
	if (percentage >= 70) return 'bg-orange-500';
	return 'bg-green-500';
}
