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
