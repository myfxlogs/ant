import { getDeviceLocale, getDeviceTimeZone } from '@/utils/date';

export const formatTimestamp = (ts: any): string => {
  if (ts == null || ts === '') return '';
  const locale = getDeviceLocale();
  const timeZone = getDeviceTimeZone();
  if (typeof ts === 'string') {
    // Numeric string from SSE bridge (Unix timestamp as string, e.g. "1717000000")
    if (/^\d+$/.test(ts)) {
      const secs = Number(ts);
      if (secs <= 0) return '';
      return new Date(secs * 1000).toLocaleString(locale, { timeZone });
    }
    return ts;
  }
  if (typeof ts === 'number') {
    if (ts <= 0) return '';
    const date = new Date(ts * 1000);
    return date.toLocaleString(locale, { timeZone });
  }
  if (ts instanceof Date) {
    return ts.toLocaleString(locale, { timeZone });
  }
  if (ts.seconds !== undefined) {
    const seconds = typeof ts.seconds === 'bigint' ? Number(ts.seconds) : ts.seconds;
    const nanos = ts.nanos || 0;
    const date = new Date(seconds * 1000 + nanos / 1000000);
    return date.toLocaleString(locale, { timeZone });
  }
  return '';
};

export const isPendingOrder = (type: string) => ['buy_limit', 'sell_limit', 'buy_stop', 'sell_stop'].includes(type);
