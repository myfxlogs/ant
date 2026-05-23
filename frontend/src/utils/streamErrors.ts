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

export function isStreamServiceProcedure(procLower: string): boolean {
  return procLower.includes('streamservice') || procLower.includes('debatev2streamservice');
}
