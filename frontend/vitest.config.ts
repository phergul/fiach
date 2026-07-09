import { fileURLToPath, URL } from 'node:url';

import { defineConfig } from 'vitest/config';

import react from '@vitejs/plugin-react';

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@bindings': fileURLToPath(new URL('./bindings', import.meta.url)),
      '@components': fileURLToPath(new URL('./src/components', import.meta.url)),
      '@hooks': fileURLToPath(new URL('./src/hooks/index.ts', import.meta.url)),
      '@pages': fileURLToPath(new URL('./src/pages/index.ts', import.meta.url)),
      '@styles': fileURLToPath(new URL('./src/styles', import.meta.url)),
      '@wailsio/runtime': fileURLToPath(new URL('./src/test/wailsRuntimeMock.ts', import.meta.url)),
      '@utils': fileURLToPath(new URL('./src/utils/index.ts', import.meta.url)),
    },
  },
  test: {
    environment: 'jsdom',
    include: ['src/**/*.test.{ts,tsx}'],
    setupFiles: ['./src/test/setup.ts'],
  },
});
