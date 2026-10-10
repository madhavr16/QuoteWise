import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// In Docker Compose the API host is "api"; natively it is localhost.
// The browser only ever talks to the Go API — never the AI service —
// so only API paths are proxied (see docs/architecture.md §1).
const apiTarget = process.env.API_URL ?? "http://localhost:8080";

export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      "/api": { target: apiTarget, changeOrigin: true },
      "/healthz": { target: apiTarget, changeOrigin: true },
      "/readyz": { target: apiTarget, changeOrigin: true },
    },
  },
});
