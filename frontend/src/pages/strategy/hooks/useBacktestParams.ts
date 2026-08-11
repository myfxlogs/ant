// Re-exports from the unified useBacktestRunner hook.
// This file is retained for backward compatibility during migration.
export type {
  BacktestMetrics,
  SweepDimension, TuneMethod,
} from '@/components/backtest/useBacktestRunner';

export { OPTIMIZER_INFO } from './useTuning';

