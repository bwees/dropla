import { sveltekit } from '@sveltejs/kit/vite';
import tailwindcss from '@tailwindcss/vite';
import { defineConfig } from 'vite';

const apiTarget = process.env.API_PROXY_TARGET ?? 'http://127.0.0.1:3000/api';

export default defineConfig({
	plugins: [tailwindcss(), sveltekit()],
	server: {
		proxy: {
			// tus builds the absolute upload URL it returns in Location from the Host
			// header, so changeOrigin must stay false: rewriting Host to the Go server
			// would make that URL point at :3000 and the browser's follow-up PATCH
			// would be cross-origin. Keeping the original Host means the URL comes
			// back pointing at this dev server, which proxies it correctly.
			'/api': {
				target: apiTarget,
				ws: true,
				changeOrigin: false,
				rewrite: (path) => path.replace(/^\/api/, '')
			},
			// Public download and CLI-upload endpoints live outside /api because they
			// are plain browser navigations that must carry the unlock cookie. Only the
			// sub-paths are proxied; /s/<token> itself is a SvelteKit route.
			'^/s/[^/]+/(all\\.zip|f/.+)$': {
				target: apiTarget.replace(/\/api\/?$/, ''),
				changeOrigin: false
			}
		}
	}
});
