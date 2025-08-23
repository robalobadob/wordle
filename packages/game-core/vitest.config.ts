import { defineConfig } from 'vitest/config';

export default defineConfig({
  test: {
    include: ['src/**/*.{test,spec}.ts'],
    exclude: ['dist', 'node_modules'],
    environment: 'node',
    globals: true, // allows `describe/it/expect` without imports
    reporters: ['default'],
    coverage: {
      enabled: false, // toggle via npm script when wanted
      provider: 'v8',
      reporter: ['text', 'html', 'lcov'],
    },
  },
});
