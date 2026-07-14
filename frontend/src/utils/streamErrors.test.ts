import { describe, it, expect } from 'vitest'
import { isLikelyStreamTransportFailure, isStreamAuthFailure, isStreamServiceProcedure } from './streamErrors'

describe('isLikelyStreamTransportFailure', () => {
  it('detects network error', () => {
    expect(isLikelyStreamTransportFailure(new Error('network error'))).toBe(true)
  })

  it('detects HTTP2 protocol error', () => {
    expect(isLikelyStreamTransportFailure(new Error('ERR_HTTP2_PROTOCOL_ERROR'))).toBe(true)
  })

  it('detects failed to fetch', () => {
    expect(isLikelyStreamTransportFailure(new Error('failed to fetch'))).toBe(true)
  })

  it('detects 524 gateway timeout', () => {
    expect(isLikelyStreamTransportFailure(new Error('status code 524'))).toBe(true)
  })

  it('detects missing request message (stream abort)', () => {
    expect(isLikelyStreamTransportFailure('missing request message')).toBe(true)
  })

  it('detects gateway timeout text', () => {
    expect(isLikelyStreamTransportFailure(new Error('gateway time-out'))).toBe(true)
  })

  it('detects deadline exceeded', () => {
    expect(isLikelyStreamTransportFailure(new Error('deadline exceeded'))).toBe(true)
  })

  it('returns false for unrelated errors', () => {
    expect(isLikelyStreamTransportFailure(new Error('something else'))).toBe(false)
  })

  it('returns false for null', () => {
    expect(isLikelyStreamTransportFailure(null)).toBe(false)
  })

  it('returns false for undefined', () => {
    expect(isLikelyStreamTransportFailure(undefined)).toBe(false)
  })

  it('detects error with cause', () => {
    const err = new Error('outer')
    err.cause = new Error('network error')
    expect(isLikelyStreamTransportFailure(err)).toBe(true)
  })
})

describe('isStreamAuthFailure', () => {
  it('detects missing authorization header', () => {
    expect(isStreamAuthFailure(new Error('missing authorization header'))).toBe(true)
  })

  it('detects unauthenticated', () => {
    expect(isStreamAuthFailure(new Error('unauthenticated'))).toBe(true)
  })

  it('detects token expired', () => {
    expect(isStreamAuthFailure(new Error('token expired'))).toBe(true)
  })

  it('detects invalid token', () => {
    expect(isStreamAuthFailure(new Error('invalid token'))).toBe(true)
  })

  it('returns false for network errors', () => {
    expect(isStreamAuthFailure(new Error('network error'))).toBe(false)
  })

  it('returns false for missing request message (not auth)', () => {
    expect(isStreamAuthFailure('missing request message')).toBe(false)
  })

  it('returns false for null', () => {
    expect(isStreamAuthFailure(null)).toBe(false)
  })

  it('detects error with code in message', () => {
    expect(isStreamAuthFailure({ code: 'unauthenticated', message: 'unauthenticated' })).toBe(true)
  })
})

describe('isStreamServiceProcedure', () => {
  it('detects streamservice in procedure name', () => {
    expect(isStreamServiceProcedure('/ant.v1.streamservice/subscribeglobal')).toBe(true)
  })

  it('returns false for non-stream procedures', () => {
    expect(isStreamServiceProcedure('/ant.v1.authservice/login')).toBe(false)
  })

  it('returns false for empty string', () => {
    expect(isStreamServiceProcedure('')).toBe(false)
  })
})
