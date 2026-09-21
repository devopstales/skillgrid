import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  base: './',
  plugins: [react(), tailwindcss()],
  server: {
    proxy: {
      '/api': 'http://127.0.0.1:7438',
    },
  },
  test: {
    // Component tests (decisions view) render real DOM + act on it; the
    // existing pure-logic tests keep running fine under jsdom too.
    environment: 'jsdom',
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
          'vendor-sigma': ['sigma', '@react-sigma/core', 'graphology', 'graphology-layout-forceatlas2', 'graphology-communities-louvain', 'graphology-types'],
          'vendor-elk': ['elkjs'],
          'vendor-cytoscape': ['cytoscape'],
          'vendor-katex': ['katex'],
        },
      },
    },
  },
})
