import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor, act, fireEvent } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import EditParamsModal from '@/pages/strategy/components/live/EditParamsModal'
import type { ScheduleRow, AccountRow } from '@/pages/strategy/hooks/libraryTypes'

class ResizeObserverMock {
  observe() {}
  unobserve() {}
  disconnect() {}
  takeRecords() { return [] }
}
globalThis.ResizeObserver = ResizeObserverMock as unknown as typeof ResizeObserver

const validateExtendedMock = vi.fn()
const templateGetMock = vi.fn()
const updateMock = vi.fn()

vi.mock('@/client/codeAssist', () => ({
  codeAssistApi: {
    validateExtended: (...args: unknown[]) => validateExtendedMock(...args),
  },
}))
vi.mock('@/client/strategy-schedules', () => ({
  strategyTemplateApi: {
    get: (...args: unknown[]) => templateGetMock(...args),
  },
  strategyScheduleV2Api: {
    update: (...args: unknown[]) => updateMock(...args),
  },
}))

const accounts: AccountRow[] = [
  { id: 'acc-1', login: '12345', brokerCompany: 'TestBroker' } as unknown as AccountRow,
]

function makeSchedule(overrides: Partial<ScheduleRow> = {}): ScheduleRow {
  return {
    id: 'sched-1',
    name: 'MACD Test',
    symbol: 'EURUSD',
    timeframe: '1h',
    accountId: 'acc-1',
    templateId: 'tpl-1',
    parameters: { TakeProfit: '50', LotSize: '0.01' },
    active: undefined,
    ...overrides,
  } as unknown as ScheduleRow
}

function renderModal(schedule: ScheduleRow | null = null, open = true) {
  return render(
    <MemoryRouter>
      <EditParamsModal
        open={open}
        schedule={schedule}
        accounts={accounts}
        onClose={() => {}}
        onUpdated={() => {}}
      />
    </MemoryRouter>
  )
}

beforeEach(() => {
  vi.clearAllMocks()
  validateExtendedMock.mockResolvedValue({
    valid: true,
    parameterEntries: [
      { name: 'TakeProfit', type: 'int', default: '50', label: '' },
      { name: 'LotSize', type: 'float', default: '0.01', label: '' },
    ],
  })
  templateGetMock.mockResolvedValue({ code: 'ctx.ParamInt("TakeProfit", 50)' })
  updateMock.mockResolvedValue({})
})

describe('E1: EditParamsModal — no infinite loop', () => {
  it('validateExtended is called exactly once (not spammed)', async () => {
    const sched = makeSchedule()
    renderModal(sched)
    // Wait for async load to settle
    await waitFor(() => {
      expect(validateExtendedMock).toHaveBeenCalledTimes(1)
    })
    // Wait a bit more to ensure no re-trigger
    await act(async () => { await new Promise(r => setTimeout(r, 200)) })
    expect(validateExtendedMock).toHaveBeenCalledTimes(1)
  })

  it('user edits are preserved after async param load completes', async () => {
    const sched = makeSchedule()
    renderModal(sched)
    // Wait for params to load
    await waitFor(() => {
      expect(validateExtendedMock).toHaveBeenCalledTimes(1)
    })
    await act(async () => { await new Promise(r => setTimeout(r, 100)) })
    // Find the TakeProfit input (strategy param, not a form field)
    const inputs = screen.getAllByRole('spinbutton') as HTMLInputElement[]
    const tpInput = inputs.find(i => i.value === '50')
    expect(tpInput).toBeTruthy()
    // Simulate user changing TakeProfit from 50 to 60
    await act(async () => {
      fireEvent.change(tpInput!, { target: { value: '60' } })
    })
    // Wait for any async effects to settle
    await act(async () => { await new Promise(r => setTimeout(r, 200)) })
    // User's edit should be preserved (not overwritten by default 50)
    const afterInputs = screen.getAllByRole('spinbutton') as HTMLInputElement[]
    const afterTp = afterInputs.find(i => i.value === '60')
    expect(afterTp).toBeTruthy()
  })
})

describe('E4: EditParamsModal — placebo risk fields removed', () => {
  it('does not render Default Volume field', () => {
    renderModal(makeSchedule())
    expect(screen.queryByLabelText('Default Volume')).toBeNull()
  })

  it('does not render Max Positions field', () => {
    renderModal(makeSchedule())
    expect(screen.queryByLabelText('Max Positions')).toBeNull()
  })

  it('does not render Stop Loss Offset field', () => {
    renderModal(makeSchedule())
    expect(screen.queryByLabelText('Stop Loss Offset')).toBeNull()
  })

  it('does not render Take Profit Offset field', () => {
    renderModal(makeSchedule())
    expect(screen.queryByLabelText('Take Profit Offset')).toBeNull()
  })

  it('does not render Max Drawdown % field', () => {
    renderModal(makeSchedule())
    expect(screen.queryByLabelText('Max Drawdown %')).toBeNull()
  })
})

describe('E5: EditParamsModal — name/symbol/timeframe/account retained', () => {
  it('renders Strategy Name field (onEdit=editCode does not cover these)', () => {
    renderModal(makeSchedule())
    expect(screen.getByLabelText('Strategy Name')).toBeTruthy()
  })

  it('renders Symbol field', () => {
    renderModal(makeSchedule())
    expect(screen.getByLabelText('Symbol')).toBeTruthy()
  })

  it('renders Timeframe field', () => {
    renderModal(makeSchedule())
    expect(screen.getByLabelText('Timeframe')).toBeTruthy()
  })
})
