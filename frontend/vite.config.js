import tailwindcss from '@tailwindcss/vite';
import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';
import { createRequire } from 'module';

const require = createRequire(import.meta.url);
const pkg = require('./package.json');

export default defineConfig({
	plugins: [tailwindcss(), sveltekit()],
	define: {
		__APP_VERSION__: JSON.stringify(pkg.version)
	},

	test: {
		include: ['src/**/*.test.ts'],
		environment: 'node'
	},

	server: {
		port: 5173,

		proxy: {
			'/api': {
				target: process.env.BACKEND_URL || 'http://localhost:8080',
				changeOrigin: true,
				configure: (proxy) => {
					proxy.on('proxyReq', (proxyReq) => {
						proxyReq.setHeader('Origin', 'http://localhost:5173');
					});
				}
			}
		}
	}
});
