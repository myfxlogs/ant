/** Errors typical of proxies / HTTP2 + long-lived Connect streams (not actionable in the UI). */
export function isLikelyStreamTransportFailure(error: unknown): boolean {
  const e = error as { message?: unknown; cause?: unknown } | null | undefined;
  const cause = e?.cause as { message?: unknown } | undefined;
  const parts = [
    String(e?.message ?? ''),
    String(error ?? ''),
    String(cause?.message ?? ''),
    String(e?.cause ?? ''),
  ]
    .join(' ')
    .toLowerCase();
  return (
    parts.includes('network error') ||
    parts.includes('err_http2') ||
    parts.includes('http2_protocol') ||
    parts.includes('protocol_error') ||
    parts.includes('failed to fetch') ||
    parts.includes('load failed') ||
    parts.includes('the network connection was lost') ||
    // Cloudflare / edge: long POST stream idle or origin slow → 524
    parts.includes(' 524') ||
    parts.includes('524 ') ||
    parts.includes('status code 524') ||
    parts.includes('http 524') ||
    parts.includes('timeout occurred') ||
    parts.includes('gateway time-out') ||
    parts.includes('gateway timeout') ||
    parts.includes('deadline exceeded') ||
    parts.includes('err_unavailable') ||
    parts.includes('unavailable')
  );
}

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
  ]
    .join(' ')
    .toLowerCase();
  return (
    parts.includes('missing request message') ||
    parts.includes('missing authorization header') ||
    parts.includes('unauthenticated') ||
    parts.includes('token expired') ||
    parts.includes('invalid token')
  );
}

export function isStreamServiceProcedure(procLower: string): boolean {
  return procLower.includes('streamservice');
}
