import js from '@eslint/js'
import globals from 'globals'
import reactHooks from 'eslint-plugin-react-hooks'
import reactRefresh from 'eslint-plugin-react-refresh'
import tseslint from 'typescript-eslint'
import { defineConfig, globalIgnores } from 'eslint/config'

export default defineConfig([
  // M0.3-5: Tier 3 permanent exemptions
  globalIgnores([
    'dist',
    'tests/e2e/**',
    'src/gen/**',        // Proto generated code
    '**/i18n/**',        // i18n resource files
  ]),
  {
    files: ['**/*.{ts,tsx}'],
    extends: [
      js.configs.recommended,
      tseslint.configs.recommended,
      reactHooks.configs.flat.recommended,
      reactRefresh.configs.vite,
    ],
    languageOptions: {
      ecmaVersion: 2020,
      globals: globals.browser,
    },
    rules: {
      // M0.3-4: no-explicit-any enforced (baseline waiver via CI --new-from-rev)
      '@typescript-eslint/no-explicit-any': 'error',

      '@typescript-eslint/no-unused-vars': ['error', {
        argsIgnorePattern: '^_',
        varsIgnorePattern: '^_',
        caughtErrorsIgnorePattern: '^_',
      }],
      'no-unused-vars': 'off',
      'no-empty': 'off',
      'react-hooks/preserve-manual-memoization': 'off',
      // React 19 strict mode rules — too aggressive for existing codebase
      'react-hooks/set-state-in-effect': 'off',
      'react-hooks/refs': 'off',
      // HMR fast-refresh is a dev-only concern — not a production code quality issue
      'react-refresh/only-export-components': 'off',

      // M0.3-3: max-lines — TS files ≤250 lines (matching AGENTS.md §复杂度硬上限)
      'max-lines': ['error', {
        max: 250,
        skipBlankLines: true,
        skipComments: true,
      }],

      // AGENTS.md §File & Function Size: TS functions ≤50 lines (soft reference)
      'max-lines-per-function': ['warn', {
        max: 50,
        skipBlankLines: true,
        skipComments: true,
        IIFEs: true,
      }],

      // AGENTS.md §File & Function Size: keep cyclomatic complexity manageable
      'complexity': ['warn', { max: 10 }],
    },
  },
])
