import { createRequire } from 'node:module';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import react from '@vitejs/plugin-react';
import { defineConfig, loadEnv } from 'vite';

const webRoot = path.dirname(fileURLToPath(import.meta.url));
const requireFromWeb = createRequire(path.join(webRoot, 'package.json'));

/** Deep import is not always resolved by Vite; pin to the real file under node_modules. */
function resolveElkBundled(): string {
  try {
    return requireFromWeb.resolve('elkjs/lib/elk.bundled.js');
  } catch {
    return requireFromWeb.resolve('elkjs/lib/elk.bundled');
  }
}

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '');
  const apiUrl = env.VITE_API_URL ?? 'http://localhost:8080';
  const elkBundled = resolveElkBundled();

  return {
    plugins: [react()],
    worker: {
      format: 'es',
    },
    resolve: {
      alias: {
        'elkjs/lib/elk.bundled.js': elkBundled,
      },
    },
    optimizeDeps: {
      include: [elkBundled],
    },
    server: {
      port: 5173,
      strictPort: true,
      /** Same-origin `/api/*` in dev → backend (no CORS, works for localhost and 127.0.0.1). */
      proxy: {
        '/api': {
          target: env.VITE_API_PROXY_TARGET ?? 'http://localhost:8080',
          changeOrigin: true,
          rewrite: (path) => path.replace(/^\/api/, ''),
        },
      },
    },
    define: {
      __CODEATLAS_API_URL__: JSON.stringify(apiUrl),
    },
  };
});
