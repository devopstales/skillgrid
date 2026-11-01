import { solidPlugin } from "esbuild-plugin-solid";
import { defineConfig } from "tsup";

export default defineConfig({
  entry: { "skillgrid-logo": "skillgrid-logo-tui.tsx" },
  format: ["esm"],
  target: "node22",
  bundle: true,
  splitting: false,
  clean: false,
  minify: false,
  outDir: "..",
  external: [
    "@opencode-ai/plugin",
    "@opencode-ai/plugin/tui",
    "@opencode/plugin",
    "@opencode/plugin/tui",
    "@opentui/core",
    "@opentui/solid",
    "solid-js",
  ],
  esbuildPlugins: [
    solidPlugin({
      solid: { generate: "universal", moduleName: "@opentui/solid" },
    }),
  ],
});
