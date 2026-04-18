import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest';
import { get } from 'svelte/store';
import {
	sidebarCollapsed,
	mobileMenuOpen,
	sidebarTransitioning,
	toggleSidebarWithTransition,
	resetSidebar
} from './sidebar';

describe('sidebar stores', () => {
	beforeEach(() => {
		vi.useFakeTimers();
		resetSidebar();
	});

	afterEach(() => {
		vi.useRealTimers();
	});

	it('initial state: all false', () => {
		expect(get(sidebarCollapsed)).toBe(false);
		expect(get(mobileMenuOpen)).toBe(false);
		expect(get(sidebarTransitioning)).toBe(false);
	});

	it('toggleSidebarWithTransition collapses sidebar and sets transitioning', () => {
		toggleSidebarWithTransition();
		expect(get(sidebarCollapsed)).toBe(true);
		expect(get(sidebarTransitioning)).toBe(true);
	});

	it('toggleSidebarWithTransition clears transitioning after 300ms', () => {
		toggleSidebarWithTransition();
		expect(get(sidebarTransitioning)).toBe(true);
		vi.advanceTimersByTime(300);
		expect(get(sidebarTransitioning)).toBe(false);
	});

	it('toggleSidebarWithTransition expands when already collapsed', () => {
		toggleSidebarWithTransition();
		vi.advanceTimersByTime(300);
		toggleSidebarWithTransition();
		expect(get(sidebarCollapsed)).toBe(false);
	});

	it('resetSidebar sets all stores to false and clears pending timer', () => {
		toggleSidebarWithTransition();
		expect(get(sidebarTransitioning)).toBe(true);
		resetSidebar();
		expect(get(sidebarCollapsed)).toBe(false);
		expect(get(mobileMenuOpen)).toBe(false);
		expect(get(sidebarTransitioning)).toBe(false);
		// Timer should be cleared — advancing time should not change state
		vi.advanceTimersByTime(300);
		expect(get(sidebarTransitioning)).toBe(false);
	});

	it('mobileMenuOpen is a writable store', () => {
		mobileMenuOpen.set(true);
		expect(get(mobileMenuOpen)).toBe(true);
		resetSidebar();
		expect(get(mobileMenuOpen)).toBe(false);
	});
});
