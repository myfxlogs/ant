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

/** 返回一对 (name, hint)：内置类型优先取 i18n，自定义类型用用户存的。 */
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
    '下一步', '下一个', '下一环节', '下一阶段', '下一位', '下一轮',
    '下一个agent', '下一个智能体', '下一位agent', '下一位智能体',
    '下一位专家', '下一个专家', '下一位能手', '换下一位',
    '继续', '进入下一步', '进入下一个', '进入下一环节', '进入下一阶段',
    '可以了', '可以了下一步', '好的下一步', '好了下一步', '没问题下一步',
    'ok', 'ok下一步', '好下一步', 'next', 'nextstep', 'nextagent',
    'continue', 'proceed', 'goon', 'goahead', '确定下一步', '行下一步',
  ]);
  return keywords.has(stripped);
}
