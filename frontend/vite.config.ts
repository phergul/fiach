import { fileURLToPath, URL } from "node:url";

import { defineConfig } from "vite";

import react from "@vitejs/plugin-react";
import wails from "@wailsio/runtime/plugins/vite";

// https://vitejs.dev/config/
export default defineConfig({
  resolve: {
    alias: {
      "@styles": fileURLToPath(new URL("./src/styles", import.meta.url)),
    },
  },
  plugins: [react(), wails("./bindings")],
});
