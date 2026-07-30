import { useCallback, useEffect, useMemo, useState } from 'react';
import { Form, message } from 'antd';
import { useTranslation } from 'react-i18next'
import { MESSAGES_PARAMETERS_PARSE_FAILED_KEY } from '@/gen/ant/v1/i18n/strategy_schedules_keys';

;
import { strategyScheduleV2Api, strategyTemplateApi } from '@/client/strategy-schedules';
import { DEFAULT_TIMEFRAME } from '@/constants/timeframes';
import { useAccountsAndSymbols } from './useAccountsAndSymbols';
import { buildSymbolOptions, formatTime } from '../scheduleUtils';
import { buildParametersFromForm, parseParametersToForm } from '../StrategyScheduleParams';
import { DEFAULT_TEMPLATES } from '../StrategyLibrary.defaults';
import type { DefaultTemplateItem } from '../StrategyLibrary.defaults';
import type { ScheduleFormValues } from '../components/EditScheduleModal';
import type { ScheduleRow, TemplateOption } from './libraryTypes';
import type { StrategyTemplate } from '@/client/strategy';
import { useScheduleTrigger, useScheduleHealth, useScheduleSSE } from './useLibrarySchedulesSub';

type ScheduleType = 'interval' | 'kline_close' | 'hf_quote';

function resolveScheduleType(rawType: string, triggerMode: string): ScheduleType {
  if (rawType === 'interval') return 'interval';
  if (rawType === 'event' || rawType === 'cron') {
    return triggerMode === 'hf_quote_stream' ? 'hf_quote' : 'kline_close';
  }
  if (rawType === 'hf_quote') return 'hf_quote';
  return 'kline_close';
}

export function useLibrarySchedules(selectedTemplateId: string) {
  const { t } = useTranslation();
  const [schedules, setSchedules] = useState<ScheduleRow[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [templates, setTemplates] = useState<StrategyTemplate[]>([]);
  const { accounts, symbols, symbolsLoading, fetchAccounts, loadSymbols } = useAccountsAndSymbols();
  const [openEdit, setOpenEdit] = useState(false);
  const [editing, setEditing] = useState<ScheduleRow | null>(null);
  const [form] = Form.useForm<ScheduleFormValues>();

  const symbolsOpts = useMemo(() => buildSymbolOptions(symbols), [symbols]);
  const accountIdWatch = Form.useWatch('accountId', form);

  const health = useScheduleHealth();
  const trigger = useScheduleTrigger();

  const filteredSchedules = useMemo(() => {
    if (!selectedTemplateId) return schedules;
    return schedules.filter(s => String(s.templateId || '') === selectedTemplateId);
  }, [schedules, selectedTemplateId]);

  const refresh = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const [tpls, schs] = await Promise.all([strategyTemplateApi.list(), strategyScheduleV2Api.list()]);
      setTemplates(tpls || []); setSchedules(schs as ScheduleRow[]); void fetchAccounts();
    } catch (e: unknown) {
      console.error('fetchSchedules failed', e);
      setError(e instanceof Error ? e.message : t('common.loadingFailed'));
    } finally { setLoading(false); }
  }, [t, fetchAccounts]);

  useEffect(() => { void refresh(); }, []); // eslint-disable-line react-hooks/exhaustive-deps

  useScheduleSSE(loading, setSchedules);

  useEffect(() => { if (selectedTemplateId) void refresh(); }, [selectedTemplateId, refresh]);

  const templatesForSelect = useMemo((): TemplateOption[] => {
    const out: TemplateOption[] = []; const seen = new Set<string>();
    templates.forEach(t => { if (!t?.id) return; seen.add(String(t.id)); out.push({ id: t.id, name: t.name, isPublic: t.isPublic }); });
    (DEFAULT_TEMPLATES as DefaultTemplateItem[]).forEach(t => { if (!t?.id) return; const id = String(t.id); if (seen.has(id)) return; out.push({ id: String(t.id), name: t.name, isPublic: t.isSystem }); });
    return out;
  }, [templates]);

  const openCreate = useCallback(() => {
    setEditing(null); form.resetFields();
    form.setFieldsValue({
      isActive: true, timeframe: DEFAULT_TIMEFRAME, symbol: '',
      scheduleType: 'kline_close', intervalMs: 300_000, hfCooldownMs: 1_000,
      parametersJson: '{}', templateId: selectedTemplateId || undefined,
    });
    setOpenEdit(true);
  }, [form, selectedTemplateId]);

  const openUpdate = useCallback((row: ScheduleRow) => {
    setEditing(row);
    const formValues = buildScheduleFormValues(row);
    form.setFieldsValue(formValues);
    void loadSymbols(row?.accountId, row?.symbol);
    setOpenEdit(true);
  }, [form, loadSymbols]);

  const submitEdit = useCallback(async () => {
    const v = await form.validateFields();
    const parsed = parseParamsJson(v.parametersJson, t);
    if (!parsed.ok) return;
    const merged = { ...parsed.params, ...buildParametersFromForm(v) };
    const sType: ScheduleType = (v.scheduleType || 'kline_close') as ScheduleType;
    const scheduleConfig = buildScheduleConfig(sType, v);
    const backendScheduleType = sType === 'interval' ? 'interval' : 'event';
    setLoading(true);
    try {
      if (editing?.id) {
        await strategyScheduleV2Api.update({
          id: editing.id, name: v.name, symbol: v.symbol, timeframe: v.timeframe,
          scheduleType: backendScheduleType, scheduleConfig, parameters: merged,
        });
        message.success(t('common.updated'));
      } else {
        const created = await strategyScheduleV2Api.create({
          templateId: v.templateId, accountId: v.accountId, name: v.name,
          symbol: v.symbol, timeframe: v.timeframe, scheduleType: backendScheduleType,
          scheduleConfig, parameters: merged,
        });
        if (v.isActive && created?.id) {
          await strategyScheduleV2Api.toggle(created.id, true);
        }
        message.success(t('common.created'));
      }
      setOpenEdit(false); setEditing(null); form.resetFields(); await refresh();
    } catch (e: unknown) { message.error(e instanceof Error ? e.message : t('common.saveFailed')); }
    finally { setLoading(false); }
  }, [editing, form, refresh, t]);

  const onToggleActive = useCallback(async (row: ScheduleRow, next: boolean) => {
    try {
      await strategyScheduleV2Api.toggle(row.id, next);
      message.success(next ? t('common.enabled') : t('common.disabled'));
      await refresh();
    } catch (e: unknown) { console.error('toggleSchedule failed', e); message.error(e instanceof Error ? e.message : t('common.operationFailed')); }
  }, [refresh, t]);

  const onDelete = useCallback(async (row: ScheduleRow) => {
    try {
      await strategyScheduleV2Api.delete(row.id);
      message.success(t('common.deleted'));
      await refresh();
    } catch (e: unknown) { console.error('deleteSchedule failed', e); message.error(e instanceof Error ? e.message : t('common.deleteFailed')); }
  }, [refresh, t]);

  useEffect(() => {
    if (!openEdit || editing?.id || !accountIdWatch) return;
    void loadSymbols(accountIdWatch);
  }, [accountIdWatch, openEdit, editing?.id, loadSymbols]);

  return {
    schedules: filteredSchedules, allSchedules: schedules,
    loading, error, templates: templatesForSelect,
    accounts, symbols: symbolsOpts, symbolsLoading,
    openEdit, setOpenEdit, editing, setEditing, form, accountIdWatch,
    ...health, ...trigger,
    formatTime, loadSymbols,
    refresh, openCreate, openUpdate, submitEdit, onToggleActive, onDelete,
  };
}

