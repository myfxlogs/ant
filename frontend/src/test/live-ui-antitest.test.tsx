import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import ScheduleTable from '@/pages/strategy/components/ScheduleTable'
import type { ScheduleRow, TemplateOption, AccountRow } from '@/pages/strategy/hooks/libraryTypes'

// ResizeObserver mock must be a constructor for antd Table
class ResizeObserverMock {
  observe() {}
  unobserve() {}
  disconnect() {}
  takeRecords() { return [] }
}
globalThis.ResizeObserver = ResizeObserverMock as unknown as typeof ResizeObserver

const noop = () => {}
const formatTime = (v: unknown) => (v ? String(v) : '—')

const baseProps = {
  templates: [] as TemplateOption[],
  accounts: [] as AccountRow[],
  loading: false,
  triggering: false,
  triggerContext: null,
  formatTime,
  onEdit: noop,
  onToggleActive: noop,
  onHealthCheck: noop,
  onManualTrigger: noop,
  onDelete: noop,
  onShowLogs: noop,
  highlightScheduleId: null,
}

function renderTable(schedules: ScheduleRow[]) {
  return render(
    <MemoryRouter>
      <ScheduleTable schedules={schedules} {...baseProps} />
    </MemoryRouter>
  )
}

// UI-2: last_error red display — render real ScheduleTable with a row that has lastError.
// Adversarial: delete `{row?.lastError && (...)}` block → error text not in DOM → RED.
describe('UI-2: ScheduleTable displays lastError', () => {
  it('renders lastError text when row has lastError', () => {
    const row: ScheduleRow = {
      id: 's1', templateId: 't1', accountId: 'a1', name: 'Test',
      symbol: 'EURUSD', timeframe: 'H1', scheduleType: 'interval',
      scheduleConfig: {}, parameters: {}, isActive: false,
      lastError: 'connection refused',
    }
    renderTable([row])
    expect(screen.getByText(/connection refused/)).toBeTruthy()
  })

  it('does not render error text when row has no lastError', () => {
    const row: ScheduleRow = {
      id: 's2', templateId: 't1', accountId: 'a1', name: 'Test',
      symbol: 'EURUSD', timeframe: 'H1', scheduleType: 'interval',
      scheduleConfig: {}, parameters: {}, isActive: false,
    }
    renderTable([row])
    expect(screen.queryByText(/⚠/)).toBeFalsy()
  })
})

// UI-3: disabled guard — test that log/health buttons are disabled when scheduleId is empty.
// Tests the real guard functions used by MyStrategiesTable actions column.
import { isLogButtonDisabled, isHealthButtonDisabled } from '@/pages/strategy/components/live/strategyJoin'

describe('UI-3: log/health buttons disabled when scheduleId empty', () => {
  it('disables buttons when scheduleId is empty', () => {
    expect(isLogButtonDisabled('')).toBe(true)
    expect(isHealthButtonDisabled('')).toBe(true)
  })

  it('enables buttons when scheduleId is non-empty', () => {
    expect(isLogButtonDisabled('sched-123')).toBe(false)
    expect(isHealthButtonDisabled('sched-123')).toBe(false)
  })
})

// UI-1: Enable→tab1 navigation — test the navigate decision as a pure function.
// Adversarial: delete the `if (next) return '/strategy/live?tab=strategies'` line →
// function returns null → navigate not called → RED.
import { getEnableNavigateTarget } from '@/pages/strategy/components/workspace/LiveSchedulesTab'

describe('UI-1: Enable success navigates to active tab', () => {
  it('returns active tab path when enabled', () => {
    expect(getEnableNavigateTarget(true)).toBe('/strategy/live?tab=strategies')
  })

  it('returns null when disabled (no navigation)', () => {
    expect(getEnableNavigateTarget(false)).toBe(null)
  })
})

// UI-4: Dual-stream join — verify the real joinSchedulesWithActive/findOrphanRuns functions.
// Adversarial: neutralize the join body in strategyJoin.ts (e.g. return schedules.map(s => ({...s})) without active)
// → joinSchedulesWithActive returns no active → test expects active to be defined → RED.
import { joinSchedulesWithActive, findOrphanRuns } from '@/pages/strategy/components/live/strategyJoin'
import type { ActiveStrategy } from '@/gen/ant/v1/strategy_runtime_pb'

describe('UI-4: Dual-stream join shows "-" for non-running schedules', () => {
  const schedules: ScheduleRow[] = [
    { id: 's1', templateId: 't1', accountId: 'a1', name: 'Test',
      symbol: 'EURUSD', timeframe: 'H1', scheduleType: 'interval',
      scheduleConfig: {}, parameters: {}, isActive: true },
  ]

  it('joinedRow has no active data when scheduleId not in active stream', () => {
    const emptyActive: ActiveStrategy[] = []
    const joined = joinSchedulesWithActive(schedules, emptyActive)
    expect(joined).toHaveLength(1)
    expect(joined[0].active).toBeUndefined()
  })

  it('joinedRow has active data when scheduleId matches', () => {
    const mockActive = { scheduleId: 's1', pnl: '100.50', runId: 'r1' } as unknown as ActiveStrategy
    const joined = joinSchedulesWithActive(schedules, [mockActive])
    expect(joined[0].active).toBeDefined()
    expect(joined[0].active?.pnl).toBe('100.50')
  })

  it('findOrphanRuns returns active strategies with no matching schedule', () => {
    const orphan = { scheduleId: 'orphan-1', runId: 'r2', pnl: '0' } as unknown as ActiveStrategy
    const matched = { scheduleId: 's1', runId: 'r1', pnl: '50' } as unknown as ActiveStrategy
    const result = findOrphanRuns([orphan, matched], schedules)
    expect(result).toHaveLength(1)
    expect(result[0].runId).toBe('r2')
  })

  it('findOrphanRuns returns active strategies with empty scheduleId', () => {
    const noSchedule = { scheduleId: '', runId: 'r3', pnl: '0' } as unknown as ActiveStrategy
    const result = findOrphanRuns([noSchedule], schedules)
    expect(result).toHaveLength(1)
    expect(result[0].runId).toBe('r3')
  })
})
