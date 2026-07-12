import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// Build output goes into the Go embed package so `go build` ships the UI
// inside the guten binary (cli/internal/webui/embed.go).
export default defineConfig({
  plugins: [react()],
  base: "./",
  build: {
    outDir: "../internal/webui/dist",
    emptyOutDir: true,
  },
  server: {
    // `npm run dev` proxies API calls to a locally running `guten ui`.
    proxy: {
      "/api": "http://127.0.0.1:4180",
      "/healthz": "http://127.0.0.1:4180",
    },
  },
});
