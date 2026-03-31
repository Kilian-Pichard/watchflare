import { describe, it, expect } from 'vitest';
import {
	formatBytes,
	formatPercent,
	formatUptime,
	getMetricClass,
	getStatusClass,
	formatRelativeTime,
	formatDateTime,
	getIntervalForTimeRange,
	getTimeRangeTimestamps
} from './utils';

describe('formatBytes', () => {
	it('formats 0 bytes', () => {
		expect(formatBytes(0)).toBe('0 B');
	});

	it('formats bytes', () => {
		expect(formatBytes(500)).toBe('500 B');
	});

	it('formats kilobytes', () => {
		expect(formatBytes(1024)).toBe('1 KB');
	});

	it('formats megabytes', () => {
		expect(formatBytes(1048576)).toBe('1 MB');
	});

	it('formats gigabytes', () => {
		expect(formatBytes(1073741824)).toBe('1 GB');
	});

	it('formats terabytes', () => {
		expect(formatBytes(1099511627776)).toBe('1 TB');
	});

	it('formats with decimals', () => {
		expect(formatBytes(1536)).toBe('1.5 KB');
	});
});

describe('formatPercent', () => {
	it('formats integer percentage', () => {
		expect(formatPercent(50)).toBe('50%');
	});

	it('formats decimal percentage with one decimal place', () => {
		expect(formatPercent(33.33)).toBe('33.3%');
	});

	it('formats zero', () => {
		expect(formatPercent(0)).toBe('0%');
	});

	it('formats 100%', () => {
		expect(formatPercent(100)).toBe('100%');
	});
});

describe('formatUptime', () => {
	it('formats minutes only', () => {
		expect(formatUptime(300)).toBe('5m');
	});

	it('formats hours and minutes', () => {
		expect(formatUptime(3660)).toBe('1h 1m');
	});

	it('formats days and hours', () => {
		expect(formatUptime(90000)).toBe('1d 1h');
	});

	it('formats zero seconds', () => {
		expect(formatUptime(0)).toBe('0m');
	});
});

describe('getMetricClass', () => {
	it('returns foreground below 70%', () => {
		expect(getMetricClass(50)).toBe('text-foreground');
	});

	it('returns warning at 70%', () => {
		expect(getMetricClass(70)).toContain('text-warning');
	});

	it('returns warning between 70-89%', () => {
		expect(getMetricClass(85)).toContain('text-warning');
	});

	it('returns danger at 90%', () => {
		expect(getMetricClass(90)).toContain('text-danger');
	});

	it('returns danger above 90%', () => {
		expect(getMetricClass(99)).toContain('text-danger');
	});
});

describe('getStatusClass', () => {
	it('returns success classes for online', () => {
		expect(getStatusClass('online')).toContain('bg-success');
	});

	it('returns danger classes for offline', () => {
		expect(getStatusClass('offline')).toContain('bg-danger');
	});

	it('returns warning classes for pending', () => {
		expect(getStatusClass('pending')).toContain('bg-warning');
	});

	it('returns muted classes for unknown status', () => {
		expect(getStatusClass('unknown')).toContain('bg-muted');
	});
});

describe('formatRelativeTime', () => {
	it('returns "Never" for null/undefined', () => {
		expect(formatRelativeTime(null)).toBe('Never');
		expect(formatRelativeTime(undefined)).toBe('Never');
	});

	it('returns seconds ago for recent timestamps', () => {
		const now = new Date().toISOString();
		expect(formatRelativeTime(now)).toBe('0s ago');
	});
});

describe('formatDateTime', () => {
	it('returns "-" for null/undefined', () => {
		expect(formatDateTime(null)).toBe('-');
		expect(formatDateTime(undefined)).toBe('-');
	});

	it('formats a valid date', () => {
		const result = formatDateTime('2024-01-15T12:00:00Z');
		expect(result).toBeTruthy();
		expect(result).not.toBe('-');
	});
});

describe('getIntervalForTimeRange', () => {
	it('returns empty for 1h (raw data)', () => {
		expect(getIntervalForTimeRange('1h')).toBe('');
	});

	it('returns 10m for 12h', () => {
		expect(getIntervalForTimeRange('12h')).toBe('10m');
	});

	it('returns 15m for 24h', () => {
		expect(getIntervalForTimeRange('24h')).toBe('15m');
	});

	it('returns 2h for 7d', () => {
		expect(getIntervalForTimeRange('7d')).toBe('2h');
	});

	it('returns 8h for 30d', () => {
		expect(getIntervalForTimeRange('30d')).toBe('8h');
	});
});

describe('getTimeRangeTimestamps', () => {
	it('returns start and end for valid range', () => {
		const result = getTimeRangeTimestamps('1h');
		expect(result).not.toBeNull();
		if (result) {
			const start = new Date(result.start);
			const end = new Date(result.end);
			const diffMs = end.getTime() - start.getTime();
			// Should be approximately 1 hour (3600000ms)
			expect(diffMs).toBeGreaterThan(3599000);
			expect(diffMs).toBeLessThan(3601000);
		}
	});

	it('returns null for invalid range', () => {
		const result = getTimeRangeTimestamps('invalid' as any);
		expect(result).toBeNull();
	});
});
