
import { STARTED_KEY } from '@/gen/ant/v1/i18n/strategy_tuning_keys';
import { useState, useCallback, useMemo } from 'react'
;
import { message } from 'antd';
import type { TFunction } from 'i18next';
import { backendSweepToSweepDimensions, DEFAULT_SWEEP_DIMS, OPTIMIZER_INFO } from './backtestParamHelpers';
import type { SweepDimension, TuneMethod } from './backtestParamHelpers';

export type { SweepDimension, TuneMethod };
export { OPTIMIZER_INFO };

export type BacktestSubTab = 'results' | 'tuning' | 'gate';

export function useTuning(t: TFunction) {
  const [subTab, setSubTab] = useState<BacktestSubTab>('results');
  const [tuneMethod, setTuneMethod] = useState<TuneMethod>('grid');
  const [sweepDimensions, setSweepDimensions] = useState<SweepDimension[]>(DEFAULT_SWEEP_DIMS);
  const [tuningRunning, setTuningRunning] = useState(false);

  const updateSweepFromCode = useCallback((dims: { key: string; type: string; default: number; min: number; max: number; step: number; hasRange: boolean }[]) => {
    const extracted = backendSweepToSweepDimensions(dims);
    if (extracted.length > 0) { setSweepDimensions(extracted); }
  }, []);

  const enabledSweepDims = useMemo(() => sweepDimensions.filter(d => d.enabled), [sweepDimensions]);
  const cartesianSize = useMemo(() => enabledSweepDims.reduce((acc, d) => acc * d.values.length, 1), [enabledSweepDims]);

  const toggleDimension = useCallback((key: string) => {
    setSweepDimensions(prev => prev.map(d => d.key === key ? { ...d, enabled: !d.enabled } : d));
  }, []);

  const runTuning = useCallback(async (params: {
    code: string; symbol: string; timeframe: string;
    startDate: string; endDate: string;
    templateId?: string;
  }): Promise<string> => {
    setTuningRunning(true);
    try {
      const paramSpace = buildParamSpace(sweepDimensions);
      const { strategyExperimentApi } = await import('@/client/strategyExperiment');
      const fromMs = params.startDate ? new Date(params.startDate).getTime() : 0;
      const toMs = params.endDate ? new Date(params.endDate).getTime() : 0;
      const result = await strategyExperimentApi.submit({
        baseTemplateId: params.templateId || '',
        parameterSpace: paramSpace as Record<string, unknown>,
        searchMethod: tuneMethod,
        maxCandidates: Math.min(cartesianSize || 24, 48),
        objective: 'balanced',
        strategyCode: params.code || '',
        symbol: params.symbol || '',
        timeframe: params.timeframe || '',
        fromTsUnixMs: BigInt(fromMs),
        toTsUnixMs: BigInt(toMs),
      });
      message.success(t(STARTED_KEY));
      return result.experiment?.id || result.jobId || '';
    } catch (e: unknown) {
      message.error(e instanceof Error ? e.message : t('strategy.tuning.failed', { defaultValue: 'Tuning failed' }));
      return '';
    } finally { setTuningRunning(false); }
  }, [sweepDimensions, tuneMethod, cartesianSize, t]);

  return {
    subTab, setSubTab,
    tuneMethod, setTuneMethod,
    sweepDimensions, updateSweepFromCode,
    toggleDimension, enabledSweepDims, cartesianSize,
    tuningRunning, runTuning,
  };
}

function buildParamSpace(sweepDimensions: { enabled: boolean; key: string; values?: number[] }[]): Record<string, number[]> {
  const paramSpace: Record<string, number[]> = {};
  const enabled = sweepDimensions.filter(d => d.enabled);
  for (const dim of enabled) {
    if (dim.values && dim.values.length > 0) paramSpace[dim.key] = dim.values as number[];
    else paramSpace[dim.key] = [0.01, 0.02, 0.03, 0.05, 0.10];
  }
  return paramSpace;
}
