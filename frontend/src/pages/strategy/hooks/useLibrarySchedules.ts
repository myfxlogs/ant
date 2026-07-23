import { useCallback, useEffect, useMemo, useState } from 'react';
import { Form, message } from 'antd';
import { useTranslation } from 'react-i18next'
import { HEALTH_MESSAGES_LOAD_FAILED_KEY, MESSAGES_EXECUTE_FAILED_KEY, MESSAGES_NO_ORDERABLE_SIGNAL_KEY, MESSAGES_ORDER_FAILED_KEY, MESSAGES_ORDER_SUBMITTED_KEY, MESSAGES_PARAMETERS_PARSE_FAILED_KEY, MESSAGES_SIGNAL_HOLD_CANNOT_ORDER_KEY, MESSAGES_STRATEGY_EXECUTE_FAILED_KEY, MESSAGES_TEMPLATE_CODE_EMPTY_CANNOT_EXECUTE_KEY, MESSAGES_VOLUME_INVALID_KEY } from '@/gen/ant/v1/i18n/strategy_schedules_keys';

;
import { strategyScheduleV2Api, strategyTemplateApi } from '@/client/strategy-schedules';
import { strategyRuntimeApi } from '@/client/strategyRuntime';
import { tradingApi } from '@/client/trading';
import { DEFAULT_TIMEFRAME } from '@/constants/timeframes';
import { scheduleHealthApi } from '@/client/scheduleHealth';
import { useAccountsAndSymbols } from './useAccountsAndSymbols';
import { buildSymbolOptions, formatTime } from '../scheduleUtils';
import { buildParametersFromForm, parseParametersToForm } from '../StrategyScheduleParams';
import { DEFAULT_TEMPLATES } from '../StrategyLibrary.defaults';
import type { DefaultTemplateItem } from '../StrategyLibrary.defaults';
import { getTradingRiskToastMessage } from '@/utils/tradingRiskError';
import type { ScheduleFormValues } from '../components/EditScheduleModal';
import type { ScheduleRow, ScheduleHealthSummary, TriggerResult, TriggerContext, TemplateOption } from './libraryTypes';
import type { StrategyTemplate } from '@/client/strategy';

