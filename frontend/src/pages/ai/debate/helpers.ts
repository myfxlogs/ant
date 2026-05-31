import { useTranslation } from 'react-i18next';
import type { AIAgentDefinitionView } from '@/client/ai';

/**
 * 8 system built-in Agent types: their display names and descriptions follow i18n locale,
 * reading `ai.settings.agent.types.<type>` and
 * `ai.settings.agent.defaults.<type>.inputHint`, not the fixed string
 * saved by the user in AI settings (stored values are frozen to a language at creation time).
 */
export const BUILTIN_AGENT_TYPES = new Set([
  'style', 'signals', 'risk', 'macro', 'sentiment', 'portfolio', 'execution', 'code',
]);

/** Returns (name, hint) pair: built-in types use i18n, custom types use user-stored values. */
export function useAgentLabel() {
  const { t } = useTranslation();
  return (a: AIAgentDefinitionView) => {
    if (BUILTIN_AGENT_TYPES.has(a.type)) {
      const name = t(`ai.settings.agent.types.${a.type}`, { defaultValue: a.type });
      const hint = t(`ai.settings.agent.defaults.${a.type}.inputHint`, { defaultValue: '' });
      return { name, hint };
    }
    return { name: a.name || a.type, hint: a.inputHint || a.identity || '' };
  };
}

export function formatElapsed(seconds: number): string {
  const m = Math.floor(seconds / 60);
  const s = seconds % 60;
  return `${m}:${String(s).padStart(2, '0')}`;
}

/**
 * Detect "request to advance" phrases — if the user sends such a message, trigger onNext directly without sending to the model.
 * Filter: length ≤ 16 chars, strip whitespace/common punctuation, match against keyword set.
 */
export function looksLikeNextIntent(raw: string): boolean {
  const t = (raw || '').trim();
  if (!t || t.length > 16) return false;
  const stripped = t.toLowerCase().replace(/[\s。.,，！!?？~～、:：;；"'""''`]+/g, '');
  const keywords = new Set([
    // English
    'next', 'next step', 'nextstage', 'nextagent', 'continue', 'proceed', 'go on', 'go ahead',
    // Chinese (zh-cn)
    '下一步', '下一个', '下一环节', '下一阶段', '下一位', '下一轮', '继续',
    // Chinese (zh-tw)
    '下一步', '下一個', '下一環節', '下一階段', '下一位', '下一輪', '繼續',
    // Japanese
    '次へ', '次に', '続ける',
  ]);
  return keywords.has(stripped);
}
