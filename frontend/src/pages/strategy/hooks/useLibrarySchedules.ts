import { useCallback, useEffect, useMemo, useState } from 'react';
import { Form, message } from 'antd';
import { useTranslation } from 'react-i18next';
import { strategyScheduleV2Api, strategyTemplateApi } from '@/client/strategy-schedules';
import { DEFAULT_TIMEFRAME } from '@/constants/timeframes';
import { scheduleHealthApi } from '@/client/scheduleHealth';
import { logApi } from '@/client/log';
import { useAccountsAndSymbols } from './useAccountsAndSymbols';
import { buildSymbolOptions, formatTime } from '../scheduleUtils';
import { buildParametersFromForm, parseParametersToForm } from '../StrategyScheduleParams';
import { DEFAULT_TEMPLATES } from '../StrategyTemplatePage.defaults';
import type { ScheduleFormValues } from '../components/EditScheduleModal';

type ScheduleType = 'interval' | 'kline_close' | 'hf_quote';

export function useLibrarySchedules(selectedTemplateId: string) {
  const { t } = useTranslation();
  const [schedules, setSchedules] = useState<any[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [templates, setTemplates] = useState<any[]>([]);
  const { accounts, symbols, symbolsLoading, fetchAccounts, loadSymbols } = useAccountsAndSymbols();
  const [openEdit, setOpenEdit] = useState(false);
  const [editing, setEditing] = useState<any | null>(null);
  const [healthOpen, setHealthOpen] = useState(false);
  const [healthLoading, setHealthLoading] = useState(false);
  const [healthTarget, setHealthTarget] = useState<any | null>(null);
  const [healthSummary, setHealthSummary] = useState<any | null>(null);
  const [triggering, setTriggering] = useState(false);
  const [openTrigger, setOpenTrigger] = useState(false);
  const [triggerResult, setTriggerResult] = useState<any>(null);
  const [triggerContext, setTriggerContext] = useState<any>(null);
  const [form] = Form.useForm<ScheduleFormValues>();

  const symbolsOpts = useMemo(() => buildSymbolOptions(symbols), [symbols]);
  const accountIdWatch = Form.useWatch('accountId', form);

  // Filter schedules by selected template
  const filteredSchedules = useMemo(() => {
    if (!selectedTemplateId) return schedules;
    return schedules.filter((s: any) => String(s.templateId || '') === selectedTemplateId);
  }, [schedules, selectedTemplateId]);

  const refresh = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const [tpls, schs] = await Promise.all([strategyTemplateApi.list(), strategyScheduleV2Api.list()]);
      setTemplates(tpls as any[]); setSchedules(schs as any[]); void fetchAccounts();
    } catch (e: any) {
      const msg = e?.message || t('common.loadingFailed'); setError(msg);
    } finally { setLoading(false); }
  }, [t, fetchAccounts]);

  useEffect(() => { void refresh(); }, [refresh]);

  // SSE streaming for live schedule updates
  useEffect(() => {
    const ctrl = new AbortController();
    (async () => {
      try {
        for await (const event of strategyScheduleV2Api.watch(ctrl.signal)) {
          setSchedules(event.schedules as any[] || []);
        }
      } catch { /* stream closed */ }
    })();
    return () => ctrl.abort();
  }, []);

  // Re-fetch schedules when template selection changes (ensures full list is loaded)
  useEffect(() => { void refresh(); }, [selectedTemplateId]); // eslint-disable-line react-hooks/exhaustive-deps

  const templatesForSelect = useMemo(() => {
    const out: any[] = []; const seen = new Set<string>();
    (templates || []).forEach((t: any) => { if (!t?.id) return; seen.add(String(t.id)); out.push(t); });
    (DEFAULT_TEMPLATES || []).forEach((t: any) => { if (!t?.id) return; const id = String(t.id); if (seen.has(id)) return; out.push(t); });
    return out;
  }, [templates]);

  const openCreate = useCallback(() => {
    setEditing(null); form.resetFields();
    form.setFieldsValue({
      isActive: true, timeframe: DEFAULT_TIMEFRAME, symbol: '',
      scheduleType: 'kline_close', intervalMs: 300_000, hfCooldownMs: 1_000,
      parametersJson: '{}',
      templateId: selectedTemplateId || undefined,
    });
    setOpenEdit(true);
  }, [form, selectedTemplateId]);

  const openUpdate = useCallback((row: any) => {
    setEditing(row);
    const conf = row?.scheduleConfig || {};
    const rawType = String(row?.scheduleType || '').toLowerCase();
    const triggerMode = String(conf?.triggerMode || 'stable_kline');
    let scheduleType: ScheduleType;
    if (rawType === 'interval' || rawType === 'cron') scheduleType = 'interval';
    else if (triggerMode === 'hf_quote_stream' || rawType === 'hf_quote') scheduleType = 'hf_quote';
    else scheduleType = 'kline_close';
    const intervalMs = typeof conf?.intervalMs === 'number' ? conf.intervalMs : typeof conf?.intervalMs === 'bigint' ? Number(conf.intervalMs) : 300_000;
    const hfCooldownMs = typeof conf?.hfCooldownMs === 'number' ? conf.hfCooldownMs : typeof conf?.hfCooldownMs === 'bigint' ? Number(conf.hfCooldownMs) : 1_000;
    const parametersJson = row?.parameters ? JSON.stringify(row.parameters, null, 2) : '{}';
    const parsedParams = parseParametersToForm(row?.parameters || {});
    form.setFieldsValue({
      id: row?.id, templateId: row?.templateId, accountId: row?.accountId,
      name: row?.name, symbol: row?.symbol, timeframe: row?.timeframe,
      defaultVolume: parsedParams.defaultVolume, maxPositions: parsedParams.maxPositions,
      stopLossPriceOffset: parsedParams.stopLossPriceOffset, takeProfitPriceOffset: parsedParams.takeProfitPriceOffset,
      maxDrawdownPct: parsedParams.maxDrawdownPct, scheduleType, intervalMs, hfCooldownMs, parametersJson,
    });
    void loadSymbols(row?.accountId, row?.symbol);
    setOpenEdit(true);
  }, [form, loadSymbols]);

  const submitEdit = useCallback(async () => {
    const v = await form.validateFields();
    let params: Record<string, string> = {};
    try { params = v.parametersJson && v.parametersJson.trim() ? JSON.parse(v.parametersJson) : {}; }
    catch { message.error(t('strategy.schedules.messages.parametersParseFailed')); return; }
    const merged = { ...params, ...buildParametersFromForm(v) };
    const sType: ScheduleType = (v.scheduleType || 'kline_close') as ScheduleType;
    const scheduleConfig: Record<string, unknown> = {
      cronExpression: '', intervalMs: 0n, eventTrigger: '',
      triggerMode: sType === 'hf_quote' ? 'hf_quote_stream' : 'stable_kline',
      stableOverrideIntervalMs: 0n, hfCooldownMs: 0n,
    };
    if (sType === 'interval') { const ms = Math.max(1000, Math.floor(Number(v.intervalMs || 300_000))); scheduleConfig.intervalMs = BigInt(ms); }
    if (sType === 'hf_quote') { const cd = Math.max(100, Math.floor(Number(v.hfCooldownMs || 1_000))); scheduleConfig.hfCooldownMs = BigInt(cd); }
    const backendScheduleType = sType === 'interval' ? 'interval' : 'cron';
    setLoading(true);
    try {
      if (editing?.id) {
        await strategyScheduleV2Api.update({
          id: editing.id, name: v.name, symbol: v.symbol, timeframe: v.timeframe,
          scheduleType: backendScheduleType, scheduleConfig: scheduleConfig as any, parameters: merged,
        });
        message.success(t('common.updated'));
      } else {
        await strategyScheduleV2Api.create({
          templateId: v.templateId, accountId: v.accountId, name: v.name,
          symbol: v.symbol, timeframe: v.timeframe, scheduleType: backendScheduleType,
          scheduleConfig: scheduleConfig as any, parameters: merged,
        });
        if (v.isActive) {
          // Find the created schedule to toggle it — SSE will update the list
          const all = await strategyScheduleV2Api.list();
          const created = (all as any[]).find((s: any) => s.templateId === v.templateId && s.accountId === v.accountId && s.symbol === v.symbol);
          if (created?.id) await strategyScheduleV2Api.toggle(created.id, true);
        }
        message.success(t('common.created'));
      }
      setOpenEdit(false); setEditing(null); form.resetFields(); await refresh();
    } catch (e: any) { message.error(e?.message || t('common.saveFailed')); }
    finally { setLoading(false); }
  }, [editing, form, refresh, t]);

  const onToggleActive = useCallback(async (row: any, next: boolean) => {
    try {
      await strategyScheduleV2Api.toggle(row.id, next);
      message.success(next ? t('common.enabled') : t('common.disabled'));
      await refresh();
    } catch (e: any) { message.error(e?.message || t('common.operationFailed')); }
  }, [refresh, t]);

  const onDelete = useCallback(async (row: any) => {
    try {
      await strategyScheduleV2Api.delete(row.id);
      message.success(t('common.deleted'));
      await refresh();
    } catch (e: any) { message.error(e?.message || t('common.deleteFailed')); }
  }, [refresh, t]);

  const onManualTrigger = useCallback(async (row: any) => {
    setTriggering(true); setTriggerResult(null); setTriggerContext({ schedule: row, accountId: row.accountId }); setOpenTrigger(true);
    try {
      const { pythonStrategyApi } = await import('@/client/pythonStrategy');
      const tpl = await strategyTemplateApi.get(row.templateId);
      const code = String((tpl as any)?.code || '');
      if (!code) throw new Error(t('strategy.schedules.messages.templateCodeEmptyCannotExecute'));
      const exec = await pythonStrategyApi.execute({ code, accountId: row.accountId, symbol: row.symbol, timeframe: row.timeframe });
      if (!exec.success) throw new Error(exec.error || t('strategy.schedules.messages.strategyExecuteFailed'));
      setTriggerResult({ logs: exec.logs || [], signal: exec.signal, meta: { templateId: row.templateId, scheduleId: row.id } });
    } catch (e: any) { setTriggerResult({ logs: [], signal: null, meta: { error: e?.message || t('strategy.schedules.messages.executeFailed') } }); }
    finally { setTriggering(false); }
  }, [t]);

  const loadScheduleHealth = useCallback(async (row: any) => {
    if (!row?.id) return; setHealthLoading(true);
    try { setHealthSummary(await scheduleHealthApi.getScheduleHealth(row.id)); }
    catch (e: any) { message.error(e?.message || t('strategy.schedules.health.messages.loadFailed')); setHealthSummary(null); }
    finally { setHealthLoading(false); }
  }, [t]);

  // Load symbols when account changes in create mode
  useEffect(() => {
    if (!openEdit || editing?.id || !accountIdWatch) return;
    void loadSymbols(accountIdWatch);
  }, [accountIdWatch, openEdit, editing?.id, loadSymbols]);

  return {
    schedules: filteredSchedules, allSchedules: schedules,
    loading, error, templates: templatesForSelect,
    accounts, symbols: symbolsOpts, symbolsLoading,
    openEdit, setOpenEdit, editing, setEditing, form, accountIdWatch,
    healthOpen, setHealthOpen, healthLoading, healthTarget, setHealthTarget, healthSummary, setHealthSummary,
    triggering, openTrigger, setOpenTrigger, triggerResult, triggerContext, setTriggerContext, setTriggerResult,
    formatTime, loadSymbols,
    refresh, openCreate, openUpdate, submitEdit, onToggleActive, onDelete, onManualTrigger, loadScheduleHealth,
  };
}
