import { describe, it, expect } from 'vitest'
import { paramLabel } from './paramLabel'

describe('paramLabel', () => {
  const i18nData = {
    locales: {
      'en': { labels: { 'MAPeriod': 'MA Period', 'RSIPeriod': 'RSI Period' } },
      'zh-cn': { labels: { 'MAPeriod': '均线周期' } },
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

  it('returns raw name when locales is empty', () => {
    expect(paramLabel('MAPeriod', 'en', { locales: {} })).toBe('MAPeriod')
  })

  it('returns raw name when locale has no labels for the param', () => {
    expect(paramLabel('NoLabel', 'en', { locales: { 'en': { labels: {} } } })).toBe('NoLabel')
  })

  it('handles locale with region suffix fallback', () => {
    expect(paramLabel('MAPeriod', 'en-US', i18nData)).toBe('MA Period')
  })
})
