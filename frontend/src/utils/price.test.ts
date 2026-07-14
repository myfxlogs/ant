import { describe, it, expect } from 'vitest'
import { getSymbolDecimals, formatPrice } from './price'

describe('getSymbolDecimals', () => {
  it('returns 5 for EURUSD', () => {
    expect(getSymbolDecimals('EURUSD')).toBe(5)
  })

  it('returns 2 for XAUUSD', () => {
    expect(getSymbolDecimals('XAUUSD')).toBe(2)
  })

  it('returns 3 for USDJPY', () => {
    expect(getSymbolDecimals('USDJPY')).toBe(3)
  })

  it('returns 2 for BTCUSD', () => {
    expect(getSymbolDecimals('BTCUSD')).toBe(2)
  })

  it('handles lowercase symbols', () => {
    expect(getSymbolDecimals('eurusd')).toBe(5)
  })

  it('handles m-suffix symbols', () => {
    expect(getSymbolDecimals('EURUSDm')).toBe(5)
  })

  it('returns default 5 for unknown symbols', () => {
    expect(getSymbolDecimals('UNKNOWN')).toBe(5)
  })

  it('returns 3 for XAGUSD', () => {
    expect(getSymbolDecimals('XAGUSD')).toBe(3)
  })
})

describe('formatPrice', () => {
  it('formats with symbol decimals', () => {
    expect(formatPrice(1.123456, 'EURUSD')).toBe('1.12346')
  })

  it('formats XAUUSD with 2 decimals', () => {
    expect(formatPrice(2050.5, 'XAUUSD')).toBe('2050.5')
  })

  it('formats USDJPY with 3 decimals', () => {
    expect(formatPrice(150.123, 'USDJPY')).toBe('150.123')
  })

  it('returns dash for undefined', () => {
    expect(formatPrice(undefined)).toBe('-')
  })

  it('returns dash for null', () => {
    expect(formatPrice(null)).toBe('-')
  })

  it('returns dash for NaN', () => {
    expect(formatPrice(NaN)).toBe('-')
  })

  it('uses default decimals without symbol', () => {
    expect(formatPrice(1.123456)).toBe('1.12346')
  })
})
