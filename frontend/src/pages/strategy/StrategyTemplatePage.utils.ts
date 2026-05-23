export type QuickRangeKey = 'CUSTOM' | '1D' | '3D' | '1W' | '1Y';

export const clamp01 = (x: number) => {
  if (x < 0) return 0;
  if (x > 1) return 1;
  return x;
};

export const pickNum = (v: any): number | undefined => {
  if (v === null || v === undefined) return undefined;
  if (typeof v === 'number' && Number.isFinite(v)) return v;
  if (typeof v === 'string' && v.trim() !== '') {
    const n = Number(v);
    return Number.isFinite(n) ? n : undefined;
  }
  return undefined;
};

export const pickMetric = (metrics: any, keys: string[]): number | undefined => {
  for (const k of keys) {
    const n = pickNum(metrics?.[k]);
    if (typeof n === 'number') return n;
  }
  return undefined;
};

export const quickRangeLabel = (t: (key: string) => string, key: QuickRangeKey) => {
  switch (key) {
    case '1D':
      return t('strategy.templates.backtest.quickRange.1d');
    case '3D':
      return t('strategy.templates.backtest.quickRange.3d');
    case '1W':
      return t('strategy.templates.backtest.quickRange.1w');
    case '1Y':
      return t('strategy.templates.backtest.quickRange.1y');
    default:
      return t('strategy.templates.backtest.quickRange.custom');
  }
};

export const isTerminalRun = (run: any) => {
  return Boolean(run?.isTerminal || run?.is_terminal);
};

export const isSucceededRun = (run: any) => {
  return Boolean(run?.isSucceeded || run?.is_succeeded);
};

export const isErrTemplateNotDraft = (e: any): boolean => {
  const msg = String(e?.rawMessage || e?.message || e || '');
  return msg.toLowerCase().includes('not a draft') || msg.toLowerCase().includes('template is not a draft');
};

export const getRunTemplateRef = (run: any): { templateId?: string; templateDraftId?: string } => {
  const templateId = String(run?.templateId || run?.template_id || '').trim();
  const templateDraftId = String(run?.templateDraftId || run?.template_draft_id || '').trim();
  return {
    templateId: templateId || undefined,
    templateDraftId: templateDraftId || undefined,
  };
};

const runTitleStorageKey = 'backtest_run_titles_v1';

export const loadRunTitles = (): Record<string, string> => {
  try {
    const raw = localStorage.getItem(runTitleStorageKey);
    if (!raw) return {};
    const obj = JSON.parse(raw);
    if (!obj || typeof obj !== 'object') return {};
    return obj as Record<string, string>;
  } catch (_e) {
    return {};
  }
};

export const saveRunTitle = (runId: string, title: string) => {
  if (!runId) return;
  try {
    const titles = loadRunTitles();
    titles[String(runId)] = String(title || '');
    localStorage.setItem(runTitleStorageKey, JSON.stringify(titles));
  } catch (_e) {
    // ignore
  }
};
