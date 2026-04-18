import { describe, it, expect, beforeEach, vi } from 'vitest';
import { get } from 'svelte/store';

// Stub localStorage before importing the store (it reads localStorage at init)
const localStorageMock: Record<string, string> = {};
vi.stubGlobal('localStorage', {
	getItem: (key: string) => localStorageMock[key] ?? null,
	setItem: (key: string, value: string) => { localStorageMock[key] = value; },
	removeItem: (key: string) => { delete localStorageMock[key]; },
});

import { uiStore } from './ui';

describe('uiStore', () => {
	beforeEach(() => {
		uiStore.reset();
		localStorageMock['metricsCollapsed'] = 'false';
	});

	it('has correct initial state', () => {
		const state = get(uiStore);
		expect(state.loading).toBe(false);
		expect(state.rightSidebarOpen).toBe(false);
		expect(state.metricsCollapsed).toBe(false);
	});

	it('setLoading(true) sets loading to true', () => {
		uiStore.setLoading(true);
		expect(get(uiStore).loading).toBe(true);
	});

	it('setLoading(false) sets loading back to false', () => {
		uiStore.setLoading(true);
		uiStore.setLoading(false);
		expect(get(uiStore).loading).toBe(false);
	});

	it('toggleRightSidebar opens the sidebar', () => {
		uiStore.toggleRightSidebar();
		expect(get(uiStore).rightSidebarOpen).toBe(true);
	});

	it('toggleRightSidebar closes the sidebar', () => {
		uiStore.toggleRightSidebar();
		uiStore.toggleRightSidebar();
		expect(get(uiStore).rightSidebarOpen).toBe(false);
	});

	it('setRightSidebar(true) opens sidebar', () => {
		uiStore.setRightSidebar(true);
		expect(get(uiStore).rightSidebarOpen).toBe(true);
	});

	it('setRightSidebar(false) closes sidebar', () => {
		uiStore.setRightSidebar(true);
		uiStore.setRightSidebar(false);
		expect(get(uiStore).rightSidebarOpen).toBe(false);
	});

	it('toggleMetricsCollapsed collapses metrics', () => {
		uiStore.toggleMetricsCollapsed();
		expect(get(uiStore).metricsCollapsed).toBe(true);
	});

	it('toggleMetricsCollapsed saves to localStorage', () => {
		uiStore.toggleMetricsCollapsed();
		expect(localStorageMock['metricsCollapsed']).toBe('true');
	});

	it('toggleMetricsCollapsed toggles back', () => {
		uiStore.toggleMetricsCollapsed();
		uiStore.toggleMetricsCollapsed();
		expect(get(uiStore).metricsCollapsed).toBe(false);
		expect(localStorageMock['metricsCollapsed']).toBe('false');
	});

	it('reset clears all state', () => {
		uiStore.setLoading(true);
		uiStore.setRightSidebar(true);
		uiStore.toggleMetricsCollapsed();
		uiStore.reset();
		const state = get(uiStore);
		expect(state.loading).toBe(false);
		expect(state.rightSidebarOpen).toBe(false);
		expect(state.metricsCollapsed).toBe(false);
	});
});
