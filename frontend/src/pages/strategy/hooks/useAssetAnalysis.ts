import { useState, useCallback } from 'react';
import { assetAnalysisClient } from '@/client/connect';
import type { AnalyzeAssetResponse } from '@/gen/ant/v1/asset_analysis_pb';
import type { PartialMessage } from '@bufbuild/protobuf';

export type AnalysisPhase = 'idle' | 'mtf_outlook' | 'sr_levels' | 'volatility' | 'ai_recommendation' | 'complete';

const PHASES: AnalysisPhase[] = ['mtf_outlook', 'sr_levels', 'volatility', 'ai_recommendation', 'complete'];

export function useAssetAnalysis() {
  const [accountId, setAccountId] = useState('');
  const [symbol, setSymbol] = useState('');
  const [phase, setPhase] = useState<AnalysisPhase>('idle');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [result, setResult] = useState<PartialMessage<AnalyzeAssetResponse>>({});
  const [progress, setProgress] = useState(0);

  const analyze = useCallback(async () => {
    if (!accountId.trim() || !symbol.trim()) return;
    setLoading(true);
    setPhase('idle');
    setError('');
    setResult({});
    setProgress(0);

    const ctrl = new AbortController();
    try {
      const stream = await assetAnalysisClient.analyzeAsset(
        { accountId: accountId.trim(), symbol: symbol.trim().toUpperCase(), klineCount: 200 },
        { signal: ctrl.signal },
      );

      let idx = 0;
      for await (const frame of stream) {
        if (frame.phase === 'complete') {
          setPhase('complete');
          setProgress(100);
          break;
        }
        if (frame.error) {
          setError(frame.error);
          setPhase('complete');
          setProgress(100);
          break;
        }
        setResult(frame);
        setPhase(frame.phase as AnalysisPhase);
        idx = PHASES.indexOf(frame.phase as AnalysisPhase);
        setProgress(Math.round(((idx + 1) / PHASES.length) * 100));
      }
    } catch (err: unknown) {
      if (!(err instanceof Error && err.name === 'AbortError')) {
        setError(err instanceof Error ? err.message : String(err));
        setPhase('complete');
      }
    } finally {
      setLoading(false);
    }
  }, [accountId, symbol]);

  return { accountId, setAccountId, symbol, setSymbol, phase, loading, error, setError, result, progress, analyze };
}