function resolveBigNumber(val: unknown, defaultVal: number): number {
  if (typeof val === 'number') return val;
  if (typeof val === 'bigint') return Number(val);
  return defaultVal;
}

function buildScheduleFormValues(row: ScheduleRow): Record<string, unknown> {
  const r = row ?? {};
  const conf = r.scheduleConfig ?? {};
  const rawType = String(r.scheduleType ?? '').toLowerCase();
  const triggerMode = String(conf.triggerMode ?? 'stable_kline');
  const scheduleType = resolveScheduleType(rawType, triggerMode);
  const intervalMs = resolveBigNumber(conf.intervalMs, 300_000);
  const hfCooldownMs = resolveBigNumber(conf.hfCooldownMs, 1_000);
  const parametersJson = r.parameters ? JSON.stringify(r.parameters, null, 2) : '{}';
  const parsedParams = parseParametersToForm(r.parameters ?? {});
  return {
    id: r.id, templateId: r.templateId, accountId: r.accountId,
    name: r.name, symbol: r.symbol, timeframe: r.timeframe,
    defaultVolume: parsedParams.defaultVolume, maxPositions: parsedParams.maxPositions,
    stopLossPriceOffset: parsedParams.stopLossPriceOffset, takeProfitPriceOffset: parsedParams.takeProfitPriceOffset,
    maxDrawdownPct: parsedParams.maxDrawdownPct, scheduleType, intervalMs, hfCooldownMs, parametersJson,
  };
}

function parseParamsJson(parametersJson: string, t: (k: string) => string): { ok: boolean; params: Record<string, string> } {
  try {
    const params = parametersJson && parametersJson.trim() ? JSON.parse(parametersJson) : {};
    return { ok: true, params };
  } catch {
    message.error(t(MESSAGES_PARAMETERS_PARSE_FAILED_KEY));
    return { ok: false, params: {} };
  }
}

function buildScheduleConfig(sType: ScheduleType, v: Record<string, unknown>): Record<string, unknown> {
  const config: Record<string, unknown> = {
    cronExpression: '', intervalMs: 0n, eventTrigger: '',
    triggerMode: sType === 'hf_quote' ? 'hf_quote_stream' : 'stable_kline',
    stableOverrideIntervalMs: 0n, hfCooldownMs: 0n,
  };
  if (sType === 'interval') {
    const ms = Math.max(1000, Math.floor(Number(v.intervalMs || 300_000)));
    config.intervalMs = BigInt(ms);
  }
  if (sType === 'hf_quote') {
    const cd = Math.max(100, Math.floor(Number(v.hfCooldownMs || 1_000)));
    config.hfCooldownMs = BigInt(cd);
  }
  return config;
}
