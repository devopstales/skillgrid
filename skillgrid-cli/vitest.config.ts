import path from "node:path";
import { fileURLToPath } from "node:url";
import { defineConfig } from "vitest/config";

const pkgRoot = path.dirname(fileURLToPath(import.meta.url));

export default defineConfig({
  root: pkgRoot,
  test: {
    environment: "node",
    include: ["src/dashboard/**/*.test.ts"]
  }
});
