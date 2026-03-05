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

// Padding partagé
export const CHART_PADDING_PERCENT = { left: 40, bottom: 24, top: 8, right: 8 };
export const CHART_PADDING_BYTES = { left: 70, bottom: 24, top: 8, right: 8 };
