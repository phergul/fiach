import { fileURLToPath, URL } from "node:url";

import { defineConfig } from "vite";

import react from "@vitejs/plugin-react";
import wails from "@wailsio/runtime/plugins/vite";

// https://vitejs.dev/config/
export default defineConfig({
  resolve: {
    alias: {
      "@app": fileURLToPath(new URL("./src/App.tsx", import.meta.url)),
      "@bindings": fileURLToPath(new URL("./bindings", import.meta.url)),
      "@components": fileURLToPath(new URL("./src/components", import.meta.url)),
      "@hooks": fileURLToPath(new URL("./src/hooks", import.meta.url)),
      "@pages": fileURLToPath(new URL("./src/pages", import.meta.url)),
      "@styles": fileURLToPath(new URL("./src/styles", import.meta.url)),
      "@utils": fileURLToPath(new URL("./src/utils", import.meta.url)),
    },
  },
  plugins: [react(), wails("./bindings")],
});
