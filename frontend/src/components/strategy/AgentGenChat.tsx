import { useState, useRef, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import { CloseOutlined } from '@ant-design/icons';
import { agentGenerateStrategyStream, type BacktestSummary } from '@/client/agentGen';
import type { AgentBacktestResult, StrategyPlan } from '@/gen/ant/v1/agent_gateway_pb';
import type { StrategyProfile } from '@/gen/ant/v1/agent_profile_pb';
import type { BacktestAnalysis } from '@/gen/ant/v1/agent_analysis_pb';
import ChatHistory, { type ChatTurn, type Phase } from './ChatHistory';
import ChatInput from './ChatInput';
import { SessionFeedback } from './SessionFeedback';
import { trackFunnelEvent, FunnelEvents } from '@/utils/analytics';
import { RETURN_LABEL_KEY as GEN_RETURN_KEY, MAX_DRAWDOWN_KEY as GEN_MAX_DRAWDOWN_KEY, SHARPE_KEY as GEN_SHARPE_KEY, WIN_RATE_KEY as GEN_WIN_RATE_KEY } from '@/gen/ant/v1/i18n/strategy_gen_keys';

interface Props {
  symbol?: string;
  timeframe?: string;
  accountId?: string;
  conversationId?: string;
  onApply: (pythonCode: string) => void;
  onDone?: () => void;
  initialTurnsRef?: React.MutableRefObject<ChatTurn[]>;
  currentCode?: string;
  lastBacktest?: BacktestSummary;
  recentBacktests?: BacktestSummary[];
}

const NO_DATA_RE = /insufficient market data|0 bars|need.*≥.*2/i;

export default function AgentGenChat({ symbol, timeframe, accountId, conversationId, onApply, onDone, initialTurnsRef, currentCode, lastBacktest, recentBacktests }: Props) {
  const { t } = useTranslation();
  const [turns, setTurns] = useState<ChatTurn[]>(initialTurnsRef?.current ?? []);
  const [userInput, setUserInput] = useState('');
  const [planRefining, setPlanRefining] = useState(false);
  const [hasCode, setHasCode] = useState(false);
  const [generating, setGenerating] = useState(false);
  const [excludedCtx, setExcludedCtx] = useState<Set<string>>(new Set());

  const toggleCtx = useCallback((key: string) => {
    setExcludedCtx(prev => {
      const next = new Set(prev);
      if (next.has(key)) next.delete(key); else next.add(key);
      return next;
    });
  }, []);

  const abortRef = useRef<(() => void) | null>(null);
  const conversationIdRef = useRef(conversationId || crypto.randomUUID());
  const turnIdRef = useRef(0);
  const currentTurnIdRef = useRef<string | null>(null);
  const lastMsgRef = useRef('');
  const streamTextRef = useRef('');
  const reasoningRef = useRef('');

  const nextTurnId = () => String(++turnIdRef.current);
  const nowTime = () => new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });

  const metricsFromResult = (r: AgentBacktestResult) => {
    const [ret, dd, sharpe, win] = [Number(r.totalReturn), Number(r.maxDrawdown), Number(r.sharpeRatio), Number(r.winRate)];
    return [
      { label: t(GEN_RETURN_KEY), value: `${ret.toFixed(1)}%`, positive: ret >= 0 },
      { label: t(GEN_MAX_DRAWDOWN_KEY), value: `${dd.toFixed(1)}%`, positive: dd <= 0 },
      { label: t(GEN_SHARPE_KEY), value: sharpe.toFixed(2), positive: sharpe >= 1 },
      { label: t(GEN_WIN_RATE_KEY), value: `${win.toFixed(1)}%` },
    ];
  };

  const updateCurrentTurn = useCallback((patch: Partial<ChatTurn>) => {
    const id = currentTurnIdRef.current;
    if (!id) return;
    setTurns((prev) => prev.map((t) => (t.id === id ? { ...t, ...patch } : t)));
  }, []);

  const makeCallbacks = useCallback(() => ({
    onPhase: (p: string) => {
      const next = p as Phase;
      updateCurrentTurn({ phase: next });
    },
    onDelta: (d: string) => {
      streamTextRef.current += d;
      updateCurrentTurn({ streamText: streamTextRef.current });
    },
    onReasoning: (r: string) => {
      reasoningRef.current += r;
      updateCurrentTurn({ reasoning: reasoningRef.current });
    },
    onPythonSource: (code: string) => {
      setHasCode(true);
      updateCurrentTurn({ hasCode: true, generatedCode: code });
    },
    onCompileError: (err: string) => updateCurrentTurn({ compileError: err }),
    onBacktestError: (err: string) => {
      updateCurrentTurn({ backtestError: err });
      if (NO_DATA_RE.test(err)) {
        updateCurrentTurn({ phase: 'done' });
      }
    },
    onCoverageScore: (score: number) => updateCurrentTurn({ coverageScore: score }),
    onResult: (result: AgentBacktestResult | undefined) => {
      if (result?.success) {
        updateCurrentTurn({
          metrics: metricsFromResult(result),
          phase: 'done',
        });
      }
    },
    onProfile: (p: StrategyProfile | undefined) => {
      updateCurrentTurn({ profile: p || undefined });
    },
    onAnalysis: (a: BacktestAnalysis | undefined) => updateCurrentTurn({ analysis: a || undefined }),
    onAttempts: (n: number) => updateCurrentTurn({ attempts: n }),
    onError: (e: string) => {
      setGenerating(false);
      updateCurrentTurn({ error: e });
      if (NO_DATA_RE.test(e)) {
        updateCurrentTurn({ phase: 'done' });
      }
    },
    onPlan: (p: StrategyPlan) => {
      updateCurrentTurn({ plan: p });
      setPlanRefining(false);
    },
    onDone: () => { setGenerating(false); onDone?.(); },
  // eslint-disable-next-line react-hooks/exhaustive-deps -- metricsFromResult is not memoized  | REF: rd.md#part-0.2-hooks-deps
  }), [updateCurrentTurn, onDone]);

  const startStream = useCallback((input: {
    message: string; symbol?: string; timeframe?: string;
    planMode?: string; planFeedback?: string; confirmedPlan?: StrategyPlan;
    lastBacktest?: BacktestSummary; recentBacktests?: BacktestSummary[];
  }) => {
    streamTextRef.current = '';
    reasoningRef.current = '';
    setGenerating(true);

    const aiTurn: ChatTurn = {
      id: nextTurnId(),
      role: 'ai',
      message: '',
      timestamp: nowTime(),
      phase: 'planning',
    };
    currentTurnIdRef.current = aiTurn.id;
    setTurns((prev) => [...prev, aiTurn]);

    const effectiveCode = excludedCtx.has('code') ? undefined : currentCode;
    const abort = agentGenerateStrategyStream({ ...input, conversationId: conversationIdRef.current, accountId, currentCode: effectiveCode }, makeCallbacks());
    abortRef.current = abort;
  }, [makeCallbacks, accountId, currentCode, excludedCtx]);

  const handleSend = useCallback(() => {
    const msg = userInput.trim();
    if (!msg) return;

    trackFunnelEvent(FunnelEvents.FIRST_GENERATION);
    abortRef.current?.();
    setPlanRefining(false);

    setTurns((prev) => [...prev, {
      id: nextTurnId(),
      role: 'user',
      message: msg,
      timestamp: nowTime(),
    }]);
    setUserInput('');
    lastMsgRef.current = msg;

    const effectiveLastBt = excludedCtx.has('backtest') ? undefined : lastBacktest;
    const effectiveRecentBts = excludedCtx.has('backtest') ? undefined : recentBacktests;
    startStream({ message: msg, symbol, timeframe, planMode: 'plan', lastBacktest: effectiveLastBt, recentBacktests: effectiveRecentBts });
  }, [userInput, symbol, timeframe, startStream, lastBacktest, recentBacktests, excludedCtx]);

  const handlePlanConfirm = useCallback(() => {
    const planTurn = turns.find((t) => t.plan);
    if (!planTurn?.plan) return;
    const savedPlan = planTurn.plan;
    lastMsgRef.current = lastMsgRef.current || '';

    startStream({
      message: lastMsgRef.current, symbol, timeframe,
      planMode: 'generate', confirmedPlan: savedPlan,
    });
  }, [turns, symbol, timeframe, startStream]);

  const handlePlanRefine = useCallback((feedback: string) => {
    setPlanRefining(true);
    startStream({
      message: lastMsgRef.current, symbol, timeframe,
      planMode: 'plan', planFeedback: feedback,
    });
  }, [symbol, timeframe, startStream]);

  const activePlanId = turns.find((t) => t.plan)?.id;

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%', overflow: 'hidden' }}>
      {/* ── Single scrollable message stream ── */}
      <div style={{ flex: '1 1 0', minHeight: 0, overflowY: 'auto' }}>
        <ChatHistory
          turns={turns}
          onPlanConfirm={handlePlanConfirm}
          onPlanRefine={handlePlanRefine}
          planRefining={planRefining}
          activePlanId={activePlanId}
          onApplyCode={onApply}
        />
      </div>

      {/* ── Context indicator with toggles ── */}
      <div style={{
        display: 'flex', gap: 6, padding: '4px 14px', flexShrink: 0,
        borderTop: '1px solid var(--ant-color-border)', background: 'var(--ant-color-fill-quaternary)',
        fontSize: 11, color: 'var(--ant-color-text-secondary)', flexWrap: 'wrap',
        minHeight: 28, alignItems: 'center',
      }}>
        {currentCode && (
          <span style={{
            display: 'inline-flex', alignItems: 'center', gap: 4, padding: '1px 8px', borderRadius: 4,
            background: excludedCtx.has('code') ? 'var(--ant-color-fill-tertiary)' : 'rgba(88, 166, 255, 0.1)',
            color: excludedCtx.has('code') ? 'var(--ant-color-text-tertiary)' : 'var(--color-info)',
            textDecoration: excludedCtx.has('code') ? 'line-through' : 'none',
            cursor: 'pointer', userSelect: 'none',
          }} onClick={() => toggleCtx('code')}>
            📝 {t('strategy.aiChat.codeLoaded', { defaultValue: 'Strategy code in context' })}
            {excludedCtx.has('code') && <span style={{ marginLeft: 2, fontSize: 10 }}>↻</span>}
            {!excludedCtx.has('code') && <CloseOutlined style={{ fontSize: 10, marginLeft: 2 }} />}
          </span>
        )}
        {lastBacktest && lastBacktest.totalTrades != null && (
          <span style={{
            display: 'inline-flex', alignItems: 'center', gap: 4, padding: '1px 8px', borderRadius: 4,
            background: excludedCtx.has('backtest') ? 'var(--ant-color-fill-tertiary)' : 'rgba(63, 185, 80, 0.1)',
            color: excludedCtx.has('backtest') ? 'var(--ant-color-text-tertiary)' : 'var(--color-success)',
            textDecoration: excludedCtx.has('backtest') ? 'line-through' : 'none',
            cursor: 'pointer', userSelect: 'none',
          }} onClick={() => toggleCtx('backtest')}>
            📊 {(lastBacktest.totalReturn ?? 0) >= 0 ? '+' : ''}{lastBacktest.totalReturn?.toFixed(1)}% · {lastBacktest.totalTrades} trades
            {excludedCtx.has('backtest') && <span style={{ marginLeft: 2, fontSize: 10 }}>↻</span>}
            {!excludedCtx.has('backtest') && <CloseOutlined style={{ fontSize: 10, marginLeft: 2 }} />}
          </span>
        )}
        {!currentCode && !lastBacktest && (
          <span>{t('strategy.aiChat.noContext', { defaultValue: 'No strategy loaded — describe what you want' })}</span>
        )}
      </div>

      {/* ── Input (always enabled) ── */}
      <ChatInput
        value={userInput}
        onChange={setUserInput}
        onSend={handleSend}
        disabled={generating}
        hasResult={hasCode}
      />

      {/* ── Session feedback (visible after first turn) ── */}
      {turns.length > 0 && (
        <SessionFeedback sessionId={conversationIdRef.current} />
      )}
    </div>
  );
}
