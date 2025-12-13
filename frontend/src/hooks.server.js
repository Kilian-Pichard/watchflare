import { redirect } from '@sveltejs/kit';

const PUBLIC_ROUTES = ['/login'];

export async function handle({ event, resolve }) {
	const token = event.cookies.get('jwt_token');
	const pathname = event.url.pathname;

	const isPublic = PUBLIC_ROUTES.some((route) => pathname === route || pathname.startsWith(route + '/'));

	// Redirect to login if accessing protected route without token
	if (!isPublic && !token) {
		throw redirect(302, '/login');
	}

	// Redirect to dashboard if accessing login with valid token
	if (isPublic && token) {
		throw redirect(302, '/');
	}

	return await resolve(event);
}
