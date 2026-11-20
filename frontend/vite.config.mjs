import { fileURLToPath } from 'node:url';
import vue from '@vitejs/plugin-vue';
import { defineConfig, loadEnv } from 'vite';

export default defineConfig(({ mode }) => {
  const { DEV_API_PROXY_TARGET } = loadEnv(mode, import.meta.dirname, 'DEV_API_PROXY_TARGET');
  const localAPI = new URL(DEV_API_PROXY_TARGET || 'http://127.0.0.1:8080');
  if (localAPI.protocol !== 'http:' ||
      !['127.0.0.1', 'localhost', '[::1]', 'backend'].includes(localAPI.hostname) ||
      localAPI.username || localAPI.password || localAPI.pathname !== '/' || localAPI.search || localAPI.hash) {
    throw new Error('DEV_API_PROXY_TARGET must be a local HTTP origin');
  }

  return {
    plugins: [vue()],
    // GitHub Actions and Docker Compose already publish these client variables.
    envPrefix: 'VUE_APP_',
    resolve: {
      alias: { '@': fileURLToPath(new URL('./src', import.meta.url)) },
    },
    server: {
      host: '0.0.0.0',
      port: 4000,
      strictPort: true,
      proxy: {
        '^/api/auth(?:/|$)': {
          target: localAPI.origin,
          rewrite: (path) => path.replace(/^\/api/, ''),
        },
      },
    },
    test: {
      environment: 'jsdom',
      environmentOptions: { jsdom: { url: 'http://localhost/' } },
      globals: true,
      include: ['tests/unit/**/*.spec.js'],
      maxWorkers: 2,
    },
  };
});
