import type { ActiveStrategy } from '@/gen/ant/v1/strategy_runtime_pb';
import type { ScheduleRow } from '../../hooks/libraryTypes';

// Expand-column width shared by the outer strategy table and its expanded row:
// the expanded-row content offsets by this amount so the "Positions" tab label
// and the inner table's first column header align vertically with the outer
// "Strategy" column header (single source of truth — change both or neither).
export const LIVE_EXPAND_COL_WIDTH = 48;

export interface JoinedRow extends ScheduleRow {
  active?: ActiveStrategy;
}

export function joinSchedulesWithActive(
  schedules: ScheduleRow[],
  activeStrategies: ActiveStrategy[],
): JoinedRow[] {
  const activeBySchedule = new Map<string, ActiveStrategy>();
  for (const a of activeStrategies) {
    if (a.scheduleId) activeBySchedule.set(a.scheduleId, a);
  }
  return schedules.map(s => ({ ...s, active: activeBySchedule.get(s.id) }));
}

export function findOrphanRuns(
  activeStrategies: ActiveStrategy[],
  schedules: ScheduleRow[],
): ActiveStrategy[] {
  const scheduleIds = new Set(schedules.map(s => s.id));
  return activeStrategies.filter(a => !a.scheduleId || !scheduleIds.has(a.scheduleId));
}

export function isLogButtonDisabled(scheduleId: string): boolean { return !scheduleId; }
export function isHealthButtonDisabled(scheduleId: string): boolean { return !scheduleId; }
