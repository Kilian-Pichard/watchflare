import type {
	LoginResponse,
	RegisterResponse,
	CreateServerResponse,
	RegenerateTokenResponse,
	GetServerResponse,
	ListServersResponse,
	GetMetricsResponse,
	GetAggregatedMetricsResponse,
	GetDroppedMetricsResponse,
	GetPackagesResponse,
	GetPackageStatsResponse,
	GetPackageCollectionsResponse,
	GetPackageHistoryResponse,
	CurrentUserResponse,
	MetricsQueryParams
} from './types';

const API_BASE_URL = 'http://localhost:8080';

interface ApiRequestOptions extends RequestInit {
	headers?: Record<string, string>;
}

// Custom API error class
export class ApiError extends Error {
	constructor(
		public status: number,
		public statusText: string,
		public data?: { error?: string; message?: string }
	) {
		super(data?.error || data?.message || statusText || 'API request failed');
		this.name = 'ApiError';
	}

	get isAuthError(): boolean {
		return this.status === 401;
	}

	get isForbidden(): boolean {
		return this.status === 403;
	}

	get isNotFound(): boolean {
		return this.status === 404;
	}

	get isServerError(): boolean {
		return this.status >= 500;
	}
}

// Check if initial setup is required (no users exist)
async function checkSetupRequired(): Promise<boolean> {
	try {
		const response = await fetch(`${API_BASE_URL}/auth/setup-required`, {
			credentials: 'include'
		});
		const data = await response.json();
		return data.setup_required;
	} catch (err) {
		console.error('Failed to check setup status:', err);
		return false;
	}
}

// Handle authentication errors
async function handleAuthError(): Promise<never> {
	try {
		const setupRequired = await checkSetupRequired();
		if (setupRequired) {
			// No users exist, redirect to registration
			window.location.href = '/register';
			throw new ApiError(401, 'Unauthorized', { message: 'Redirecting to registration' });
		} else {
			// Users exist but not authenticated, redirect to login
			window.location.href = '/login';
			throw new ApiError(401, 'Unauthorized', { message: 'Redirecting to login' });
		}
	} catch (err) {
		// If checking setup status fails, default to login
		window.location.href = '/login';
		throw new ApiError(401, 'Unauthorized', { message: 'Redirecting to login' });
	}
}

// Make API request with credentials (cookies sent automatically)
async function apiRequest<T>(endpoint: string, options: ApiRequestOptions = {}): Promise<T> {
	const headers = {
		'Content-Type': 'application/json',
		...options.headers
	};

	let response: Response;
	let data: unknown;

	try {
		response = await fetch(`${API_BASE_URL}${endpoint}`, {
			...options,
			headers,
			credentials: 'include' // Important: send cookies with requests
		});

		// Try to parse JSON response
		try {
			data = await response.json();
		} catch (parseError) {
			// If JSON parsing fails, create error with status text
			if (!response.ok) {
				throw new ApiError(response.status, response.statusText);
			}
			throw new ApiError(500, 'Invalid response format');
		}
	} catch (err) {
		// Network or fetch errors
		if (err instanceof ApiError) {
			throw err;
		}
		// Network error (e.g., no internet, CORS, etc.)
		throw new ApiError(0, 'Network error', {
			message: err instanceof Error ? err.message : 'Failed to connect to server'
		});
	}

	if (!response.ok) {
		// Handle authentication errors
		if (response.status === 401) {
			await handleAuthError();
		}

		// Throw API error for other cases
		throw new ApiError(
			response.status,
			response.statusText,
			data as { error?: string; message?: string }
		);
	}

	return data as T;
}

// Auth API calls
export async function register(email: string, password: string): Promise<RegisterResponse> {
	return apiRequest<RegisterResponse>('/auth/register', {
		method: 'POST',
		body: JSON.stringify({ email, password })
	});
}

export async function login(email: string, password: string): Promise<LoginResponse> {
	return apiRequest<LoginResponse>('/auth/login', {
		method: 'POST',
		body: JSON.stringify({ email, password })
	});
}

export async function logout(): Promise<{ message: string }> {
	return apiRequest<{ message: string }>('/auth/logout', {
		method: 'POST'
	});
}

export async function changePassword(currentPassword: string, newPassword: string): Promise<{ message: string }> {
	return apiRequest<{ message: string }>('/auth/change-password', {
		method: 'PUT',
		body: JSON.stringify({
			current_password: currentPassword,
			new_password: newPassword
		})
	});
}

// Server API calls
export async function listServers(): Promise<ListServersResponse> {
	return apiRequest<ListServersResponse>('/servers');
}

export async function getServer(id: string): Promise<GetServerResponse> {
	return apiRequest<GetServerResponse>(`/servers/${id}`);
}

export async function createServer(
	name: string,
	configuredIP?: string,
	allowAnyIP?: boolean
): Promise<CreateServerResponse> {
	return apiRequest<CreateServerResponse>('/servers', {
		method: 'POST',
		body: JSON.stringify({
			name,
			configured_ip: configuredIP,
			allow_any_ip: allowAnyIP
		})
	});
}

