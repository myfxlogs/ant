import { describe, it, expect } from 'vitest'
import { isTradingAccountEnabled } from './accountStatus'
import type { Account } from '@/types/account'

function makeAccount(overrides: Partial<Account> = {}): Account {
  return {
    id: 'test-1',
    mtType: 'MT4',
    status: 'connected',
    ...overrides,
  } as Account
}

describe('isTradingAccountEnabled', () => {
  it('returns false for null', () => {
    expect(isTradingAccountEnabled(null)).toBe(false)
  })

  it('returns false for undefined', () => {
    expect(isTradingAccountEnabled(undefined)).toBe(false)
  })

  it('returns true for connected account', () => {
    expect(isTradingAccountEnabled(makeAccount({ status: 'connected' }))).toBe(true)
  })

  it('returns false for disconnected account', () => {
    expect(isTradingAccountEnabled(makeAccount({ status: 'disconnected' }))).toBe(false)
  })

  it('returns false for frozen account', () => {
    expect(isTradingAccountEnabled(makeAccount({ status: 'frozen' }))).toBe(false)
  })

  it('returns true for unknown status', () => {
    expect(isTradingAccountEnabled(makeAccount({ status: 'connecting' }))).toBe(true)
  })

  it('returns true for empty status', () => {
    expect(isTradingAccountEnabled(makeAccount({ status: '' }))).toBe(true)
  })

  it('uses accountStatus as fallback', () => {
    expect(isTradingAccountEnabled(makeAccount({ status: '', accountStatus: 'frozen' } as Partial<Account>))).toBe(false)
  })

  it('returns false when isDisabled is true (legacy)', () => {
    expect(isTradingAccountEnabled(makeAccount({ status: 'connected', isDisabled: true } as Partial<Account>))).toBe(false)
  })

  it('is case-insensitive for status', () => {
    expect(isTradingAccountEnabled(makeAccount({ status: 'FROZEN' }))).toBe(false)
    expect(isTradingAccountEnabled(makeAccount({ status: 'Disconnected' }))).toBe(false)
  })
})
