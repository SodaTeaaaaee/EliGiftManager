import { fileURLToPath, URL } from "node:url";
import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";

// https://vite.dev/config/
export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      "@": fileURLToPath(new URL("./src", import.meta.url)),
    },
  },
  server: {
    host: "127.0.0.1",
    port: 5173,
    strictPort: true,
    // The Deno-backed fs watcher crashes when chokidar constructs a watch on
    // a path that was just deleted. Dev-flow paths that do exactly that:
    // `wails3 generate bindings -clean=true` stages output in a
    // frontend/.bindings-tmp-<rand> dir and swaps it into frontend/bindings/
    // (so both the tmp dir and the target dir churn), and vite's own temp
    // config bundle (vite.config.ts.timestamp-*.mjs) is unlinked right after
    // loading while the initial scan may still see it. Keep them all out of
    // the dev watch.
    watch: {
      ignored: [
        "**/bindings",
        "**/bindings/**",
        "**/.bindings-tmp-*",
        "**/.bindings-tmp-*/**",
        "**/vite.config.ts.timestamp-*",
      ],
    },
  },
  build: {
    emptyOutDir: true,
  },
  preview: {
    host: "127.0.0.1",
    port: 4173,
    strictPort: true,
  },
});
