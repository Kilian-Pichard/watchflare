const API_BASE_URL = 'http://localhost:8080';

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

export async function createServer(name, type, configuredIP, allowAnyIP) {
	return apiRequest('/servers', {
		method: 'POST',
		body: JSON.stringify({
			name,
			type,
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
