/// <reference types="vitest/config" />
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  base: './',
  plugins: [react(), tailwindcss()],
  server: {
    proxy: {
      // Proxy JSON API calls to live mnemonic serve. Bypass HTML navigations so
      // SPA routes (/mnemonic/graph, /docs, /plans, /project/spikes, …) stay on Vite.
      // Swagger is always proxied (never HTML-bypassed) — the iframe must load
      // the Go Swagger UI, not the Vite SPA shell.
      '/swagger': 'http://127.0.0.1:7438',
      '/openapi.yaml': 'http://127.0.0.1:7438',
      ...(Object.fromEntries(
        [
          '/api',
          '/activity',
          '/code',
          '/context',
          '/docs',
          '/git',
          '/health',
          '/memory',
          '/mnemonic',
          '/observations',
          '/plans',
          '/project',
          '/projects',
          '/prototypes',
          '/search',
          '/security',
          '/sessions',
          '/specs',
          '/spikes',
          '/tracker',
          '/web',
        ].map((prefix) => [
          prefix,
          {
            target: 'http://127.0.0.1:7438',
            bypass(req: { headers: { accept?: string }; url?: string }) {
              const accept = req.headers.accept ?? ''
              // SPA navigations (text/html) stay on Vite; JSON/API stay proxied.
              if (accept.includes('text/html')) return req.url
              return undefined
            },
          },
        ]),
      ) as Record<string, { target: string; bypass: (req: { headers: { accept?: string }; url?: string }) => string | undefined }>),
    },
  },
  test: {
    // Component tests (decisions view) render real DOM + act on it; the
    // existing pure-logic tests keep running fine under jsdom too.
    environment: 'jsdom',
    // globals enables @testing-library/react's afterEach auto-cleanup (it
    // looks the cleanup hook up via a global import), so each render() is
    // torn down between tests.
    globals: true,
  },
  build: {
    outDir: '../skillgrid-cli/internal/mnemonic/http/ui/dist',
    emptyOutDir: true,
    // mermaid (~728 kB) is a single monolithic lib that Rollup can't split
    // further; it's dynamic-imported so it's lazy-loaded only when a diagram
    // renders, but it still trips the 500 kB warning. Bump the limit so the
    // build is quiet — the real win is per-library cacheable chunks, not
    // inlining heavy deps into the app bundle.
    chunkSizeWarningLimit: 800,
    rollupOptions: {
      output: {
        entryFileNames: 'assets/[name]-[hash].js',
        chunkFileNames: 'assets/[name]-[hash].js',
        assetFileNames: 'assets/[name]-[hash][extname]',
        // Split heavy third-party deps into their own stable chunks so the app
        // bundle stays small and browser caching works per-library. mermaid is
        // intentionally NOT listed: it's dynamic-imported (MermaidBlock), so
        // Rollup keeps it as its own lazy chunk loaded only on demand.
        manualChunks: {
          'vendor-react': ['react', 'react-dom'],
          'vendor-d3': ['d3'],
          'vendor-elk': ['elkjs'],
          'vendor-cytoscape': ['cytoscape'],
          'vendor-katex': ['katex'],
        },
      },
    },
  },
})
