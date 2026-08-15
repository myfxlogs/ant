export function formatTime(v: unknown): string {
  if (!v) return '-';
  const ms = typeof v === 'bigint' ? Number(v) : typeof v === 'number' ? v : 0;
  return ms ? new Date(ms).toLocaleString() : '-';
}

export function getEnableNavigateTarget(next: boolean): string | null {
  return next ? '/strategy/live?tab=strategies' : null;
}
