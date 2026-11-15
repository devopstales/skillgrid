import { defineConfig } from "vitest/config";
import { fileURLToPath } from "node:url";

export default defineConfig({
  resolve: {
    // The "obsidian" package is type-only at build time (esbuild marks it
    // external) and its package.json has an empty "main", so vite cannot
    // resolve it in tests. Alias it to a value stub for the few runtime
    // imports (Plugin, TFile, Notice, Modal, Setting).
    alias: {
      obsidian: fileURLToPath(new URL("./tests/mocks/obsidian.ts", import.meta.url)),
    },
  },
});
