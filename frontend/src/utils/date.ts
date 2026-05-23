import type { Timestamp } from '@bufbuild/protobuf/wkt';
import i18n, { normalizeLanguage } from '@/i18n';

export function getDeviceLocale(): string {
  if (typeof navigator === 'undefined') return 'en-US';
  const lang = (Array.isArray(navigator.languages) && navigator.languages[0]) || navigator.language;
  return lang || 'en-US';
}

export function getDeviceTimeZone(): string | undefined {
  try {
    const tz = Intl?.DateTimeFormat?.().resolvedOptions?.().timeZone;
    return tz || undefined;
  } catch (_e) {
    return undefined;
  }
}

export function getUILocale(): string {
  const lang = normalizeLanguage(i18n.language);
  if (lang === 'zh-cn') return 'zh-CN';
  if (lang === 'zh-tw') return 'zh-TW';
  if (lang === 'ja') return 'ja-JP';
  if (lang === 'vi') return 'vi-VN';
  return 'en-US';
}

/** Localized full month name for calendar month 1–12 (UI language). */
export function formatMonthLongName(month: number): string {
  if (month < 1 || month > 12) return '';
  const d = new Date(Date.UTC(2000, month - 1, 1));
  return d.toLocaleString(getUILocale(), { month: 'long', timeZone: 'UTC' });
}

export function formatDateTime(dateStr: string | Timestamp | null | undefined): string {
  if (!dateStr) return '-';
  const locale = getUILocale();
  const timeZone = getDeviceTimeZone();
  
  if (typeof dateStr === 'object' && 'seconds' in dateStr) {
    const date = timestampToDate(dateStr as Timestamp);
    return date.toLocaleString(locale, {
      timeZone,
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
      hour12: false
    }).replace(/\//g, '-');
  }
  
  let cleanStr = dateStr as string;
  if (cleanStr.endsWith('Z')) {
    cleanStr = cleanStr.replace('Z', '+00:00');
  }
  cleanStr = cleanStr.replace(/\.\d+/, '');
  cleanStr = cleanStr.replace('T', ' ');
  
  const date = new Date(cleanStr);
  if (isNaN(date.getTime())) {
    return dateStr as string;
  }
  
  return date.toLocaleString(locale, {
    timeZone,
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hour12: false
  }).replace(/\//g, '-');
}

export function formatDate(dateStr: string | Timestamp | null | undefined): string {
  if (!dateStr) return '-';
  const locale = getUILocale();
  const timeZone = getDeviceTimeZone();
  
  if (typeof dateStr === 'object' && 'seconds' in dateStr) {
    const date = timestampToDate(dateStr as Timestamp);
    return date.toLocaleDateString(locale, {
      timeZone,
      year: 'numeric',
      month: '2-digit',
      day: '2-digit'
    }).replace(/\//g, '-');
  }
  
  let cleanStr = dateStr as string;
  if (cleanStr.endsWith('Z')) {
    cleanStr = cleanStr.replace('Z', '+00:00');
  }
  cleanStr = cleanStr.replace(/\.\d+/, '');
  cleanStr = cleanStr.replace('T', ' ');
  
  const date = new Date(cleanStr);
  if (isNaN(date.getTime())) {
    return dateStr as string;
  }
  
  return date.toLocaleDateString(locale, {
    timeZone,
    year: 'numeric',
    month: '2-digit',
    day: '2-digit'
  }).replace(/\//g, '-');
}

export function formatTime(dateStr: string | Timestamp | null | undefined): string {
  if (!dateStr) return '-';
  const locale = getUILocale();
  const timeZone = getDeviceTimeZone();
  
  if (typeof dateStr === 'object' && 'seconds' in dateStr) {
    const date = timestampToDate(dateStr as Timestamp);
    return date.toLocaleTimeString(locale, {
      timeZone,
      hour: '2-digit',
      minute: '2-digit',
      hour12: false
    });
  }
  
  let cleanStr = dateStr as string;
  if (cleanStr.endsWith('Z')) {
    cleanStr = cleanStr.replace('Z', '+00:00');
  }
  cleanStr = cleanStr.replace(/\.\d+/, '');
  cleanStr = cleanStr.replace('T', ' ');
  
  const date = new Date(cleanStr);
  if (isNaN(date.getTime())) {
    return dateStr as string;
  }
  
  return date.toLocaleTimeString(locale, {
    timeZone,
    hour: '2-digit',
    minute: '2-digit',
    hour12: false
  });
}

export function formatHoldingTime(raw: string | null | undefined): string {
  if (!raw || raw === '-') return '-';
  const s = String(raw).trim();
  if (!s) return '-';

  const lang = normalizeLanguage(i18n.language);
  const needsSpace = lang === 'en' || lang === 'vi';

  if (/^<\s*1\s*分钟$/.test(s)) {
    return i18n.t('common.time.lessThanMinute');
  }

  const m = s.match(/^(\d+)\s*(分钟|小时|天)$/);
  if (!m) return s;

  const value = Number(m[1]);
  if (!Number.isFinite(value)) return s;

  const unitCN = m[2];
  let unitKey = '';
  if (unitCN === '分钟') unitKey = 'common.time.minute';
  if (unitCN === '小时') unitKey = 'common.time.hour';
  if (unitCN === '天') unitKey = 'common.time.day';
  if (!unitKey) return s;

  const unit = i18n.t(unitKey);
  return needsSpace ? `${value} ${unit}` : `${value}${unit}`;
}

function timestampToDate(timestamp: Timestamp): Date {
  const ms = Number(timestamp.seconds) * 1000 + Math.floor(Number(timestamp.nanos) / 1000000);
  return new Date(ms);
}

/** Human-readable duration from seconds (e.g. average holding); uses i18n time units. */
export function formatDurationFromSeconds(seconds: number | null | undefined): string {
  if (seconds == null || !Number.isFinite(seconds) || seconds <= 0) {
    return '—';
  }
  const lang = normalizeLanguage(i18n.language);
  const needsSpace = lang === 'en' || lang === 'vi';
  if (seconds < 60) {
    return i18n.t('common.time.lessThanMinute');
  }
  if (seconds < 3600) {
    const minutes = Math.floor(seconds / 60);
    const unit = i18n.t('common.time.minute');
    return needsSpace ? `${minutes} ${unit}` : `${minutes}${unit}`;
  }
  if (seconds < 86400) {
    const hours = Math.floor(seconds / 3600);
    const unit = i18n.t('common.time.hour');
    return needsSpace ? `${hours} ${unit}` : `${hours}${unit}`;
  }
  const days = Math.floor(seconds / 86400);
  const unit = i18n.t('common.time.day');
  return needsSpace ? `${days} ${unit}` : `${days}${unit}`;
}
