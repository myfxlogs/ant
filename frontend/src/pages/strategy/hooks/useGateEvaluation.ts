import { useState, useCallback, useRef, useEffect } from 'react';
import { gateApi } from '@/client/gate';
import type { GateResult, GatePipelineSummary } from '@/gen/ant/v1/ai_gate_pb';

export function useGateEvaluation() {
  const [gateLoading, setGateLoading] = useState(false);
  const [gateGates, setGateGates] = useState<GateResult[]>([]);
  const [gateSummary, setGateSummary] = useState<GatePipelineSummary | null>(null);
  const [gateError, setGateError] = useState('');
  const gateStopRef = useRef<(() => void) | null>(null);

  // Cleanup SSE connection on unmount.
  useEffect(() => () => { gateStopRef.current?.(); }, []);

  const runGate = useCallback((backtestRunId: string, onSwitchTab: () => void) => {
    if (!backtestRunId) return;
    gateStopRef.current?.();
    setGateLoading(true); setGateGates([]); setGateSummary(null); setGateError('');
    onSwitchTab();
    const stop = gateApi.runEvaluation(
      { backtestRunId },
      {
        onGate: (g) => setGateGates(prev => [...prev, g]),
        onCompleted: (s) => { setGateSummary(s); setGateLoading(false); },
        onError: (e) => { setGateError(String(e?.message ?? e ?? 'Unknown error')); setGateLoading(false); },
      },
    );
    gateStopRef.current = stop;
  }, []);

  return { gateLoading, gateGates, gateSummary, gateError, runGate };
}
