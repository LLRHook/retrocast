import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";
import path from "path";

export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
  server: {
    proxy: {
      "/api": {
        target: process.env.RETROCAST_PROXY_URL || "https://localhost",
        changeOrigin: true,
        secure: false,
      },
      "/gateway": {
        target: process.env.RETROCAST_PROXY_URL || "https://localhost",
        ws: true,
        changeOrigin: true,
        secure: false,
      },
    },
  },
});
