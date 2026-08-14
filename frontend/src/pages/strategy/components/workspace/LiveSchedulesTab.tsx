import { useState, useEffect, useCallback, useMemo, useRef } from 'react';
import { Button, Form, message } from 'antd';
import { PlusOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { create } from '@bufbuild/protobuf';
import { strategyScheduleV2Api, strategyTemplateApi } from '@/client/strategy-schedules';
import { ScheduleConfigSchema } from '@/gen/ant/v1/strategy_schedule_entity_pb';
import ScheduleTable from '../ScheduleTable';
import EditScheduleModal from '../EditScheduleModal';
import TriggerModal from '../TriggerModal';
import ScheduleHealthModal from '../ScheduleHealthModal';
import ScheduleLogsModal from '../ScheduleLogsModal';
import { useAccountsAndSymbols } from '../../hooks/useAccountsAndSymbols';
import type { ScheduleRow, ScheduleHealthSummary, TriggerResult, TriggerContext, TemplateOption } from '../../hooks/libraryTypes';
import type { ScheduleFormValues, ScheduleType } from '../EditScheduleModal';
import { scheduleHealthApi } from '@/client/scheduleHealth';
import { useNavigate } from 'react-router-dom';
import { strategyActiveApi } from '@/client/strategy';
import { tradingApi } from '@/client/trading';
import { getTradingRiskToastMessage } from '@/utils/tradingRiskError';
import { parseParametersToForm, buildParametersFromForm } from '../../StrategyScheduleParams';
import { DEFAULT_TEMPLATES } from '../../StrategyLibrary.defaults';
import type { DefaultTemplateItem } from '../../StrategyLibrary.defaults';
import { DEFAULT_TIMEFRAME } from '@/constants/timeframes';
import {
  COMMON_UPDATED_KEY, COMMON_CREATED_KEY, COMMON_SAVE_FAILED_KEY,
  COMMON_ENABLED_KEY, COMMON_DISABLED_KEY, COMMON_OPERATION_FAILED_KEY,
  COMMON_DELETED_KEY, COMMON_DELETE_FAILED_KEY,
} from '@/gen/ant/v1/i18n/base_keys';
import { MESSAGES_ORDER_SUBMITTED_KEY, MESSAGES_ORDER_FAILED_KEY, MESSAGES_PARAMETERS_PARSE_FAILED_KEY, CREATE_SCHEDULE_KEY } from '@/gen/ant/v1/i18n/strategy_schedules_keys';

function formatTime(v: unknown): string {
  if (!v) return '-';
  const ms = typeof v === 'bigint' ? Number(v) : typeof v === 'number' ? v : 0;
  return ms ? new Date(ms).toLocaleString() : '-';
}
export function getEnableNavigateTarget(next: boolean): string | null { return next ? '/strategy/live?tab=strategies' : null; }

export default function LiveSchedulesTab({ highlightScheduleId, healthId }: { highlightScheduleId?: string | null; healthId?: string | null }) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [schedules, setSchedules] = useState<ScheduleRow[]>([]);
  const [templates, setTemplates] = useState<TemplateOption[]>([]);
  const [loading, setLoading] = useState(false);
  const [openEdit, setOpenEdit] = useState(false);
  const [editing, setEditing] = useState<ScheduleRow | null>(null);
  const [form] = Form.useForm<ScheduleFormValues>();
  const accountIdWatch = Form.useWatch('accountId', form);
  const { accounts, symbols, symbolsLoading, fetchAccounts, loadSymbols } = useAccountsAndSymbols();

  const [healthOpen, setHealthOpen] = useState(false);
  const [healthLoading, setHealthLoading] = useState(false);
  const [healthTarget, setHealthTarget] = useState<ScheduleRow | null>(null);
  const [logsScheduleId, setLogsScheduleId] = useState<string | null>(null);
  const [healthSummary, setHealthSummary] = useState<ScheduleHealthSummary | null>(null);

  const [triggering, setTriggering] = useState(false);
  const [openTrigger, setOpenTrigger] = useState(false);
  const [triggerResult, setTriggerResult] = useState<TriggerResult | null>(null);
  const [triggerContext, setTriggerContext] = useState<TriggerContext | null>(null);
  const triggerRunIdRef = useRef<string | null>(null);
  const triggerAbortRef = useRef<AbortController | null>(null);

  const stopTriggerRun = useCallback(() => {
    if (triggerAbortRef.current) { triggerAbortRef.current.abort(); triggerAbortRef.current = null; }
    if (triggerRunIdRef.current) { void strategyActiveApi.stop(triggerRunIdRef.current); triggerRunIdRef.current = null; }
  }, []);

  const symbolsOpts = useMemo(() => symbols.map(s => ({ value: s.value, label: s.label })), [symbols]);

  const refresh = useCallback(async () => {
    setLoading(true);
    try {
      const [tpls, schs] = await Promise.all([strategyTemplateApi.list(), strategyScheduleV2Api.list(), fetchAccounts()]);
      const tplsOpts: TemplateOption[] = [];
      const seen = new Set<string>();
      (tpls || []).forEach((t: { id?: string; name?: string; isPublic?: boolean; isSystem?: boolean }) => { if (t?.id) { seen.add(String(t.id)); tplsOpts.push({ id: t.id, name: t.name || '', isPublic: t.isPublic }); } });
      (DEFAULT_TEMPLATES as DefaultTemplateItem[]).forEach(t => { if (t?.id && !seen.has(String(t.id))) tplsOpts.push({ id: String(t.id), name: t.name, isPublic: t.isSystem }); });
      setTemplates(tplsOpts);
      setSchedules(schs as ScheduleRow[]);
    } catch { /* ignore */ } finally { setLoading(false); }
  }, [fetchAccounts]);

  useEffect(() => { void refresh(); }, [refresh]);

  useEffect(() => {
    let active = true;
    const connect = async () => {
      while (active) {
        const ctrl = new AbortController();
        try {
          const streamDone = (async () => {
            for await (const event of strategyScheduleV2Api.watch(ctrl.signal)) {
              if (!active) break;
              setSchedules((event.schedules || []) as ScheduleRow[]);
            }
          })();
          await Promise.race([streamDone, new Promise(r => setTimeout(r, 90_000))]);
          ctrl.abort();
        } catch { /* reconnect */ }
        if (!active) break;
        await new Promise(r => setTimeout(r, 2000));
      }
    };
    connect();
    return () => { active = false; };
  }, []);

  const loadScheduleHealth = useCallback(async (row: ScheduleRow) => {
    if (!row?.id) return;
    setHealthLoading(true);
    try { setHealthSummary(await scheduleHealthApi.getScheduleHealth(row.id) as ScheduleHealthSummary); }
    catch { setHealthSummary(null); } finally { setHealthLoading(false); }
  }, []);

  useEffect(() => {
    if (!healthId || !schedules.length) return;
    const target = schedules.find(s => s.id === healthId);
    if (target) { setHealthTarget(target); void loadScheduleHealth(target); setHealthOpen(true); }
  }, [healthId, schedules, loadScheduleHealth]);

  const onManualTrigger = useCallback(async (row: ScheduleRow) => {
    stopTriggerRun();
    setTriggering(true); setTriggerResult(null);
    setTriggerContext({ schedule: row, accountId: row.accountId }); setOpenTrigger(true);
    try {
      const tpl = await strategyTemplateApi.get(row.templateId);
      const code = String(tpl?.code || '');
      if (!code) throw new Error('Template code empty');
      const resp = await strategyActiveApi.start({ accountId: row.accountId, strategyCode: code, symbol: row.symbol, timeframe: row.timeframe, mode: 'paper', strategyId: row.templateId, params: row.parameters });
      if (!resp.success) throw new Error(resp.error || 'StartStrategy failed');
      triggerRunIdRef.current = resp.runId;
      setTriggerResult({ logs: ['Run started, listening for signals...'], signal: null, meta: { templateId: row.templateId, scheduleId: row.id } });
      const abort = new AbortController(); triggerAbortRef.current = abort;
      (async () => {
        try { for await (const ev of strategyActiveApi.watchSignals(resp.runId, abort.signal)) {
          const s = ev as Record<string, unknown>;
          setTriggerResult(prev => ({ logs: [...(prev?.logs || []), `Signal: ${s.signalType ?? ''} ${s.volume ?? ''} @ ${s.price ?? ''}`], signal: s as TriggerResult['signal'], meta: prev?.meta || {} }));
          setTriggering(false);
        } } catch (e) { if ((e as { name?: string })?.name !== 'AbortError') setTriggerResult(prev => ({ logs: [...(prev?.logs || []), `Stream ended: ${e instanceof Error ? e.message : String(e)}`], signal: prev?.signal ?? null, meta: prev?.meta || {} })); }
      })();
    } catch (e: unknown) { setTriggerResult({ logs: [], signal: null, meta: { error: e instanceof Error ? e.message : String(e) } }); }
    finally { setTriggering(false); }
  }, [stopTriggerRun]);

  const doOrderSend = useCallback(async () => {
    if (!triggerContext?.schedule) return;
    const { schedule } = triggerContext;
    const raw = triggerResult?.signal;
    if (!raw) return;
    const action = String(raw?.type ?? raw?.signalType ?? '').trim().toLowerCase();
    if (!action || action === 'hold') return;
    const volume = Number(raw?.volume || 0);
    if (!(volume > 0)) return;
    try {
      const res = await tradingApi.orderSend({ accountId: schedule.accountId, symbol: raw.symbol || schedule.symbol, type: action, volume, price: Number(raw?.price || 0), stopLoss: Number(raw?.stopLoss || 0), takeProfit: Number(raw?.takeProfit || 0), comment: String(raw?.comment || '') });
      if (res.error) { message.error(getTradingRiskToastMessage({ riskCode: res.riskError?.code, error: res.error, message: res.message, fallback: res.error })); return; }
      message.success(t(MESSAGES_ORDER_SUBMITTED_KEY));
      stopTriggerRun();
      setOpenTrigger(false); setTriggerContext(null); setTriggerResult(null);
    } catch (e: unknown) { message.error(e instanceof Error ? e.message : t(MESSAGES_ORDER_FAILED_KEY)); }
  }, [triggerContext, triggerResult, t, stopTriggerRun]);

  const openCreate = useCallback(() => {
    setEditing(null); form.resetFields();
    form.setFieldsValue({ isActive: true, timeframe: DEFAULT_TIMEFRAME, symbol: '', scheduleType: 'kline_close', intervalMs: 300_000, hfCooldownMs: 1_000, parametersJson: '{}' });
    setOpenEdit(true);
  }, [form]);

  const openUpdate = useCallback((row: ScheduleRow) => {
    setEditing(row);
    const conf = row?.scheduleConfig || {};
    const rawType = String(row?.scheduleType || '').toLowerCase();
    const triggerMode = String(conf?.triggerMode || 'stable_kline');
    let scheduleType: ScheduleType = 'kline_close';
    if (rawType === 'interval') scheduleType = 'interval';
    else if (triggerMode === 'hf_quote_stream') scheduleType = 'hf_quote';
    const toMs = (v: unknown, dft: number) => typeof v === 'number' ? v : typeof v === 'bigint' ? Number(v) : dft;
    const parametersJson = row?.parameters ? JSON.stringify(row.parameters, null, 2) : '{}';
    const parsed = parseParametersToForm(row?.parameters || {});
    form.setFieldsValue({
      id: row?.id, templateId: row?.templateId, accountId: row?.accountId, name: row?.name, symbol: row?.symbol, timeframe: row?.timeframe,
      defaultVolume: parsed.defaultVolume, maxPositions: parsed.maxPositions, stopLossPriceOffset: parsed.stopLossPriceOffset, takeProfitPriceOffset: parsed.takeProfitPriceOffset,
      maxDrawdownPct: parsed.maxDrawdownPct, scheduleType, intervalMs: toMs(conf?.intervalMs, 300_000), hfCooldownMs: toMs(conf?.hfCooldownMs, 1_000), parametersJson,
    });
    void loadSymbols(row?.accountId || '');
    setOpenEdit(true);
  }, [form, loadSymbols]);

  const submitEdit = useCallback(async () => {
    const v = await form.validateFields();
    let params: Record<string, string> = {};
    try { params = v.parametersJson?.trim() ? JSON.parse(v.parametersJson) : {}; } catch { message.error(t(MESSAGES_PARAMETERS_PARSE_FAILED_KEY)); return; }
    const merged = { ...params, ...buildParametersFromForm(v) };
    const sType: ScheduleType = (v.scheduleType || 'kline_close') as ScheduleType;
    const scheduleConfig = create(ScheduleConfigSchema, { cronExpression: '', intervalMs: 0n, eventTrigger: '', triggerMode: sType === 'hf_quote' ? 'hf_quote_stream' : 'stable_kline', stableOverrideIntervalMs: 0n, hfCooldownMs: 0n });
    if (sType === 'interval') scheduleConfig.intervalMs = BigInt(Math.max(1000, Math.floor(Number(v.intervalMs || 300_000))));
    if (sType === 'hf_quote') scheduleConfig.hfCooldownMs = BigInt(Math.max(100, Math.floor(Number(v.hfCooldownMs || 1_000))));
    const backendType = sType === 'interval' ? 'interval' : 'event';
    setLoading(true);
    try {
      if (editing?.id) {
        await strategyScheduleV2Api.update({ id: editing.id, name: v.name, symbol: v.symbol, timeframe: v.timeframe, scheduleType: backendType, scheduleConfig, parameters: merged });
        message.success(t(COMMON_UPDATED_KEY));
      } else {
        const created = await strategyScheduleV2Api.create({ templateId: v.templateId, accountId: v.accountId, name: v.name, symbol: v.symbol, timeframe: v.timeframe, scheduleType: backendType, scheduleConfig, parameters: merged });
        if (v.isActive && created?.id) await strategyScheduleV2Api.toggle(created.id, true);
        message.success(t(COMMON_CREATED_KEY));
      }
      setOpenEdit(false); setEditing(null); form.resetFields(); await refresh();
    } catch (e: unknown) { message.error(e instanceof Error ? e.message : t(COMMON_SAVE_FAILED_KEY)); } finally { setLoading(false); }
  }, [editing, form, refresh, t]);

  const onToggleActive = useCallback(async (row: ScheduleRow, next: boolean) => {
    try { await strategyScheduleV2Api.toggle(row.id, next); message.success(next ? t(COMMON_ENABLED_KEY) : t(COMMON_DISABLED_KEY)); await refresh();
      const target = getEnableNavigateTarget(next);
      if (target) navigate(target);
    } catch (e: unknown) { message.error(e instanceof Error ? e.message : t(COMMON_OPERATION_FAILED_KEY)); }
  }, [refresh, t, navigate]);

  const onDelete = useCallback(async (row: ScheduleRow) => {
    try { await strategyScheduleV2Api.delete(row.id); message.success(t(COMMON_DELETED_KEY)); await refresh(); }
    catch (e: unknown) { message.error(e instanceof Error ? e.message : t(COMMON_DELETE_FAILED_KEY)); }
  }, [refresh, t]);

  useEffect(() => {
    if (!openEdit || editing?.id || !accountIdWatch) return;
    void loadSymbols(accountIdWatch);
  }, [accountIdWatch, openEdit, editing?.id, loadSymbols]);

  useEffect(() => () => stopTriggerRun(), [stopTriggerRun]);

  return (
    <div>
      <div style={{ marginBottom: 12, display: 'flex', justifyContent: 'flex-end' }}>
        <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
          {t(CREATE_SCHEDULE_KEY)}
        </Button>
      </div>

      <ScheduleTable
        schedules={schedules} templates={templates} accounts={accounts}
        loading={loading} triggering={triggering} triggerContext={triggerContext}
        formatTime={formatTime} onEdit={openUpdate} onToggleActive={onToggleActive}
        onHealthCheck={loadScheduleHealth} onManualTrigger={onManualTrigger}
        onDelete={onDelete} onShowLogs={(row) => setLogsScheduleId(row.id)}
        highlightScheduleId={highlightScheduleId}
      />
      <EditScheduleModal
        editing={editing} open={openEdit} loading={loading} form={form}
        templates={templates} accounts={accounts} symbols={symbolsOpts}
        symbolsLoading={symbolsLoading} accountIdWatch={accountIdWatch}
        onCancel={() => { setOpenEdit(false); setEditing(null); form.resetFields(); }} onOk={submitEdit}
      />
      <TriggerModal
        open={openTrigger} triggering={triggering} triggerResult={triggerResult} triggerContext={triggerContext}
        onClose={() => { stopTriggerRun(); setOpenTrigger(false); setTriggerContext(null); setTriggerResult(null); }}
        onRerun={() => { if (triggerContext?.schedule) onManualTrigger(triggerContext.schedule as unknown as ScheduleRow); }}
        onConfirmOrder={doOrderSend}
      />
      <ScheduleHealthModal
        open={healthOpen} loading={healthLoading}
        target={healthTarget as unknown as Record<string, unknown> | null}
        summary={healthSummary as unknown as Record<string, unknown> | null}
        onRefresh={() => { if (healthTarget) loadScheduleHealth(healthTarget); }}
        onClose={() => { setHealthOpen(false); setHealthTarget(null); setHealthSummary(null); }}
        formatTime={formatTime}
      />
      <ScheduleLogsModal
        open={logsScheduleId !== null}
        scheduleId={logsScheduleId}
        onClose={() => setLogsScheduleId(null)}
      />
    </div>
  );
}
