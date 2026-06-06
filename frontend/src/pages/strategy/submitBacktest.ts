import dayjs from 'dayjs';
import { message } from 'antd';
import type { FormInstance } from 'antd';
import type { TFunction } from 'i18next';
import { pythonStrategyApi } from '@/client/pythonStrategy';
import { wrapStrategyCodeWithParams } from './backtestParamInjection';
import { quickRangeLabel, saveRunTitle, type QuickRangeKey } from './StrategyTemplatePage.utils';
import type { StrategyTemplate } from '@/client/strategy';
import { codeAssistApi } from '@/client/codeAssist';

export function applyQuickRange(key: QuickRangeKey, setQuickRange: (k: QuickRangeKey) => void, backtestForm: FormInstance<any>) {
  setQuickRange(key);
  if (key === 'CUSTOM') return;
  const to = dayjs();
  const from = key === '1D' ? to.subtract(1, 'day') : key === '3D' ? to.subtract(3, 'day') : key === '1W' ? to.subtract(1, 'week') : to.subtract(1, 'year');
  backtestForm.setFieldsValue({ range: [from, to] });
}

export async function doSubmitBacktest(
  t: TFunction,
  backtestTemplate: StrategyTemplate | null,
  backtestForm: FormInstance<any>,
  backtestRequiredParams: { key: string; required: boolean }[],
  backtestParamValues: Record<string, string>,
  accounts: { id?: string }[],
  loadSymbols: (accountId: string) => Promise<void>,
) {
  if (!backtestTemplate) return;
  const values = await backtestForm.validateFields();
  const range = values.range as [dayjs.Dayjs, dayjs.Dayjs] | undefined;
  if (!range?.[0] || !range?.[1] || typeof range[0].toDate !== 'function') {
    message.error(t('strategy.templates.messages.selectBacktestRange')); return;
  }
  const fromDate = range[0].toDate(); const toDate = range[1].toDate();
  const extraSymbols = Array.isArray(values.extraSymbols) ? (values.extraSymbols as string[]).map((s) => String(s)).filter((s) => !!s && s !== String(values.symbol)) : [];
  const missingParams = backtestRequiredParams.filter((p) => p.required).filter((p) => { const v = backtestParamValues[p.key]; return v === undefined || v === null || v === ''; });
  if (missingParams.length > 0) {
    message.error(t('strategy.codeAssist.fillRequiredParams', { defaultValue: 'Please fill the required parameters: {{keys}}', keys: missingParams.map((m) => m.key).join(', ') }));
    return;
  }
  const codeToSubmit = wrapStrategyCodeWithParams(String(backtestTemplate?.code || ''), backtestParamValues);
  const resp = await pythonStrategyApi.startBacktestRun({
    code: codeToSubmit, accountId: String(values.accountId), symbol: String(values.symbol),
    timeframe: String(values.timeframe), initialCapital: Number(values.initialCapital || 10000),
    mode: 'KLINE_RANGE', from: fromDate, to: toDate,
    templateId: String(backtestTemplate?.id || '').startsWith('default-') ? undefined : String(backtestTemplate?.id || ''),
    extraSymbols,
  });
  saveRunTitle(String(resp?.runId || ''), String(values.title || dayjs().format('YYYY-MM-DD')));
  message.success(t('strategy.templates.messages.backtestSubmitted'));
  return resp.runId;
}

export async function openBacktestModal(
  t: TFunction,
  template: StrategyTemplate,
  fetchTemplateCodeIfNeeded: (tpl: StrategyTemplate) => Promise<StrategyTemplate>,
  accounts: { id?: string }[],
  loadSymbols: (accountId: string) => Promise<void>,
  backtestForm: FormInstance<any>,
  setBacktestTemplate: (tpl: StrategyTemplate) => void,
  setBacktestRequiredParams: (params: any[]) => void,
  setBacktestParamValues: (vals: Record<string, string>) => void,
  setBacktestModalOpen: (open: boolean) => void,
  setQuickRange: (k: QuickRangeKey) => void,
) {
  let full: StrategyTemplate;
  try { full = await fetchTemplateCodeIfNeeded(template); setBacktestTemplate(full); }
  catch { message.error(t('strategy.templates.messages.readStrategyCodeFailed')); return; }
  setBacktestRequiredParams([]); setBacktestParamValues({});
  try { const ext = await codeAssistApi.validateExtended(String(full?.code || '')); if (ext.valid) setBacktestRequiredParams(ext.parameters || []); }
  catch { /* ignore */ }
  setBacktestModalOpen(true); setQuickRange('1D');
  const defaultAccountId = accounts?.[0]?.id ? String(accounts[0].id) : '';
  const defaultTo = dayjs(); const defaultFrom = dayjs().add(-1, 'day');
  backtestForm.setFieldsValue({
    title: `${dayjs().format('YYYY-MM-DD HH:mm')} ${quickRangeLabel(t, '1D')}`,
    accountId: defaultAccountId, symbol: '', timeframe: 'H1', initialCapital: 10000,
    range: [defaultFrom, defaultTo], extraSymbols: [],
  });
  if (defaultAccountId) await loadSymbols(defaultAccountId);
}
