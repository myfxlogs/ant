import { codeAssistApi } from '@/client/codeAssist';
import type { ParameterEntry } from '@/gen/ant/v1/parameter_entry_pb';
import type { TemplateI18n } from '@/gen/ant/v1/strategy_template_entity_pb';
import { create, fromBinary, toBinary } from '@bufbuild/protobuf';
import { TemplateI18nSchema } from '@/gen/ant/v1/strategy_template_entity_pb';

const SUPPORTED_LOCALES = ['en', 'zh-cn', 'zh-tw', 'ja', 'vi'] as const;

interface I18nLike {
  locales?: Record<string, { labels?: Record<string, string> }>;
}

/**
 * Look up the display label for a strategy parameter name in the current locale.
 * Falls back to the raw parameter name if no translation is available.
 *
 * Priority: TemplateI18n.locales[locale].labels[name] → en → raw name
 */
export function paramLabel(
  name: string,
  locale: string,
  i18nData: I18nLike | TemplateI18n | null | undefined,
): string {
  const locales = (i18nData as I18nLike | null)?.locales;
  if (!locales) return name;

  const tryLocale = (lc: string) => locales[lc]?.labels?.[name];

  return tryLocale(locale)
    || tryLocale(locale.split('-')[0])
    || tryLocale('en')
    || name;
}

/**
 * Translate parameter names to all supported locales and build a TemplateI18n proto.
 * Returns a serialized TemplateI18n binary (Uint8Array) for storage in strategy_templates.i18n.
 * Returns null on failure (translation is best-effort, never blocks save).
 */
export async function buildParamI18n(parameters: ParameterEntry[]): Promise<Uint8Array | null> {
  if (!parameters || parameters.length === 0) return null;
  try {
    const names = parameters.map(p => p.name).filter(Boolean);
    if (names.length === 0) return null;

    const translations = await codeAssistApi.translateParamLabels(names);
    const locales: Record<string, { labels: Record<string, string> }> = {};

    for (const locale of SUPPORTED_LOCALES) {
      const labels: Record<string, string> = {};
      for (const name of names) {
        labels[name] = translations[locale]?.[name] || name;
      }
      locales[locale] = { labels };
    }

    const i18n = create(TemplateI18nSchema, { locales });
    return toBinary(TemplateI18nSchema, i18n);
  } catch {
    return null;
  }
}

/**
 * Deserialize a TemplateI18n from binary bytes (from DB).
 */
export function parseI18n(bytes: Uint8Array | null | undefined): TemplateI18n | undefined {
  if (!bytes || bytes.length === 0) return undefined;
  try {
    return fromBinary(TemplateI18nSchema, bytes);
  } catch {
    return undefined;
  }
}
