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
	return `${Number.isInteger(value) ? value : value.toFixed(1)} ${units[unitIndex]}`;
}

// Format temperature in Celsius
export function formatTemperature(celsius: number): string {
	if (celsius === 0) return 'N/A';
	return `${celsius.toFixed(1)}°C`;
}
