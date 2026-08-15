import { FORMAT_CRON_KEY, FORMAT_INTERVAL_KEY } from '@/gen/ant/v1/i18n/strategy_schedules_keys';
import type { ScheduleRow } from '../hooks/libraryTypes';

export function formatSchedule(t: (key: string, opts?: Record<string, unknown>) => string, row: ScheduleRow) {
  const conf = row?.scheduleConfig || {};
  if (row?.scheduleType === "interval") {
    const raw = conf?.intervalMs;
    const ms =
      typeof raw === "number"
        ? raw
        : typeof raw === "bigint"
          ? Number(raw)
          : undefined;
    if (typeof ms === "number" && Number.isFinite(ms) && ms > 0) {
      const s = Math.max(1, Math.floor(ms / 1000));
      return t(FORMAT_INTERVAL_KEY, { s } as Record<string, unknown>);
    }
    return "-";
  }
  const cron = String(conf?.cronExpression || "").trim();
  return cron ? t(FORMAT_CRON_KEY, { expr: cron } as Record<string, unknown>) : "-";
}
