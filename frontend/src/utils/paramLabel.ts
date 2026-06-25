/**
 * Look up the display label for a strategy parameter name in the current locale.
 * Falls back to the raw parameter name if no translation is available.
 *
 * Priority: TemplateI18n.Params[name].label[locale] → TemplateI18n.Params[name].label[en] → raw name
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
    || labels['zh-cn']
    || name;
}
