import type { TFunction } from 'i18next';

export function formatMode(mode: string | undefined, t: TFunction): string {
  if (!mode) return '-';
  const key = `strategy.live.mode_${mode}`;
  const fallback = mode.charAt(0).toUpperCase() + mode.slice(1);
  return t(key, { defaultValue: fallback });
}
