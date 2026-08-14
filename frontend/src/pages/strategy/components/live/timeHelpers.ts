export function secondsSince(ts: { seconds?: bigint; nanos?: number } | null | undefined): number {
  if (!ts || !ts.seconds) return Infinity;
  return (Date.now() - Number(ts.seconds) * 1000) / 1000;
}

export function formatAgo(ts: { seconds?: bigint; nanos?: number } | null | undefined): string {
  if (!ts || !ts.seconds) return '-';
  const ms = Number(ts.seconds) * 1000;
  const diff = Math.max(0, Date.now() - ms);
  if (diff < 60_000) return 'now';
  const m = Math.floor(diff / 60_000);
  if (m < 60) return `${m}m ago`;
  const h = Math.floor(m / 60);
  return `${h}h ago`;
}
