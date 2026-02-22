import { describe, it, expect, vi, beforeEach } from 'vitest';
import { get } from 'svelte/store';

// Mock the api module before importing the store
vi.mock('$lib/api', () => ({
	getServerMetrics: vi.fn()
}));

// Mock the constants module
vi.mock('$lib/constants', () => ({
	MAX_METRICS_POINTS_DASHBOARD: 5
}));

// Mock the utils module
vi.mock('$lib/utils', () => ({
	logger: { error: vi.fn(), warn: vi.fn(), log: vi.fn() }
}));

import { metricsStore, metricsData } from './metrics';

function fakeMetric(serverId: string, cpu: number) {
	return {
		server_id: serverId,
		timestamp: new Date().toISOString(),
		cpu_usage_percent: cpu,
		memory_total_bytes: 1000,
		memory_used_bytes: 500,
		disk_total_bytes: 2000,
		disk_used_bytes: 1000
	};
}

describe('metricsStore', () => {
	beforeEach(() => {
		metricsStore.clear();
	});

	it('starts with empty data', () => {
		const state = get(metricsStore);
		expect(state.data).toEqual({});
		expect(state.loading).toEqual({});
		expect(state.error).toBeNull();
	});

	it('updates server metrics via updateServerMetrics', () => {
		const metric = fakeMetric('s1', 50);
		metricsStore.updateServerMetrics('s1', metric);
		const data = get(metricsData);
		expect(data['s1']).toHaveLength(1);
		expect(data['s1'][0].cpu_usage_percent).toBe(50);
	});

	it('caps metrics at MAX_METRICS_POINTS_DASHBOARD', () => {
		// MAX is mocked to 5
		for (let i = 0; i < 8; i++) {
			metricsStore.updateServerMetrics('s1', fakeMetric('s1', i * 10));
		}
		const data = get(metricsData);
		expect(data['s1']).toHaveLength(5);
		// Should keep the last 5 (i=3..7 → 30,40,50,60,70)
		expect(data['s1'][0].cpu_usage_percent).toBe(30);
		expect(data['s1'][4].cpu_usage_percent).toBe(70);
	});

	it('keeps metrics separate per server', () => {
		metricsStore.updateServerMetrics('s1', fakeMetric('s1', 10));
		metricsStore.updateServerMetrics('s2', fakeMetric('s2', 20));
		const data = get(metricsData);
		expect(data['s1']).toHaveLength(1);
		expect(data['s2']).toHaveLength(1);
		expect(data['s1'][0].cpu_usage_percent).toBe(10);
		expect(data['s2'][0].cpu_usage_percent).toBe(20);
	});

	it('clears metrics for a specific server', () => {
		metricsStore.updateServerMetrics('s1', fakeMetric('s1', 10));
		metricsStore.updateServerMetrics('s2', fakeMetric('s2', 20));
		metricsStore.clearForServer('s1');
		const data = get(metricsData);
		expect(data['s1']).toBeUndefined();
		expect(data['s2']).toHaveLength(1);
	});

	it('clears all metrics', () => {
		metricsStore.updateServerMetrics('s1', fakeMetric('s1', 10));
		metricsStore.updateServerMetrics('s2', fakeMetric('s2', 20));
		metricsStore.clear();
		expect(get(metricsData)).toEqual({});
	});

	it('getForServer returns empty array for unknown server', () => {
		expect(metricsStore.getForServer('unknown')).toEqual([]);
	});
});
