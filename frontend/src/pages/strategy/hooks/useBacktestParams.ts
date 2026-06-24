// Re-exports from the unified useBacktestRunner hook.
// This file is retained for backward compatibility during migration.
export {
  useBacktestRunner as useBacktestParams,
  PRESETS, DATE_PRESETS,
} from '@/components/backtest/useBacktestRunner';

export type {
  BacktestMetrics, BacktestStatus, ChartTrade,
  StrategyDirective, PresetKey, ExtractedParam,
  SweepDimension, TuneMethod, BacktestSubTab,
} from '@/components/backtest/useBacktestRunner';

export { OPTIMIZER_INFO } from './useTuning';

