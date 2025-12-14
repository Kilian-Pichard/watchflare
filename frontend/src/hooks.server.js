import { redirect } from '@sveltejs/kit';

const PUBLIC_ROUTES = ['/login', '/register'];

async function checkSetupRequired() {
	try {
		const response = await fetch('http://localhost:8080/auth/setup-required');
		const data = await response.json();
		return data.setup_required;
	} catch (error) {
		console.error('Failed to check setup status:', error);
		return false;
	}
}

export async function handle({ event, resolve }) {
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
}
