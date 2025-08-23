import js from '@eslint/js';
import tseslint from 'typescript-eslint';
import eslintConfigPrettier from 'eslint-config-prettier';

export default [
  // Ignore build artifacts across workspaces
  {
    ignores: [
      '**/dist/**',
      '**/build/**',
      '**/coverage/**',
      '**/.turbo/**',
      '**/node_modules/**',
    ],
  },

  // JS/TS recommended rules
  js.configs.recommended,
  ...tseslint.configs.recommended,

  // Your project rules (add/override here)
  {
    files: ['**/*.{ts,tsx,js}'],
    languageOptions: {
      ecmaVersion: 2022,
      sourceType: 'module',
    },
    rules: {
      // Example: allow console in server code during dev
      // 'no-console': ['warn', { allow: ['warn', 'error'] }],
    },
  },

  // Turn off rules that conflict with Prettier — must be last
  eslintConfigPrettier,
];
