/**
 * Timeframe constants — single source of truth for all period values.
 *
 * Internal format matches klinecharts, TradingView, ClickHouse storage:
 *   1m, 5m, 15m, 30m, 1h, 4h, 1d, 1w
 *
 * Usage:
 *   import { TIMEFRAMES, type Timeframe } from '@/constants/timeframes';
 *   <Select options={TIMEFRAMES.map(tf => ({ value: tf, label: tf }))} />
 *   const [tf, setTf] = useState<Timeframe>('1h');
 */

export const TIMEFRAMES = ['1m', '5m', '15m', '30m', '1h', '4h', '1d', '1w'] as const;

export type Timeframe = (typeof TIMEFRAMES)[number];

/** Default timeframe for forms and initial state. */
export const DEFAULT_TIMEFRAME: Timeframe = '1h';
