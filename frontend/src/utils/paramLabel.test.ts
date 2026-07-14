import { describe, it, expect } from 'vitest'
import { paramLabel } from './paramLabel'

describe('paramLabel', () => {
  const i18nData = {
    params: {
      'MAPeriod': {
        label: {
          'en': 'MA Period',
          'zh-cn': '均线周期',
        },
      },
      'RSIPeriod': {
        label: {
          'en': 'RSI Period',
        },
      },
    },
  }

  it('returns label for exact locale match', () => {
    expect(paramLabel('MAPeriod', 'zh-cn', i18nData)).toBe('均线周期')
  })

  it('returns label for en locale', () => {
    expect(paramLabel('MAPeriod', 'en', i18nData)).toBe('MA Period')
  })

  it('falls back to en when zh-tw not found (only zh-cn exists)', () => {
    expect(paramLabel('MAPeriod', 'zh-tw', i18nData)).toBe('MA Period')
  })

  it('falls back to en when locale not found', () => {
    expect(paramLabel('MAPeriod', 'ja', i18nData)).toBe('MA Period')
  })

  it('falls back to en when only en is available', () => {
    expect(paramLabel('RSIPeriod', 'zh-cn', i18nData)).toBe('RSI Period')
  })

  it('returns raw name when no i18n data', () => {
    expect(paramLabel('UnknownParam', 'en', i18nData)).toBe('UnknownParam')
  })

  it('returns raw name when i18nData is null', () => {
    expect(paramLabel('MAPeriod', 'en', null)).toBe('MAPeriod')
  })

  it('returns raw name when i18nData is undefined', () => {
    expect(paramLabel('MAPeriod', 'en', undefined)).toBe('MAPeriod')
  })

  it('returns raw name when params is empty', () => {
    expect(paramLabel('MAPeriod', 'en', { params: {} })).toBe('MAPeriod')
  })

  it('returns raw name when param has no label', () => {
    expect(paramLabel('NoLabel', 'en', { params: { NoLabel: {} } })).toBe('NoLabel')
  })

  it('returns raw name when label map is empty', () => {
    expect(paramLabel('Empty', 'en', { params: { Empty: { label: {} } } })).toBe('Empty')
  })

  it('handles locale with region suffix fallback', () => {
    expect(paramLabel('MAPeriod', 'en-US', i18nData)).toBe('MA Period')
  })
})