export async function deleteServer(id: string): Promise<{ message: string }> {
	return apiRequest<{ message: string }>(`/servers/${id}`, {
		method: 'DELETE'
	});
}

export async function regenerateToken(id: string): Promise<RegenerateTokenResponse> {
	return apiRequest<RegenerateTokenResponse>(`/servers/${id}/regenerate-token`, {
		method: 'POST'
	});
}

export async function validateIP(id: string, selectedIP: string): Promise<{ message: string }> {
	return apiRequest<{ message: string }>(`/servers/${id}/validate-ip`, {
		method: 'PUT',
		body: JSON.stringify({ selected_ip: selectedIP })
	});
}

export async function updateConfiguredIP(id: string, newIP: string): Promise<{ message: string }> {
	return apiRequest<{ message: string }>(`/servers/${id}/change-ip`, {
		method: 'PUT',
		body: JSON.stringify({ new_ip: newIP })
	});
}

export async function ignoreIPMismatch(id: string): Promise<{ message: string }> {
	return apiRequest<{ message: string }>(`/servers/${id}/ignore-ip-mismatch`, {
		method: 'PUT'
	});
}

export async function dismissReactivation(id: string): Promise<{ message: string }> {
	return apiRequest<{ message: string }>(`/servers/${id}/dismiss-reactivation`, {
		method: 'PUT'
	});
}

// User preferences API calls
export async function getCurrentUser(): Promise<CurrentUserResponse> {
	return apiRequest<CurrentUserResponse>('/auth/user');
}

export async function updatePreferences(
	defaultTimeRange: string,
	theme: string
): Promise<{ message: string }> {
	return apiRequest<{ message: string }>('/auth/preferences', {
		method: 'PUT',
		body: JSON.stringify({
			default_time_range: defaultTimeRange,
			theme: theme
		})
	});
}

// Metrics API calls
export async function getServerMetrics(
	serverId: string,
	params: MetricsQueryParams = {}
): Promise<GetMetricsResponse> {
	const queryParams = new URLSearchParams();

	// Support new time_range parameter (1h, 12h, 24h, 7d, 30d)
	// Backend handles start/end/interval calculation automatically
	if (params.time_range) {
		queryParams.append('time_range', params.time_range);
	}
	if (params.limit) queryParams.append('limit', params.limit.toString());
	if (params.offset) queryParams.append('offset', params.offset.toString());

	const query = queryParams.toString();
	return apiRequest<GetMetricsResponse>(
		`/servers/${serverId}/metrics${query ? '?' + query : ''}`
	);
}

// Get dropped metrics summary for the last 24 hours
export async function getDroppedMetrics(): Promise<GetDroppedMetricsResponse> {
	return apiRequest<GetDroppedMetricsResponse>('/servers/dropped-metrics');
}

// Get aggregated metrics from all online servers
export async function getAggregatedMetrics(
	timeRange?: string
): Promise<GetAggregatedMetricsResponse> {
	const queryParams = new URLSearchParams();
	if (timeRange) {
		queryParams.append('time_range', timeRange);
	}
	const query = queryParams.toString();
	return apiRequest<GetAggregatedMetricsResponse>(
		`/servers/metrics/aggregated${query ? '?' + query : ''}`
	);
}

// Package API calls
interface PackageQueryParams {
	limit?: number;
	offset?: number;
	package_manager?: string;
	search?: string;
}

export async function getServerPackages(
	serverId: string,
	params: PackageQueryParams = {}
): Promise<GetPackagesResponse> {
	const queryParams = new URLSearchParams();
	if (params.limit) queryParams.append('limit', params.limit.toString());
	if (params.offset) queryParams.append('offset', params.offset.toString());
	if (params.package_manager) queryParams.append('package_manager', params.package_manager);
	if (params.search) queryParams.append('search', params.search);

	const query = queryParams.toString();
	return apiRequest<GetPackagesResponse>(
		`/servers/${serverId}/packages${query ? '?' + query : ''}`
	);
}

export async function getPackageStats(serverId: string): Promise<GetPackageStatsResponse> {
	return apiRequest<GetPackageStatsResponse>(`/servers/${serverId}/packages/stats`);
}

interface CollectionQueryParams {
	limit?: number;
	offset?: number;
}

export async function getPackageCollections(
	serverId: string,
	params: CollectionQueryParams = {}
): Promise<GetPackageCollectionsResponse> {
	const queryParams = new URLSearchParams();
	if (params.limit) queryParams.append('limit', params.limit.toString());
	if (params.offset) queryParams.append('offset', params.offset.toString());

	const query = queryParams.toString();
	return apiRequest<GetPackageCollectionsResponse>(
		`/servers/${serverId}/packages/collections${query ? '?' + query : ''}`
	);
}

export async function getPackageHistory(
	serverId: string,
	params: CollectionQueryParams = {}
): Promise<GetPackageHistoryResponse> {
	const queryParams = new URLSearchParams();
	if (params.limit) queryParams.append('limit', params.limit.toString());
	if (params.offset) queryParams.append('offset', params.offset.toString());

	const query = queryParams.toString();
	return apiRequest(`/servers/${serverId}/packages/history${query ? '?' + query : ''}`);
}
