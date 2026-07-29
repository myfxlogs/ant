/** Errors typical of proxies / HTTP2 + long-lived Connect streams (not actionable in the UI). */
const TRANSPORT_FAILURE_KEYWORDS = [
  'network error', 'err_http2', 'http2_protocol', 'protocol_error',
  'failed to fetch', 'load failed', 'the network connection was lost',
  ' 524', '524 ', 'status code 524', 'http 524',
  'timeout occurred', 'gateway time-out', 'gateway timeout',
  'deadline exceeded', 'err_unavailable', 'unavailable',
  // Request body not fully sent — typical when a long-lived stream is
  // aborted mid-flight (page refresh / component unmount / navigation).
  // This is a transport interruption, NOT an authentication failure.
  'missing request message',
];

export function isLikelyStreamTransportFailure(error: unknown): boolean {
  const e = error as { message?: unknown; cause?: unknown } | null | undefined;
  const cause = e?.cause as { message?: unknown } | undefined;
  const parts = [
    String(e?.message ?? ''),
    String(error ?? ''),
    String(cause?.message ?? ''),
    String(e?.cause ?? ''),
  ].join(' ').toLowerCase();
  return TRANSPORT_FAILURE_KEYWORDS.some(kw => parts.includes(kw));
}

const AUTH_FAILURE_KEYWORDS = [
  'missing authorization header', 'unauthenticated', 'token expired', 'invalid token',
];

/**
 * Auth-related errors on StreamService procedures — token is expired or
 * missing and the reactive refresh has already failed. The user needs to
 * re-login; the stream cannot recover on its own.
 */
export function isStreamAuthFailure(error: unknown): boolean {
  const e = error as { code?: unknown; message?: unknown; rawMessage?: unknown } | null | undefined;
  const parts = [
    String(e?.message ?? ''),
    String(e?.rawMessage ?? ''),
    String(error ?? ''),
  ].join(' ').toLowerCase();
  return AUTH_FAILURE_KEYWORDS.some(kw => parts.includes(kw));
}

export function isStreamServiceProcedure(procLower: string): boolean {
  return procLower.includes('streamservice');
}
