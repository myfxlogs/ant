import type { ActiveStrategy } from '@/gen/ant/v1/strategy_runtime_pb';
import type { ScheduleRow } from '../../hooks/libraryTypes';

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
