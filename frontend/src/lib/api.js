const API_BASE_URL = 'http://localhost:8080';

// Check if initial setup is required (no users exist)
async function checkSetupRequired() {
	try {
		const response = await fetch(`${API_BASE_URL}/auth/setup-required`, {
			credentials: 'include'
		});
		const data = await response.json();
		return data.setup_required;
	} catch (err) {
		return false;
	}
}

// Make API request with credentials (cookies sent automatically)
async function apiRequest(endpoint, options = {}) {
	const headers = {
		'Content-Type': 'application/json',
		...options.headers
	};

	const response = await fetch(`${API_BASE_URL}${endpoint}`, {
		...options,
		headers,
		credentials: 'include' // Important: send cookies with requests
	});

	const data = await response.json();

	if (!response.ok) {
		// If 401 Unauthorized, check if setup is required
		if (response.status === 401) {
			const setupRequired = await checkSetupRequired();
			if (setupRequired) {
				// No users exist, redirect to registration
				window.location.href = '/register';
				return; // Don't throw error, just redirect
			} else {
				// Users exist but not authenticated, redirect to login
				window.location.href = '/login';
				return; // Don't throw error, just redirect
			}
		}
		throw new Error(data.error || 'API request failed');
	}

	return data;
}

// Auth API calls
export async function register(email, password) {
	return apiRequest('/auth/register', {
		method: 'POST',
		body: JSON.stringify({ email, password })
	});
}

export async function login(email, password) {
	return apiRequest('/auth/login', {
		method: 'POST',
		body: JSON.stringify({ email, password })
	});
}

export async function logout() {
	return apiRequest('/auth/logout', {
		method: 'POST'
	});
}

export async function changePassword(currentPassword, newPassword) {
	return apiRequest('/auth/change-password', {
		method: 'PUT',
		body: JSON.stringify({
			current_password: currentPassword,
			new_password: newPassword
		})
	});
}

// Server API calls
export async function listServers() {
	return apiRequest('/servers');
}

export async function getServer(id) {
	return apiRequest(`/servers/${id}`);
}

export async function createServer(name, configuredIP, allowAnyIP) {
	return apiRequest('/servers', {
		method: 'POST',
		body: JSON.stringify({
			name,
			configured_ip: configuredIP,
			allow_any_ip: allowAnyIP
		})
	});
}

export async function deleteServer(id) {
	return apiRequest(`/servers/${id}`, {
		method: 'DELETE'
	});
}

export async function regenerateToken(id) {
	return apiRequest(`/servers/${id}/regenerate-token`, {
		method: 'POST'
	});
}

export async function validateIP(id, selectedIP) {
	return apiRequest(`/servers/${id}/validate-ip`, {
		method: 'PUT',
		body: JSON.stringify({ selected_ip: selectedIP })
	});
}

export async function updateConfiguredIP(id, newIP) {
	return apiRequest(`/servers/${id}/change-ip`, {
		method: 'PUT',
		body: JSON.stringify({ new_ip: newIP })
	});
}

export async function ignoreIPMismatch(id) {
	return apiRequest(`/servers/${id}/ignore-ip-mismatch`, {
		method: 'PUT'
	});
}

// User preferences API calls
export async function getCurrentUser() {
	return apiRequest('/auth/user');
}

export async function updatePreferences(defaultTimeRange, theme) {
	return apiRequest('/auth/preferences', {
		method: 'PUT',
		body: JSON.stringify({
			default_time_range: defaultTimeRange,
			theme: theme
		})
	});
}

// Metrics API calls
export async function getServerMetrics(serverId, params = {}) {
	const queryParams = new URLSearchParams();
	if (params.start) queryParams.append('start', params.start);
	if (params.end) queryParams.append('end', params.end);
	if (params.interval) queryParams.append('interval', params.interval);

	const query = queryParams.toString();
	return apiRequest(`/servers/${serverId}/metrics${query ? '?' + query : ''}`);
}