type ScheduleType = 'interval' | 'kline_close' | 'hf_quote';

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

  // ── Health state (lazy-loaded per schedule) ──
  const [healthOpen, setHealthOpen] = useState(false);
  const [healthLoading, setHealthLoading] = useState(false);
  const [healthTarget, setHealthTarget] = useState<ScheduleRow | null>(null);
  const [healthSummary, setHealthSummary] = useState<ScheduleHealthSummary | null>(null);

  const loadScheduleHealth = useCallback(async (row: ScheduleRow) => {
    if (!row?.id) return; setHealthLoading(true);
    try { setHealthSummary(await scheduleHealthApi.getScheduleHealth(row.id) as ScheduleHealthSummary); }
    catch (e: unknown) { message.error(e instanceof Error ? e.message : t(HEALTH_MESSAGES_LOAD_FAILED_KEY)); setHealthSummary(null); }
    finally { setHealthLoading(false); }
  }, [t]);

  // ── Trigger state ──
  const [triggering, setTriggering] = useState(false);
  const [openTrigger, setOpenTrigger] = useState(false);
  const [triggerResult, setTriggerResult] = useState<TriggerResult | null>(null);
  const [triggerContext, setTriggerContext] = useState<TriggerContext | null>(null);

  const onManualTrigger = useCallback(async (row: ScheduleRow) => {
    setTriggering(true); setTriggerResult(null);
    setTriggerContext({ schedule: row, accountId: row.accountId }); setOpenTrigger(true);
    try {
      const tpl = await strategyTemplateApi.get(row.templateId);
      const code = String(tpl?.code || '');
      if (!code) throw new Error(t(MESSAGES_TEMPLATE_CODE_EMPTY_CANNOT_EXECUTE_KEY));
      const exec = await strategyRuntimeApi.execute({ code, accountId: row.accountId, symbol: row.symbol, timeframe: row.timeframe });
      if (!exec.success) throw new Error(exec.error || t(MESSAGES_STRATEGY_EXECUTE_FAILED_KEY));
      setTriggerResult({ logs: exec.logs || [], signal: exec.signal ?? null, meta: { templateId: row.templateId, scheduleId: row.id } });
    } catch (e: unknown) {
      setTriggerResult({ logs: [], signal: null, meta: { error: e instanceof Error ? e.message : t(MESSAGES_EXECUTE_FAILED_KEY) } });
    }
    finally { setTriggering(false); }
  }, [t]);

  const doOrderSend = useCallback(async () => {
    if (!triggerContext?.schedule) return;
    const { schedule } = triggerContext;
    const raw = triggerResult?.signal;
    if (!raw) { message.error(t(MESSAGES_NO_ORDERABLE_SIGNAL_KEY)); return; }
    const signal = raw;
    const rawAction = String(signal?.type ?? signal?.signalType ?? '').trim().toLowerCase();
    const action = rawAction === 'buy' || rawAction === 'sell' ? rawAction : '';
    const volumeNum = typeof signal?.volume === 'number' ? signal.volume : Number(signal?.volume);
    const volume = Number.isFinite(volumeNum) ? volumeNum : 0;
    if (!action || rawAction === 'hold') { message.error(t(MESSAGES_SIGNAL_HOLD_CANNOT_ORDER_KEY)); return; }
    if (!(volume > 0)) { message.error(t(MESSAGES_VOLUME_INVALID_KEY)); return; }
    const payload = {
      accountId: schedule.accountId, symbol: signal.symbol || schedule.symbol, type: action, volume,
      price: typeof signal?.price === 'number' ? signal.price : Number(signal?.price || 0),
      stopLoss: typeof signal?.stopLoss === 'number' ? signal.stopLoss : Number(signal?.stopLoss || 0),
      takeProfit: typeof signal?.takeProfit === 'number' ? signal.takeProfit : Number(signal?.takeProfit || 0),
      comment: String(signal?.comment || ''),
    };
    try {
      const res = await tradingApi.orderSend(payload);
      if (res.error) { message.error(getTradingRiskToastMessage({ riskCode: res.riskError?.code, error: res.error, message: res.message, fallback: res.error || t(MESSAGES_ORDER_FAILED_KEY) })); return; }
      message.success(t(MESSAGES_ORDER_SUBMITTED_KEY));
      setOpenTrigger(false); setTriggerContext(null); setTriggerResult(null);
    } catch (e: unknown) { message.error(e instanceof Error ? e.message : t(MESSAGES_ORDER_FAILED_KEY)); }
  }, [triggerContext, triggerResult, t]);

  // ── Schedule CRUD ──
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

  // SSE — start after initial fetch, proactive reconnect before Cloudflare 100s timeout.
  const [sseReady, setSseReady] = useState(false);
  useEffect(() => { if (!loading) setSseReady(true); }, [loading]);
  useEffect(() => {
    if (!sseReady) return;
    let active = true;
    const RECONNECT_MS = 90_000; // reconnect before Cloudflare 100s timeout
    let backoffMs = 2000;
    const MAX_BACKOFF_MS = 30_000;

    const connect = async () => {
      while (active) {
        const ctrl = new AbortController();
        try {
          // Race: stream vs proactive reconnect timer.
          const streamDone = (async () => {
            for await (const event of strategyScheduleV2Api.watch(ctrl.signal)) {
              if (!active) break;
              setSchedules((event.schedules || []) as ScheduleRow[]);
            }
          })();
          const timerDone = new Promise(r => setTimeout(r, RECONNECT_MS));
          await Promise.race([streamDone, timerDone]);
          ctrl.abort(); // proactively abort before Cloudflare kills it
          backoffMs = 2000; // reset backoff on clean disconnect
        } catch { /* stream error — reconnect with backoff */ }
        if (!active) break;
        await new Promise(r => setTimeout(r, backoffMs));
        backoffMs = Math.min(backoffMs * 2, MAX_BACKOFF_MS);
      }
    };
    connect();
    return () => { active = false; };
  }, [sseReady]);

  // Re-fetch on template change
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
    const conf = row?.scheduleConfig || {};
    const rawType = String(row?.scheduleType || '').toLowerCase();
    const triggerMode = String(conf?.triggerMode || 'stable_kline');
    let scheduleType: ScheduleType;
    if (rawType === 'interval') scheduleType = 'interval';
    else if (rawType === 'event') {
      if (triggerMode === 'hf_quote_stream') scheduleType = 'hf_quote';
      else scheduleType = 'kline_close';
    }
    // Backward compat: old records stored kline_close/hf_quote as "cron" with triggerMode.
    else if (rawType === 'cron') {
      if (triggerMode === 'hf_quote_stream') scheduleType = 'hf_quote';
      else scheduleType = 'kline_close';
    }
    else if (rawType === 'hf_quote') scheduleType = 'hf_quote';
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
    catch { message.error(t(MESSAGES_PARAMETERS_PARSE_FAILED_KEY)); return; }
    const merged = { ...params, ...buildParametersFromForm(v) };
    const sType: ScheduleType = (v.scheduleType || 'kline_close') as ScheduleType;
    const scheduleConfig: Record<string, unknown> = {
      cronExpression: '', intervalMs: 0n, eventTrigger: '',
      triggerMode: sType === 'hf_quote' ? 'hf_quote_stream' : 'stable_kline',
      stableOverrideIntervalMs: 0n, hfCooldownMs: 0n,
    };
    if (sType === 'interval') { const ms = Math.max(1000, Math.floor(Number(v.intervalMs || 300_000))); scheduleConfig.intervalMs = BigInt(ms); }
    if (sType === 'hf_quote') { const cd = Math.max(100, Math.floor(Number(v.hfCooldownMs || 1_000))); scheduleConfig.hfCooldownMs = BigInt(cd); }
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
    healthOpen, setHealthOpen, healthLoading, healthTarget, setHealthTarget, healthSummary, setHealthSummary,
    triggering, openTrigger, setOpenTrigger, triggerResult, triggerContext, setTriggerContext, setTriggerResult,
    formatTime, loadSymbols,
    refresh, openCreate, openUpdate, submitEdit, onToggleActive, onDelete, onManualTrigger, loadScheduleHealth,
    doOrderSend,
  };
}
