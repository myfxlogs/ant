import { strategyActiveApi } from '@/client/strategy';
import { strategyTemplateApi } from '@/client/strategy-schedules';
import type { ScheduleRow, TriggerResult, TriggerContext } from '../../hooks/libraryTypes';
import type { Dispatch, SetStateAction, MutableRefObject } from 'react';

export interface ManualTriggerParams {
  row: ScheduleRow;
  triggerRunIdRef: MutableRefObject<string | null>;
  triggerAbortRef: MutableRefObject<AbortController | null>;
  stopTriggerRun: () => void;
  setTriggering: (v: boolean) => void;
  setTriggerResult: Dispatch<SetStateAction<TriggerResult | null>>;
  setTriggerContext: (v: TriggerContext | null) => void;
  setOpenTrigger: (v: boolean) => void;
}

export async function manualTriggerStart(p: ManualTriggerParams) {
  const { row, triggerRunIdRef, triggerAbortRef, stopTriggerRun } = p;
  stopTriggerRun();
  p.setTriggering(true);
  p.setTriggerResult(null);
  p.setTriggerContext({ schedule: row, accountId: row.accountId });
  p.setOpenTrigger(true);
  try {
    const tpl = await strategyTemplateApi.get(row.templateId);
    const code = String(tpl?.code || '');
    if (!code) throw new Error('Template code empty');
    const resp = await strategyActiveApi.start({ accountId: row.accountId, strategyCode: code, symbol: row.symbol, timeframe: row.timeframe, mode: 'paper', strategyId: row.templateId, params: row.parameters });
    if (!resp.success) throw new Error(resp.error || 'StartStrategy failed');
    triggerRunIdRef.current = resp.runId;
    p.setTriggerResult({ logs: ['Run started, listening for signals...'], signal: null, meta: { templateId: row.templateId, scheduleId: row.id } });
    const abort = new AbortController();
    triggerAbortRef.current = abort;
    (async () => {
      try {
        for await (const ev of strategyActiveApi.watchSignals(resp.runId, abort.signal)) {
          const s = ev as Record<string, unknown>;
          p.setTriggerResult((prev: TriggerResult | null) => ({ logs: [...(prev?.logs || []), `Signal: ${s.signalType ?? ''} ${s.volume ?? ''} @ ${s.price ?? ''}`], signal: s as TriggerResult['signal'], meta: prev?.meta || {} }));
          p.setTriggering(false);
        }
      } catch (e) {
        if ((e as { name?: string })?.name !== 'AbortError') {
          p.setTriggerResult((prev: TriggerResult | null) => ({ logs: [...(prev?.logs || []), `Stream ended: ${e instanceof Error ? e.message : String(e)}`], signal: prev?.signal ?? null, meta: prev?.meta || {} }));
        }
      }
    })();
  } catch (e: unknown) {
    p.setTriggerResult({ logs: [], signal: null, meta: { error: e instanceof Error ? e.message : String(e) } });
  } finally {
    p.setTriggering(false);
  }
}
