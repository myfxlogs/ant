import { useState, useRef, useCallback } from 'react';
import { executePlan, diagnosePlan, type ExecuteCallbacks, type PlanCallbacks } from '@/client/strategyPlan';
import { strategyRuntimeApi } from '@/client/strategyRuntime';
import { isSucceededRun } from '@/pages/strategy/StrategyTemplatePage.utils';
import type { ToolCall, ToolResult, BacktestMetricsMsg } from '@/gen/ant/v1/strategy_execution_pb';
import { BacktestMetricsMsgSchema } from '@/gen/ant/v1/strategy_execution_pb';
import { create } from '@bufbuild/protobuf';
import type { BacktestRunUpdate } from '@/gen/ant/v1/backtest_run_query_pb';

export type Phase = 'idle' | 'generating' | 'tool_call' | 'tool_result' | 'done' | 'error';

interface UseExecutionPanelArgs {
  plan: string;
  symbol?: string;
  timeframe?: string;
  sessionId?: string;
  previousCode?: string;
}

export function useExecutionPanel({ plan, symbol, timeframe, sessionId, previousCode }: UseExecutionPanelArgs) {
  const [phase, setPhase] = useState<Phase>('idle');
  const [currentPhase, setCurrentPhase] = useState('');
  const [streamCode, setStreamCode] = useState('');
  const [code, setCode] = useState('');
  const [prevCode, setPrevCode] = useState(previousCode || '');
  const [toolResults, setToolResults] = useState<ToolResult[]>([]);
  const [currentTool, setCurrentTool] = useState('');
  const [error, setError] = useState('');
  const [feedback, setFeedback] = useState('');
  const [analysis, setAnalysis] = useState('');
  const [discussionReply] = useState('');
  const [metrics, setMetrics] = useState<BacktestMetricsMsg | null>(null);
  const [diagnosis, setDiagnosis] = useState('');
  const [pendingFeedback, setPendingFeedback] = useState('');
  const abortRef = useRef<(() => void) | null>(null);
  const watchRef = useRef<(() => void) | null>(null);

  useState(() => () => { abortRef.current?.(); watchRef.current?.(); });

  const start = useCallback(() => {
    setPhase('generating'); setStreamCode(''); setCode(''); setPrevCode(previousCode || ''); setError('');
    setToolResults([]); setCurrentTool('');

    const abort = executePlan(
      { plan, conversationId: sessionId, symbol, timeframe, previousCode: previousCode || '' },
      {
        onPhase: (p) => { setCurrentPhase(p); if (p === 'generating') setPhase('generating'); },
        onDelta: (d) => setStreamCode(s => s + d),
        onCode: (c) => { setCode(c); setStreamCode(''); },
        onPreviousCode: (c) => setPrevCode(c),
        onAnalysis: (a) => setAnalysis(a),
        onToolCall: (tc: ToolCall) => { setPhase('tool_call'); setCurrentTool(tc.name); },
        onToolResult: (tr: ToolResult) => {
          setPhase('tool_result'); setCurrentTool('');
          setToolResults(prev => [...prev, tr]);
          if (tr.name === 'backtest' && tr.success && tr.outputJson) {
            try {
              const out = JSON.parse(tr.outputJson) as { run_id: string };
              if (out.run_id) {
                watchRef.current?.();
                watchRef.current = strategyRuntimeApi.watchBacktestRun(out.run_id, (update: BacktestRunUpdate) => {
                  if (update.run && isSucceededRun(update.run) && update.metrics) {
                    setMetrics(create(BacktestMetricsMsgSchema, {
                      totalReturn: update.metrics.totalReturn, sharpeRatio: update.metrics.sharpeRatio,
                      maxDrawdown: update.metrics.maxDrawdown, winRate: update.metrics.winRate,
                      totalTrades: update.metrics.totalTrades, profitFactor: update.metrics.profitFactor,
                    }));
                  }
                });
              }
            } catch { /* ignore */ }
          }
        },
        onError: (e) => { setError(e); setPhase('error'); },
        onDone: () => { setPhase('done'); },
      } satisfies ExecuteCallbacks,
    );
    abortRef.current = abort;
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  const started = useRef(false);
  if (!started.current) { started.current = true; setTimeout(start, 0); }

  const handleFeedback = useCallback(() => {
    const msg = feedback.trim();
    if (!msg) return;
    setFeedback('');
    setDiagnosis('');
    setPendingFeedback(msg);
    setPhase('generating'); setError('');

    const abort = diagnosePlan(
      { plan, conversationId: sessionId, feedbackMessage: msg,
        currentCode: code || streamCode,
        backtestMetricsJson: metrics ? JSON.stringify(metrics) : '' },
      {
        onDelta: () => {},
        onPlan: (p) => { setDiagnosis(p); setPhase('done'); },
        onError: (e) => { setError(e); setPhase('error'); },
        onDone: () => {},
      } satisfies PlanCallbacks,
    );
    abortRef.current = abort;
  }, [feedback, plan, sessionId, code, streamCode, metrics]);

  const handleConfirmDiagnosis = useCallback(() => {
    const msg = pendingFeedback;
    setDiagnosis(''); setPendingFeedback('');
    const currentCode = code || streamCode;
    setPrevCode(currentCode); setStreamCode(''); setCode(''); setAnalysis('');
    setPhase('generating'); setToolResults([]); setError(''); setMetrics(null);

    const abort = executePlan(
      { plan, conversationId: sessionId, symbol, timeframe, previousCode: currentCode,
        feedbackMessage: msg, backtestMetricsJson: metrics ? JSON.stringify(metrics) : '' },
      {
        onPhase: (p) => setCurrentPhase(p),
        onAnalysis: (a) => setAnalysis(a),
        onDelta: (d) => setStreamCode(s => s + d),
        onCode: (c) => { setCode(c); setStreamCode(''); setPhase('done'); },
        onPreviousCode: (c) => setPrevCode(c),
        onToolCall: (tc: ToolCall) => { setPhase('tool_call'); setCurrentTool(tc.name); },
        onToolResult: (tr: ToolResult) => {
          setPhase('tool_result'); setCurrentTool('');
          setToolResults(prev => [...prev, tr]);
        },
        onError: (e) => { setError(e); setPhase('error'); },
        onDone: () => { setPhase('done'); },
      } satisfies ExecuteCallbacks,
    );
    abortRef.current = abort;
  }, [pendingFeedback, plan, sessionId, symbol, timeframe, code, streamCode, metrics]);

  const handleRetryFeedback = useCallback(() => {
    const msg = pendingFeedback;
    setDiagnosis(''); setPendingFeedback('');
    setFeedback(msg);
    setPhase('done');
  }, [pendingFeedback]);

  const busy = phase === 'generating' || phase === 'tool_call';

  return {
    phase, currentPhase, streamCode, code, prevCode, toolResults, currentTool,
    error, feedback, setFeedback, analysis, discussionReply, metrics,
    diagnosis, pendingFeedback,
    start, handleFeedback, handleConfirmDiagnosis, handleRetryFeedback,
    busy, setDiagnosis, setPendingFeedback,
  };
}
