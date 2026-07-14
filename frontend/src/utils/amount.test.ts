import { describe, it, expect } from 'vitest'
import { formatAmount } from './amount'

describe('formatAmount', () => {
  it('formats integer string with 2 decimals', () => {
    expect(formatAmount('1234')).toBe('1234.00')
  })

  it('formats decimal string with 2 decimals', () => {
    expect(formatAmount('1234.5')).toBe('1234.50')
  })

  it('formats exact 2-decimal string', () => {
    expect(formatAmount('1234.56')).toBe('1234.56')
  })

  it('preserves decimals beyond 2nd place', () => {
    expect(formatAmount('1234.567')).toBe('1234.567')
  })

  it('strips trailing zeros beyond 2nd decimal', () => {
    expect(formatAmount('1234.560')).toBe('1234.56')
  })

  it('handles number input', () => {
    expect(formatAmount(1234.5)).toBe('1234.50')
  })

  it('handles null', () => {
    expect(formatAmount(null)).toBe('0.00')
  })

  it('handles undefined', () => {
    expect(formatAmount(undefined)).toBe('0.00')
  })

  it('handles NaN string', () => {
    expect(formatAmount('not-a-number')).toBe('0.00')
  })

  it('handles zero', () => {
    expect(formatAmount(0)).toBe('0.00')
  })

  it('handles negative numbers', () => {
    expect(formatAmount(-1234.5)).toBe('-1234.50')
  })

  it('handles large numbers', () => {
    expect(formatAmount('999999.99')).toBe('999999.99')
  })
})
