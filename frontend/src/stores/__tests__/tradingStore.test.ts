import { describe, it, expect, beforeEach } from 'vitest'
import { useTradingStore } from '@/stores/tradingStore'
import type { Position, TradeLog } from '@/types/trading'

function makePosition(overrides: Partial<Position> = {}): Position {
  return {
    ticket: 1001,
    symbol: 'EURUSD',
    type: 'buy',
    volume: 0.1,
    openPrice: 1.1,
    currentPrice: 1.105,
    sl: 0,
    tp: 0,
    profit: 50,
    swap: 0,
    commission: 0,
    openTime: '2024-01-01T00:00:00Z',
    comment: '',
    ...overrides,
  }
}

describe('tradingStore', () => {
  beforeEach(() => {
    useTradingStore.setState({
      positions: [],
      positionsMap: new Map(),
      tradeLogs: [],
      accountInfo: { balance: 0, credit: 0, profit: 0, profitPercent: 0, equity: 0, margin: 0, freeMargin: 0, marginLevel: 0 },
      accountInfoMap: new Map(),
      accountReceivedData: new Set(),
      lastStreamProfitAtByAccount: new Map(),
      currentAccountId: null,
      loading: false,
    })
  })

  it('starts with empty state', () => {
    const s = useTradingStore.getState()
    expect(s.positions).toHaveLength(0)
    expect(s.tradeLogs).toHaveLength(0)
    expect(s.currentAccountId).toBeNull()
    expect(s.loading).toBe(false)
  })

  it('setCurrentAccountId sets current account', () => {
    useTradingStore.getState().setCurrentAccountId('acc-1')
    expect(useTradingStore.getState().currentAccountId).toBe('acc-1')
  })

  it('setPositions stores positions for account', () => {
    useTradingStore.getState().setCurrentAccountId('acc-1')
    useTradingStore.getState().setPositions('acc-1', [
      makePosition({ ticket: 1 }) as unknown as Record<string, unknown>,
    ])
    expect(useTradingStore.getState().positions).toHaveLength(1)
    expect(useTradingStore.getState().positions[0].ticket).toBe(1)
  })

  it('addPosition adds and deduplicates by ticket', () => {
    useTradingStore.getState().setCurrentAccountId('acc-1')
    const pos = makePosition({ ticket: 1001 })
    useTradingStore.getState().addPosition('acc-1', pos)
    expect(useTradingStore.getState().positions).toHaveLength(1)
    // Adding same ticket again should not duplicate
    useTradingStore.getState().addPosition('acc-1', pos)
    expect(useTradingStore.getState().positions).toHaveLength(1)
  })

  it('updatePosition modifies existing position', () => {
    useTradingStore.getState().setCurrentAccountId('acc-1')
    useTradingStore.getState().addPosition('acc-1', makePosition({ ticket: 1001, profit: 50 }))
    useTradingStore.getState().updatePosition('acc-1', 1001, { profit: 100 })
    expect(useTradingStore.getState().positions[0].profit).toBe(100)
  })

  it('removePosition removes by ticket', () => {
    useTradingStore.getState().setCurrentAccountId('acc-1')
    useTradingStore.getState().addPosition('acc-1', makePosition({ ticket: 1001 }))
    useTradingStore.getState().removePosition('acc-1', 1001)
    expect(useTradingStore.getState().positions).toHaveLength(0)
  })

  it('removePosition is no-op for non-existent ticket', () => {
    useTradingStore.getState().setCurrentAccountId('acc-1')
    useTradingStore.getState().addPosition('acc-1', makePosition({ ticket: 1001 }))
    useTradingStore.getState().removePosition('acc-1', 9999)
    expect(useTradingStore.getState().positions).toHaveLength(1)
  })

  it('setTradeLogs replaces logs', () => {
    const logs: TradeLog[] = [{ ticket: 1, symbol: 'EURUSD', type: 'buy', volume: 0.1, price: 1.1, time: 0, profit: 0 } as unknown as TradeLog]
    useTradingStore.getState().setTradeLogs(logs)
    expect(useTradingStore.getState().tradeLogs).toHaveLength(1)
  })

  it('addTradeLog prepends to logs', () => {
    useTradingStore.getState().setTradeLogs([
      { ticket: 1, symbol: 'EURUSD', type: 'buy', volume: 0.1, price: 1.1, time: 0, profit: 0 } as unknown as TradeLog,
    ])
    useTradingStore.getState().addTradeLog({ ticket: 2, symbol: 'GBPUSD', type: 'sell', volume: 0.2, price: 1.3, time: 0, profit: 0 } as unknown as TradeLog)
    const logs = useTradingStore.getState().tradeLogs
    expect(logs).toHaveLength(2)
    expect(logs[0].ticket).toBe(2)
  })

  it('setAccountInfo merges partial info', () => {
    useTradingStore.getState().setCurrentAccountId('acc-1')
    useTradingStore.getState().setAccountInfo({ balance: 1000 })
    expect(useTradingStore.getState().accountInfo.balance).toBe(1000)
    useTradingStore.getState().setAccountInfo({ profit: 50 })
    expect(useTradingStore.getState().accountInfo.profit).toBe(50)
    expect(useTradingStore.getState().accountInfo.balance).toBe(1000)
  })

  it('setAccountInfoById stores per-account and marks received', () => {
    useTradingStore.getState().setAccountInfoById('acc-1', { balance: 500 })
    expect(useTradingStore.getState().getAccountInfoById('acc-1')?.balance).toBe(500)
    expect(useTradingStore.getState().hasReceivedData('acc-1')).toBe(true)
    expect(useTradingStore.getState().hasReceivedData('acc-2')).toBe(false)
  })

  it('setCurrentAccountId loads cached positions and accountInfo', () => {
    useTradingStore.getState().setPositions('acc-1', [makePosition({ ticket: 1 }) as unknown as Record<string, unknown>])
    useTradingStore.getState().setAccountInfoById('acc-1', { balance: 2000 })
    useTradingStore.getState().setCurrentAccountId('acc-1')
    expect(useTradingStore.getState().positions).toHaveLength(1)
    expect(useTradingStore.getState().accountInfo.balance).toBe(2000)
  })

  it('setCurrentAccountId(null) resets to defaults', () => {
    useTradingStore.getState().setCurrentAccountId('acc-1')
    useTradingStore.getState().setAccountInfo({ balance: 500 })
    useTradingStore.getState().setCurrentAccountId(null)
    expect(useTradingStore.getState().currentAccountId).toBeNull()
    expect(useTradingStore.getState().accountInfo.balance).toBe(0)
  })

  it('setLoading toggles loading', () => {
    useTradingStore.getState().setLoading(true)
    expect(useTradingStore.getState().loading).toBe(true)
  })
})
