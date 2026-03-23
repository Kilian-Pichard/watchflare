// Format date tooltip partagé
export function formatTooltipDate(date: Date, timeFormat: '12h' | '24h' = '24h'): string {
	return date.toLocaleString('en-US', {
		day: 'numeric',
		month: 'short',
		hour: '2-digit',
		minute: '2-digit',
		second: '2-digit',
		hour12: timeFormat === '12h',
	});
}

// Format bytes per second (for disk I/O and network charts)
export function formatRate(bytesPerSec: number, unit: 'bytes' | 'bits' = 'bytes'): string {
	if (unit === 'bits') {
		const bps = bytesPerSec * 8;
		if (bps <= 0) return '0 bps';
		const units = ['bps', 'Kbps', 'Mbps', 'Gbps'];
		let value = bps;
		let i = 0;
		while (value >= 1000 && i < units.length - 1) { value /= 1000; i++; }
		return `${Number.isInteger(value) ? value : value.toFixed(1)} ${units[i]}`;
	}
	if (bytesPerSec <= 0) return '0 B/s';
	const units = ['B/s', 'KB/s', 'MB/s', 'GB/s'];
	let value = bytesPerSec;
	let i = 0;
	while (value >= 1024 && i < units.length - 1) { value /= 1024; i++; }
	return `${Number.isInteger(value) ? value : value.toFixed(1)} ${units[i]}`;
}

// Format temperature (Celsius or Fahrenheit)
export function formatTemperature(celsius: number, unit: 'celsius' | 'fahrenheit' = 'celsius'): string {
	if (celsius === 0) return 'N/A';
	if (unit === 'fahrenheit') {
		const fahrenheit = celsius * 9 / 5 + 32;
		return `${fahrenheit.toFixed(1)}°F`;
	}
	return `${celsius.toFixed(1)}°C`;
}
