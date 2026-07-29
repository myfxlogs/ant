import { useCallback, useEffect, useState } from 'react';
import type { Dispatch, SetStateAction } from 'react';
import { message } from 'antd';
import { useTranslation } from 'react-i18next';
import {
  MESSAGES_NO_ORDERABLE_SIGNAL_KEY, MESSAGES_ORDER_FAILED_KEY,
  MESSAGES_ORDER_SUBMITTED_KEY, MESSAGES_SIGNAL_HOLD_CANNOT_ORDER_KEY,
  MESSAGES_VOLUME_INVALID_KEY, MESSAGES_EXECUTE_FAILED_KEY,
  MESSAGES_STRATEGY_EXECUTE_FAILED_KEY, MESSAGES_TEMPLATE_CODE_EMPTY_CANNOT_EXECUTE_KEY,
} from '@/gen/ant/v1/i18n/strategy_schedules_keys';
import { strategyScheduleV2Api } from '@/client/strategy-schedules';
import { strategyTemplateApi } from '@/client/strategy-schedules';
import { strategyRuntimeApi } from '@/client/strategyRuntime';
import { tradingApi } from '@/client/trading';
import { getTradingRiskToastMessage } from '@/utils/tradingRiskError';
import type { ScheduleRow, ScheduleHealthSummary, TriggerResult, TriggerContext } from './libraryTypes';
import { scheduleHealthApi } from '@/client/scheduleHealth';
import { HEALTH_MESSAGES_LOAD_FAILED_KEY } from '@/gen/ant/v1/i18n/strategy_schedules_keys';

export function useScheduleTrigger() {
  const { t } = useTranslation();
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

  return { triggering, openTrigger, setOpenTrigger, triggerResult, setTriggerResult, triggerContext, setTriggerContext, onManualTrigger, doOrderSend };
}

export function useScheduleHealth() {
  const { t } = useTranslation();
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

  return { healthOpen, setHealthOpen, healthLoading, healthTarget, setHealthTarget, healthSummary, setHealthSummary, loadScheduleHealth };
}

export function useScheduleSSE(loading: boolean, setSchedules: Dispatch<SetStateAction<ScheduleRow[]>>) {
  const [sseReady, setSseReady] = useState(false);
  useEffect(() => { if (!loading) setSseReady(true); }, [loading]);
  useEffect(() => {
    if (!sseReady) return;
    let active = true;
    const RECONNECT_MS = 90_000;
    let backoffMs = 2000;
    const MAX_BACKOFF_MS = 30_000;

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
          backoffMs = 2000;
        } catch { /* stream error — reconnect with backoff */ }
        if (!active) break;
        await new Promise(r => setTimeout(r, backoffMs));
        backoffMs = Math.min(backoffMs * 2, MAX_BACKOFF_MS);
      }
    };
    connect();
    return () => { active = false; };
    // eslint-disable-next-line react-hooks/exhaustive-deps -- setSchedules is a useState setter, stable across renders
  }, [sseReady]);
  return { sseReady };
}
