import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { get } from 'svelte/store';
import { toasts } from './toasts';

describe('toasts store', () => {
	beforeEach(() => {
		vi.useFakeTimers();
		toasts.clear();
	});

	afterEach(() => {
		vi.useRealTimers();
	});

	it('starts empty', () => {
		expect(get(toasts)).toEqual([]);
	});

	it('adds a toast', () => {
		toasts.add('Hello', 'info', 0);
		const items = get(toasts);
		expect(items).toHaveLength(1);
		expect(items[0].message).toBe('Hello');
		expect(items[0].type).toBe('info');
	});

	it('removes a toast by id', () => {
		const id = toasts.add('Remove me', 'error', 0);
		expect(get(toasts)).toHaveLength(1);
		toasts.remove(id);
		expect(get(toasts)).toEqual([]);
	});

	it('clears all toasts', () => {
		toasts.add('A', 'info', 0);
		toasts.add('B', 'warning', 0);
		expect(get(toasts)).toHaveLength(2);
		toasts.clear();
		expect(get(toasts)).toEqual([]);
	});

	it('auto-removes toast after duration', () => {
		toasts.add('Temporary', 'info', 3000);
		expect(get(toasts)).toHaveLength(1);
		vi.advanceTimersByTime(3000);
		expect(get(toasts)).toEqual([]);
	});

	it('does not auto-remove when duration is 0', () => {
		toasts.add('Permanent', 'info', 0);
		vi.advanceTimersByTime(10000);
		expect(get(toasts)).toHaveLength(1);
	});

	it('assigns incrementing ids', () => {
		const id1 = toasts.add('First', 'info', 0);
		const id2 = toasts.add('Second', 'info', 0);
		expect(id2).toBeGreaterThan(id1);
	});
});
