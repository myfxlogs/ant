import { describe, it, expect, vi } from 'vitest'
import { render, fireEvent, screen } from '@testing-library/react'

// BacktestHistoryPanel：主区回测历史列表，点击一条触发 onOpen 加载。

vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (k: string, o?: unknown) => {
      if (typeof o === 'string') return o
      if (o && typeof o === 'object' && 'defaultValue' in (o as object)) return String((o as { defaultValue: unknown }).defaultValue)
      return k
    },
  }),
}))

import BacktestHistoryPanel from '@/pages/strategy/components/workspace/BacktestHistoryPanel'

const runs = [
  { id: 'r1', name: '均线策略', startedAt: '2026-09-08', totalReturn: 12.5, totalTrades: 30 },
  { id: 'r2', name: 'MACD', startedAt: '2026-09-07', totalReturn: -3.2, totalTrades: 10 },
]

describe('BacktestHistoryPanel', () => {
  it('renders runs and routes click to onOpen', () => {
    const onOpen = vi.fn()
    render(<BacktestHistoryPanel runs={runs} loading={false} onOpen={onOpen} />)

    expect(screen.getByText('均线策略')).toBeTruthy()
    expect(screen.getByText('MACD')).toBeTruthy()
    fireEvent.click(screen.getByText('MACD'))
    expect(onOpen).toHaveBeenCalledWith('r2')
  })

  it('shows empty state when no runs', () => {
    render(<BacktestHistoryPanel runs={[]} loading={false} onOpen={vi.fn()} />)
    expect(screen.getByText(/暂无回测记录/)).toBeTruthy()
  })

  it('renders protobuf Timestamp startedAt without crashing (React #31 regression)', () => {
    // 生产数据形状：runs.startedAt 是 protobuf Timestamp 对象（非字符串）。
    const protoRuns = [
      { id: 'r1', name: '回测 A', startedAt: { seconds: 1788840000, nanos: 500000000 }, totalReturn: 3.1, totalTrades: 8 },
      { id: 'r2', name: '回测 B', startedAt: '', totalReturn: -1, totalTrades: 2 },
    ]
    render(<BacktestHistoryPanel runs={protoRuns as never} loading={false} onOpen={vi.fn()} />)
    expect(screen.getByText('回测 A')).toBeTruthy()
    const stamp = screen.getByText(/2026/)
    expect(stamp.textContent).toBeTruthy()
    expect(screen.getByText('回测 B')).toBeTruthy()
  })
})
