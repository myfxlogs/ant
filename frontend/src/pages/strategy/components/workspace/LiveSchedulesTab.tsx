import { useState, useEffect, useCallback, useMemo } from 'react';
import { Button, Form, message } from 'antd';
import { PlusOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { strategyScheduleV2Api, strategyTemplateApi } from '@/client/strategy-schedules';
import ScheduleTable from '../ScheduleTable';
import EditScheduleModal from '../EditScheduleModal';
import TriggerModal from '../TriggerModal';
import ScheduleHealthModal from '../ScheduleHealthModal';
import { useAccountsAndSymbols } from '../../hooks/useAccountsAndSymbols';
import type { ScheduleRow, ScheduleHealthSummary, TriggerResult, TriggerContext, TemplateOption } from '../../hooks/libraryTypes';
import type { ScheduleFormValues, ScheduleType } from '../EditScheduleModal';
import { scheduleHealthApi } from '@/client/scheduleHealth';
import { strategyRuntimeApi } from '@/client/strategyRuntime';
import { tradingApi } from '@/client/trading';
import { getTradingRiskToastMessage } from '@/utils/tradingRiskError';
import { parseParametersToForm, buildParametersFromForm } from '../../StrategyScheduleParams';
import { DEFAULT_TEMPLATES } from '../../StrategyLibrary.defaults';
import type { DefaultTemplateItem } from '../../StrategyLibrary.defaults';
import { DEFAULT_TIMEFRAME } from '@/constants/timeframes';

function formatTime(v: unknown): string {
  if (!v) return '-';
  const ms = typeof v === 'bigint' ? Number(v) : typeof v === 'number' ? v : 0;
  if (!ms) return '-';
  return new Date(ms).toLocaleString();
}

export default function LiveSchedulesTab() {
  const { t } = useTranslation();
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
  const [healthSummary, setHealthSummary] = useState<ScheduleHealthSummary | null>(null);

  const [triggering, setTriggering] = useState(false);
  const [openTrigger, setOpenTrigger] = useState(false);
  const [triggerResult, setTriggerResult] = useState<TriggerResult | null>(null);
  const [triggerContext, setTriggerContext] = useState<TriggerContext | null>(null);

  const symbolsOpts = useMemo(() => symbols.map(s => ({ value: s, label: s })), [symbols]);

  const refresh = useCallback(async () => {
    setLoading(true);
    try {
      const [tpls, schs] = await Promise.all([strategyTemplateApi.list(), strategyScheduleV2Api.list()]);
      const tplsOpts: TemplateOption[] = [];
      const seen = new Set<string>();
      (tpls || []).forEach((t: any) => { if (t?.id) { seen.add(String(t.id)); tplsOpts.push({ id: t.id, name: t.name, isPublic: t.isPublic }); } });
      (DEFAULT_TEMPLATES as DefaultTemplateItem[]).forEach(t => { if (t?.id && !seen.has(String(t.id))) tplsOpts.push({ id: String(t.id), name: t.name, isPublic: t.isSystem }); });
      setTemplates(tplsOpts);
      setSchedules(schs as ScheduleRow[]);
      void fetchAccounts();
    } catch { /* ignore */ } finally { setLoading(false); }
  }, [fetchAccounts]);

  useEffect(() => { void refresh(); }, [refresh]);

  useEffect(() => {
    let active = true;
    const RECONNECT_MS = 90_000;
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
          const timerDone = new Promise(r => setTimeout(r, RECONNECT_MS));
          await Promise.race([streamDone, timerDone]);
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

  const onManualTrigger = useCallback(async (row: ScheduleRow) => {
    setTriggering(true); setTriggerResult(null);
    setTriggerContext({ schedule: row, accountId: row.accountId }); setOpenTrigger(true);
    try {
      const tpl = await strategyTemplateApi.get(row.templateId);
      const code = String(tpl?.code || '');
      if (!code) throw new Error('Template code empty');
      const exec = await strategyRuntimeApi.execute({ code, accountId: row.accountId, symbol: row.symbol, timeframe: row.timeframe });
      if (!exec.success) throw new Error(exec.error || 'Execute failed');
      setTriggerResult({ logs: exec.logs || [], signal: exec.signal as any, meta: { templateId: row.templateId, scheduleId: row.id } });
    } catch (e: any) { setTriggerResult({ logs: [], signal: null, meta: { error: e?.message } }); }
    finally { setTriggering(false); }
  }, []);

  const doOrderSend = useCallback(async () => {
    if (!triggerContext?.schedule) return;
    const { schedule } = triggerContext;
    const raw = triggerResult?.signal;
    if (!raw) return;
    const action = String(raw?.type ?? raw?.signalType ?? raw?.signal ?? '').trim().toLowerCase();
    if (!action || action === 'hold') return;
    const volume = Number(raw?.volume || 0);
    if (!(volume > 0)) return;
    try {
      const res = await tradingApi.orderSend({
        accountId: schedule.accountId, symbol: raw.symbol || schedule.symbol,
        type: action, volume, price: Number(raw?.price || 0),
        stopLoss: Number(raw?.stopLoss || 0), takeProfit: Number(raw?.takeProfit || 0), comment: String(raw?.comment || ''),
      });
      if (res.error) { message.error(getTradingRiskToastMessage({ riskCode: res.riskError?.code, error: res.error, message: res.message, fallback: res.error })); return; }
      message.success(t('strategy.schedules.orderSubmitted', 'Order submitted'));
      setOpenTrigger(false); setTriggerContext(null); setTriggerResult(null);
    } catch (e: any) { message.error(e?.message || 'Order failed'); }
  }, [triggerContext, triggerResult, t]);

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
    const intervalMs = typeof conf?.intervalMs === 'number' ? conf.intervalMs : typeof conf?.intervalMs === 'bigint' ? Number(conf.intervalMs) : 300_000;
    const hfCooldownMs = typeof conf?.hfCooldownMs === 'number' ? conf.hfCooldownMs : typeof conf?.hfCooldownMs === 'bigint' ? Number(conf.hfCooldownMs) : 1_000;
    const parametersJson = row?.parameters ? JSON.stringify(row.parameters, null, 2) : '{}';
    const parsed = parseParametersToForm(row?.parameters || {});
    form.setFieldsValue({
      id: row?.id, templateId: row?.templateId, accountId: row?.accountId, name: row?.name, symbol: row?.symbol, timeframe: row?.timeframe,
      defaultVolume: parsed.defaultVolume, maxPositions: parsed.maxPositions, stopLossPriceOffset: parsed.stopLossPriceOffset, takeProfitPriceOffset: parsed.takeProfitPriceOffset,
      maxDrawdownPct: parsed.maxDrawdownPct, scheduleType, intervalMs, hfCooldownMs, parametersJson,
    });
    void loadSymbols(row?.accountId, row?.symbol);
    setOpenEdit(true);
  }, [form, loadSymbols]);

  const submitEdit = useCallback(async () => {
    const v = await form.validateFields();
    let params: Record<string, string> = {};
    try { params = v.parametersJson?.trim() ? JSON.parse(v.parametersJson) : {}; } catch { message.error('Parameters parse failed'); return; }
    const merged = { ...params, ...buildParametersFromForm(v) };
    const sType: ScheduleType = (v.scheduleType || 'kline_close') as ScheduleType;
    const scheduleConfig: Record<string, unknown> = {
      cronExpression: '', intervalMs: 0n, eventTrigger: '',
      triggerMode: sType === 'hf_quote' ? 'hf_quote_stream' : 'stable_kline',
      stableOverrideIntervalMs: 0n, hfCooldownMs: 0n,
    };
    if (sType === 'interval') { const ms = Math.max(1000, Math.floor(Number(v.intervalMs || 300_000))); scheduleConfig.intervalMs = BigInt(ms); }
    if (sType === 'hf_quote') { const cd = Math.max(100, Math.floor(Number(v.hfCooldownMs || 1_000))); scheduleConfig.hfCooldownMs = BigInt(cd); }
    const backendType = sType === 'interval' ? 'interval' : 'event';
    setLoading(true);
    try {
      if (editing?.id) {
        await strategyScheduleV2Api.update({ id: editing.id, name: v.name, symbol: v.symbol, timeframe: v.timeframe, scheduleType: backendType, scheduleConfig: scheduleConfig as any, parameters: merged });
        message.success(t('common.updated'));
      } else {
        const created: any = await strategyScheduleV2Api.create({ templateId: v.templateId, accountId: v.accountId, name: v.name, symbol: v.symbol, timeframe: v.timeframe, scheduleType: backendType, scheduleConfig: scheduleConfig as any, parameters: merged });
        if (v.isActive && created?.id) await strategyScheduleV2Api.toggle(created.id, true);
        message.success(t('common.created'));
      }
      setOpenEdit(false); setEditing(null); form.resetFields(); await refresh();
    } catch (e: any) { message.error(e?.message || t('common.saveFailed')); } finally { setLoading(false); }
  }, [editing, form, refresh, t]);

  const onToggleActive = useCallback(async (row: ScheduleRow, next: boolean) => {
    try { await strategyScheduleV2Api.toggle(row.id, next); message.success(next ? t('common.enabled') : t('common.disabled')); await refresh(); }
    catch (e: any) { message.error(e?.message || t('common.operationFailed')); }
  }, [refresh, t]);

  const onDelete = useCallback(async (row: ScheduleRow) => {
    try { await strategyScheduleV2Api.delete(row.id); message.success(t('common.deleted')); await refresh(); }
    catch (e: any) { message.error(e?.message || t('common.deleteFailed')); }
  }, [refresh, t]);

  useEffect(() => {
    if (!openEdit || editing?.id || !accountIdWatch) return;
    void loadSymbols(accountIdWatch);
  }, [accountIdWatch, openEdit, editing?.id, loadSymbols]);

  return (
    <div>
      <div style={{ marginBottom: 12, display: 'flex', justifyContent: 'flex-end' }}>
        <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
          {t('strategy.schedules.create', 'Create Schedule')}
        </Button>
      </div>

      <ScheduleTable
        schedules={schedules}
        templates={templates}
        accounts={accounts}
        loading={loading}
        triggering={triggering}
        triggerContext={triggerContext}
        formatTime={formatTime}
        onEdit={openUpdate}
        onToggleActive={onToggleActive}
        onHealthCheck={loadScheduleHealth}
        onManualTrigger={onManualTrigger}
        onDelete={onDelete}
      />

      <EditScheduleModal
        editing={editing}
        open={openEdit}
        loading={loading}
        form={form}
        templates={templates}
        accounts={accounts}
        symbols={symbolsOpts}
        symbolsLoading={symbolsLoading}
        accountIdWatch={accountIdWatch}
        onCancel={() => { setOpenEdit(false); setEditing(null); form.resetFields(); }}
        onOk={submitEdit}
      />
      <TriggerModal
        open={openTrigger}
        triggering={triggering}
        result={triggerResult}
        context={triggerContext}
        onCancel={() => { setOpenTrigger(false); setTriggerContext(null); setTriggerResult(null); }}
        onOrderSend={doOrderSend}
      />
      <ScheduleHealthModal
        open={healthOpen}
        loading={healthLoading}
        target={healthTarget}
        summary={healthSummary}
        onClose={() => { setHealthOpen(false); setHealthTarget(null); setHealthSummary(null); }}
      />
    </div>
  );
}
