import { codeAssistApi } from '@/client/codeAssist';

const SUPPORTED_LOCALES = ['en', 'zh-cn', 'zh-tw', 'ja', 'vi'] as const;

/**
 * Look up the display label for a strategy parameter name in the current locale.
 * Falls back to the raw parameter name if no translation is available.
 *
 * Priority: TemplateI18n.Params[name].label[locale] → en → raw name
 */
export function paramLabel(
  name: string,
  locale: string,
  i18nData: { params?: Record<string, { label?: Record<string, string> }> } | null | undefined,
): string {
  if (!i18nData?.params?.[name]?.label) return name;

  const labels = i18nData.params[name].label!;
  return labels[locale]
    || labels[locale.split('-')[0]]
    || labels['en']
    || name;
}

/**
 * Translate parameter names to all supported locales and build TemplateI18n JSON.
 * Returns a JSON string suitable for storage in strategy_templates.i18n.
 * Returns empty string on failure (translation is best-effort, never blocks save).
 */
export async function buildParamI18n(parametersJson: string): Promise<string> {
  if (!parametersJson) return '';
  try {
    const params: { name: string }[] = JSON.parse(parametersJson);
    const names = params.map(p => p.name).filter(Boolean);
    if (names.length === 0) return '';

    const translations = await codeAssistApi.translateParamLabels(names);
    const i18n: { params: Record<string, { label: Record<string, string> }> } = { params: {} };

    for (const name of names) {
      const labels: Record<string, string> = {};
      for (const locale of SUPPORTED_LOCALES) {
        labels[locale] = translations[locale]?.[name] || name;
      }
      i18n.params[name] = { label: labels };
    }

    return JSON.stringify(i18n);
  } catch {
    return '';
  }
}
