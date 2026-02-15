import { describe, it, expect, vi, beforeEach } from 'vitest';
import { ApiError } from './api';

describe('ApiError', () => {
	it('creates error with message from data.error', () => {
		const err = new ApiError(400, 'Bad Request', { error: 'Invalid input' });
		expect(err.message).toBe('Invalid input');
		expect(err.status).toBe(400);
	});

	it('creates error with message from data.message', () => {
		const err = new ApiError(500, 'Internal', { message: 'Something broke' });
		expect(err.message).toBe('Something broke');
	});

	it('falls back to statusText', () => {
		const err = new ApiError(404, 'Not Found');
		expect(err.message).toBe('Not Found');
	});

	it('falls back to default message', () => {
		const err = new ApiError(0, '');
		expect(err.message).toBe('API request failed');
	});

	describe('status getters', () => {
		it('isAuthError is true for 401', () => {
			expect(new ApiError(401, 'Unauthorized').isAuthError).toBe(true);
			expect(new ApiError(403, 'Forbidden').isAuthError).toBe(false);
		});

		it('isForbidden is true for 403', () => {
			expect(new ApiError(403, 'Forbidden').isForbidden).toBe(true);
			expect(new ApiError(401, 'Unauthorized').isForbidden).toBe(false);
		});

		it('isNotFound is true for 404', () => {
			expect(new ApiError(404, 'Not Found').isNotFound).toBe(true);
			expect(new ApiError(400, 'Bad Request').isNotFound).toBe(false);
		});

		it('isServerError is true for 500+', () => {
			expect(new ApiError(500, 'Internal').isServerError).toBe(true);
			expect(new ApiError(502, 'Bad Gateway').isServerError).toBe(true);
			expect(new ApiError(499, 'Client').isServerError).toBe(false);
		});
	});
});
