import { TIME_RANGES } from '$lib/utils';
import type { TimeRange } from '$lib/types';

// Calcul du xDomain ancré au dernier point de données
export function computeXDomain(
	chartData: { date: Date }[],
	timeRange?: TimeRange
): [Date, Date] | undefined {
	if (!timeRange || chartData.length === 0) return undefined;
	const range = TIME_RANGES.find((r) => r.value === timeRange);
	if (!range) return undefined;
	const lastDate = chartData[chartData.length - 1].date;
	return [new Date(lastDate.getTime() - range.seconds * 1000), lastDate];
}

// Filtre les données pour ne garder que celles dans la fenêtre xDomain
export function filterByDomain<T extends { date: Date }>(
	data: T[],
	domain: [Date, Date] | undefined
): T[] {
	if (!domain) return data;
	const [start, end] = domain;
	return data.filter((d) => d.date >= start && d.date <= end);
}

// Format xAxis partagé
export function formatXAxis(d: Date): string {
	return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
}

// Format date tooltip partagé
export function formatTooltipDate(date: Date): string {
	return date.toLocaleDateString('fr-FR', {
		day: 'numeric',
		month: 'short',
		hour: '2-digit',
		minute: '2-digit',
		second: '2-digit'
	});
}

// Format bytes per second (for disk I/O and network charts)
export function formatRate(bytesPerSec: number): string {
	if (bytesPerSec === 0) return '0 B/s';
	const units = ['B/s', 'KB/s', 'MB/s', 'GB/s'];
	let value = bytesPerSec;
	let unitIndex = 0;
	while (value >= 1024 && unitIndex < units.length - 1) {
		value /= 1024;
		unitIndex++;
	}
	return `${value.toFixed(1)} ${units[unitIndex]}`;
}

// Format temperature in Celsius
export function formatTemperature(celsius: number): string {
	if (celsius === 0) return 'N/A';
	return `${celsius.toFixed(1)}°C`;
}

// Padding partagé
export const CHART_PADDING_PERCENT = { left: 40, bottom: 24, top: 8, right: 8 };
export const CHART_PADDING_BYTES = { left: 70, bottom: 24, top: 8, right: 8 };
export const CHART_PADDING_RATE = { left: 70, bottom: 24, top: 8, right: 8 };
