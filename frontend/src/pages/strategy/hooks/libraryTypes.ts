import type { StrategyTemplate } from '@/client/strategy';

// ── Template helpers ──
export function isSystemTemplate(tpl: StrategyTemplate): boolean {
  const id = String(tpl.id || '');
  const tags = tpl.tags || [];
  return Boolean(tpl.isSystem) || tags.includes('preset') || id.startsWith('default-');
}

export function isPublicTemplate(tpl: StrategyTemplate): boolean {
  return Boolean(tpl.isPublic);
}

// Minimal template reference used by schedule forms and tables
export interface TemplateOption {
  id: string;
  name: string;
  isPublic?: boolean;
}

// ── Schedule entity (the shape returned by API) ──
export interface ScheduleRow {
  id: string;
  templateId: string;
  accountId: string;
  name: string;
  symbol: string;
  timeframe: string;
  scheduleType: string;
  scheduleConfig: Record<string, unknown>;
  parameters: Record<string, string>;
  isActive: boolean;
  lastRunAt?: string;
  nextRunAt?: string;
  enableCount?: number;
  createdAt?: string;
}

// ── Backtest run entity ──
export interface BacktestRunRow {
  id: string;
  templateId: string;
  templateDraftId?: string;
  symbol: string;
  timeframe: string;
  status: number;
  createdAt: string;
  metrics?: Record<string, unknown>;
  equityCurve?: unknown[];
  title?: string;
  error?: string;
}

// ── Account entity ──
export interface AccountRow {
  id: string;
  login?: string;
  brokerCompany?: string;
  brokerServer?: string;
  mtType?: string;
  leverage?: number;
  isDisabled?: boolean;
}

// ── Symbol option ──
export interface SymbolOption {
  value: string;
  label: string;
}

// ── Health summary ──
export interface ScheduleHealthSummary {
  totalRuns: number;
  successRuns: number;
  failedRuns: number;
  successRate: number;
  lastRunAt?: string;
  latestError: string;
  latestOrderTicket: string | number;
  latestOrderProfit: number | null;
  gradeLevel: string;
  gradeColor: string;
  gradeNoteCode: string;
  greenSuccessRate: number;
  greenMaxFailedRuns: number;
  yellowSuccessRate: number;
  minSampleSize: number;
  runLogs: unknown[];
  orders: unknown[];
}

// ── Trigger result ──
export interface TriggerResult {
  logs: string[];
  signal: {
    signalId?: string;
    direction?: string;
    confidence?: number;
    type?: string;
    signalType?: string;
    volume?: number;
    price?: number;
    stopLoss?: number;
    takeProfit?: number;
    comment?: string;
    symbol?: string;
  } | null;
  meta: Record<string, unknown>;
}

export interface TriggerContext {
  schedule: ScheduleRow;
  accountId: string;
}
