import i18n from '@/i18n';

const AI_PREFS_KEY = 'ai_user_preferences_v1';

export function loadUserPrefs(): string {
  try {
    return localStorage.getItem(AI_PREFS_KEY) || '';
  } catch {
    return '';
  }
}

export function saveUserPrefs(next: string) {
  try {
    localStorage.setItem(AI_PREFS_KEY, next);
  } catch {
    // localStorage may be unavailable
  }
}

export function strategyValidationRulesText(): string {
  return [
    i18n.t('ai.store.strategyRules.title'),
    '',
    i18n.t('ai.store.strategyRules.rules.noImport'),
    i18n.t('ai.store.strategyRules.rules.noGlobal'),
    i18n.t('ai.store.strategyRules.rules.noDunderAccess'),
    i18n.t('ai.store.strategyRules.rules.noDunderName'),
    i18n.t('ai.store.strategyRules.rules.noDangerousCalls'),
    i18n.t('ai.store.strategyRules.rules.runSignature'),
    i18n.t('ai.store.strategyRules.rules.mustDefineEntry'),
    '',
    i18n.t('ai.store.strategyRules.allowedGlobals'),
  ].join('\n');
}

export function buildChatContext(): string {
  const prefs = loadUserPrefs().trim();
  const parts: string[] = [];
  parts.push(`Locale: ${i18n.language || ''}`.trim());
  parts.push('');
  parts.push(strategyValidationRulesText());
  if (prefs) {
    parts.push('');
    parts.push(i18n.t('ai.store.context.userPrefsTitle'));
    parts.push(prefs);
  }
  parts.push('');
  parts.push(i18n.t('ai.store.context.outputTitle'));
  parts.push(i18n.t('ai.store.context.outputRules.wrapPython'));
  parts.push(i18n.t('ai.store.context.outputRules.validateFirst'));
  parts.push(i18n.t('ai.store.context.outputRules.noImport'));
  return parts.join('\n');
}
