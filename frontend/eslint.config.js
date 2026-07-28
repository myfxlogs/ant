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

      // M0.3-3: max-lines — TS files ≤250 lines (matching AGENT.md §复杂度硬上限)
      'max-lines': ['error', {
        max: 250,
        skipBlankLines: true,
        skipComments: true,
      }],
    },
  },
  {
    // Context providers and shared helper files export both components and constants/types.
    // Splitting them is a larger refactor — allow constant exports for now.
    files: [
      'src/**/*Context.tsx',
      'src/**/*Helpers.tsx',
      'src/**/chatUtils.tsx',
      'src/**/ChartToolbar.tsx',
      'src/**/CompareModal.tsx',
      'src/**/VersionHistoryTab.tsx',
      'src/**/LiveStrategyPageSignalDrawer.tsx',
      'src/**/EditScheduleBasicFields.tsx',
    ],
    rules: {
      'react-refresh/only-export-components': ['warn', { allowConstantExport: true }],
    },
  },
  {
    // Files exceeding 250-line limit — splitting is a deferred refactor
    files: [
      'src/client/analyticsMappers.ts',
      'src/pages/admin/SweepManagement.tsx',
      'src/pages/marketplace/components/AutoGeneratePanel.tsx',
      'src/pages/marketplace/components/OptimizationTab.tsx',
      'src/pages/marketplace/components/StrategyDetailModal.tsx',
      'src/pages/share/SharePerformancePage.tsx',
      'src/pages/strategy/components/workspace/BacktestHistoryDrawer.tsx',
      'src/pages/strategy/components/workspace/VersionHistoryDrawer.tsx',
      'src/pages/strategy/components/workspace/WorkspaceCodePanel.tsx',
      'src/pages/subscription/SubscriptionPage.tsx',
    ],
    rules: {
      'max-lines': ['error', { max: 350, skipBlankLines: true, skipComments: true }],
    },
  },
])
