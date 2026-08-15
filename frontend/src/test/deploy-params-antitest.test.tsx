import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor, act, fireEvent } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import DeployScheduleModal from '@/pages/strategy/components/DeployScheduleModal'

class ResizeObserverMock {
  observe() {}
  unobserve() {}
  disconnect() {}
  takeRecords() { return [] }
}
globalThis.ResizeObserver = ResizeObserverMock as unknown as typeof ResizeObserver

const validateExtendedMock = vi.fn()
const templateGetMock = vi.fn()
const createScheduleMock = vi.fn()
const fetchAccountsMock = vi.fn()

vi.mock('@/client/codeAssist', () => ({
  codeAssistApi: {
    validateExtended: (...args: unknown[]) => validateExtendedMock(...args),
  },
}))
vi.mock('@/client/strategy-schedules', () => ({
  strategyScheduleApi: {
    createSchedule: (...args: unknown[]) => createScheduleMock(...args),
  },
  strategyTemplateApi: {
    get: (...args: unknown[]) => templateGetMock(...args),
  },
}))
vi.mock('@/hooks/useAccount', () => ({
  useAccount: () => ({
    accounts: [{ id: 'acc-1', brokerServer: 'Exness', login: '12345', isDisabled: false }],
    fetchAccounts: (...args: unknown[]) => fetchAccountsMock(...args),
  }),
}))

beforeEach(() => {
  vi.clearAllMocks()
  fetchAccountsMock.mockResolvedValue(undefined)
  templateGetMock.mockResolvedValue({ code: 'ctx.ParamInt("TakeProfit", 50)\nctx.ParamDecimal("LotSize", 0.01)' })
  validateExtendedMock.mockResolvedValue({
    valid: true,
    parameterEntries: [
      { name: 'TakeProfit', type: 'int', default: '50', label: '' },
      { name: 'LotSize', type: 'float', default: '0.01', label: '' },
    ],
  })
  createScheduleMock.mockResolvedValue({ id: 'new-sched-1' })
})

function renderModal(open = true) {
  return render(
    <MemoryRouter>
      <DeployScheduleModal
        open={open}
        templateId="tpl-1"
        templateName="MACD Sample"
        onClose={() => {}}
        onCreated={() => {}}
      />
    </MemoryRouter>
  )
}

describe('D1: DeployScheduleModal — parameter section', () => {
  it('renders parameter inputs with default values pre-filled', async () => {
    renderModal()
    await waitFor(() => {
      expect(validateExtendedMock).toHaveBeenCalledTimes(1)
    })
    await act(async () => { await new Promise(r => setTimeout(r, 100)) })
    // TakeProfit default=50, LotSize default=0.01 should be visible
    const inputs = screen.getAllByRole('spinbutton') as HTMLInputElement[]
    const tp = inputs.find(i => i.value === '50')
    const lot = inputs.find(i => i.value === '0.01')
    expect(tp).toBeTruthy()
    expect(lot).toBeTruthy()
  })

  it('user-modified parameter value persists (not overwritten by defaults)', async () => {
    renderModal()
    await waitFor(() => {
      expect(validateExtendedMock).toHaveBeenCalledTimes(1)
    })
    await act(async () => { await new Promise(r => setTimeout(r, 100)) })

    // Change LotSize from 0.01 to 0.02
    const inputs = screen.getAllByRole('spinbutton') as HTMLInputElement[]
    const lotInput = inputs.find(i => i.value === '0.01')
    expect(lotInput).toBeTruthy()
    await act(async () => {
      fireEvent.change(lotInput!, { target: { value: '0.02' } })
    })

    // Wait for any async effects to settle
    await act(async () => { await new Promise(r => setTimeout(r, 200)) })

    // User's edit should be preserved (not overwritten by default 0.01)
    const afterInputs = screen.getAllByRole('spinbutton') as HTMLInputElement[]
    const afterLot = afterInputs.find(i => i.value === '0.02')
    expect(afterLot).toBeTruthy()
  })

  it('shows "no parameters" text when template has no params', async () => {
    validateExtendedMock.mockResolvedValue({
      valid: true,
      parameterEntries: [],
    })
    renderModal()
    await waitFor(() => {
      expect(validateExtendedMock).toHaveBeenCalledTimes(1)
    })
    await act(async () => { await new Promise(r => setTimeout(r, 100)) })
    expect(screen.getByText(/no input parameters/i)).toBeTruthy()
  })
})
