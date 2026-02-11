import { redirect, type Handle } from '@sveltejs/kit';

const PUBLIC_ROUTES = ['/login', '/register'];

interface SetupStatusCache {
	value: boolean | null;
	timestamp: number;
	ttl: number;
}

// Cache setup status (it rarely changes)
let setupStatusCache: SetupStatusCache = {
	value: null,
	timestamp: 0,
	ttl: 60000 // 1 minute cache
};

async function checkSetupRequired(): Promise<boolean> {
	const now = Date.now();

	// Return cached value if still valid
	if (setupStatusCache.value !== null && (now - setupStatusCache.timestamp) < setupStatusCache.ttl) {
		return setupStatusCache.value;
	}

	try {
		const response = await fetch('http://localhost:8080/auth/setup-required');
		const data = await response.json() as { setup_required: boolean };

		// Update cache
		setupStatusCache.value = data.setup_required;
		setupStatusCache.timestamp = now;

		return data.setup_required;
	} catch (error) {
		console.error('Failed to check setup status:', error);
		// Return cached value if available, otherwise false
		return setupStatusCache.value !== null ? setupStatusCache.value : false;
	}
}

export const handle: Handle = async ({ event, resolve }) => {
	const token = event.cookies.get('jwt_token');
	const pathname = event.url.pathname;

	const isPublic = PUBLIC_ROUTES.some((route) => pathname === route || pathname.startsWith(route + '/'));

	// Check if initial setup is required
	const setupRequired = await checkSetupRequired();

	// Redirect to register if setup is required and accessing login
	if (setupRequired && pathname === '/login') {
		throw redirect(302, '/register');
	}

	// Redirect to login if setup is complete and accessing register
	if (!setupRequired && pathname === '/register' && !token) {
		throw redirect(302, '/login');
	}

	// Redirect to login/register if accessing protected route without token
	if (!isPublic && !token) {
		throw redirect(302, setupRequired ? '/register' : '/login');
	}

	// Redirect to dashboard if accessing login/register with valid token
	if (isPublic && token) {
		throw redirect(302, '/');
	}

	return await resolve(event);
};
