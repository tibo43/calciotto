import js from '@eslint/js';
import { defineConfig, globalIgnores } from 'eslint/config';
import vue from 'eslint-plugin-vue';
import globals from 'globals';

export default defineConfig([
  globalIgnores(['dist/', 'playwright-report/', 'test-results/', 'loadtest/reports/']),
  js.configs.recommended,
  ...vue.configs['flat/essential'],
  {
    files: ['src/**/*.{js,vue}'],
    languageOptions: { globals: globals.browser },
  },
  {
    files: ['public/sw.js'],
    languageOptions: { globals: { ...globals.serviceworker, module: 'readonly' } },
  },
  {
    files: ['*.config.{js,mjs}', 'scripts/**/*.{js,mjs}', 'loadtest/**/*.js'],
    languageOptions: { globals: globals.node },
  },
  {
    files: ['tests/visual/**/*.js'],
    languageOptions: { globals: { ...globals.node, ...globals.browser } },
  },
  {
    files: ['tests/unit/**/*.spec.js'],
    languageOptions: { globals: { ...globals.browser, ...globals.node, ...globals.vitest } },
  },
]);
