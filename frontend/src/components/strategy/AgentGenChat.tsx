import { useState, useRef, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import { agentGenerateStrategyStream } from '@/client/agentGen';
import type { AgentBacktestResult, StrategyPlan } from '@/gen/ant/v1/agent_gateway_pb';
import type { StrategyProfile } from '@/gen/ant/v1/agent_profile_pb';
import type { BacktestAnalysis } from '@/gen/ant/v1/agent_analysis_pb';
import ChatHistory, { type ChatTurn, type Phase } from './ChatHistory';
import ChatInput from './ChatInput';

interface Props {
  symbol?: string;
  timeframe?: string;
  conversationId?: string;
  onApply: (pythonCode: string) => void;
  onDone?: () => void;
  initialTurnsRef?: React.MutableRefObject<ChatTurn[]>;
}

const NO_DATA_RE = /insufficient market data|0 bars|need.*≥.*2/i;

export default function AgentGenChat({ symbol, timeframe, conversationId, onApply, onDone, initialTurnsRef }: Props) {
  const { t } = useTranslation();
  const [turns, setTurns] = useState<ChatTurn[]>(initialTurnsRef?.current ?? []);
  const [userInput, setUserInput] = useState('');
  const [planRefining, setPlanRefining] = useState(false);
  const [hasCode, setHasCode] = useState(false);
  const [generating, setGenerating] = useState(false);

  const abortRef = useRef<(() => void) | null>(null);
  const conversationIdRef = useRef(conversationId || crypto.randomUUID());
  const turnIdRef = useRef(0);
  const currentTurnIdRef = useRef<string | null>(null);
  const lastMsgRef = useRef('');
  const confirmedPlanRef = useRef<StrategyPlan | null>(null);
  const streamTextRef = useRef('');
  const reasoningRef = useRef('');

  const nextTurnId = () => String(++turnIdRef.current);
  const nowTime = () => new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });

  const metricsFromResult = (r: AgentBacktestResult) => [
    { label: t('strategy.gen.return', 'Return'), value: `${r.totalReturn.toFixed(1)}%`, positive: r.totalReturn >= 0 },
    { label: t('strategy.gen.maxDrawdown', 'Max DD'), value: `${r.maxDrawdown.toFixed(1)}%`, positive: r.maxDrawdown <= 0 },
    { label: t('strategy.gen.sharpe', 'Sharpe'), value: r.sharpeRatio.toFixed(2), positive: r.sharpeRatio >= 1 },
    { label: t('strategy.gen.winRate', 'Win'), value: `${r.winRate.toFixed(1)}%` },
  ];

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
      updateCurrentTurn({ error: e });
      if (NO_DATA_RE.test(e)) {
        updateCurrentTurn({ phase: 'done' });
      }
    },
    onPlan: (p: StrategyPlan) => {
      updateCurrentTurn({ plan: p });
      setPlanRefining(false);
    },
    onDone: onDone || (() => {}),
  }), [onApply, updateCurrentTurn, onDone]);

  const startStream = useCallback((input: {
    message: string; symbol?: string; timeframe?: string;
    planMode?: string; planFeedback?: string; confirmedPlan?: StrategyPlan;
  }) => {
    streamTextRef.current = '';
    reasoningRef.current = '';

    const aiTurn: ChatTurn = {
      id: nextTurnId(),
      role: 'ai',
      message: '',
      timestamp: nowTime(),
      phase: 'planning',
    };
    currentTurnIdRef.current = aiTurn.id;
    setTurns((prev) => [...prev, aiTurn]);

    const abort = agentGenerateStrategyStream({ ...input, conversationId: conversationIdRef.current }, makeCallbacks());
    abortRef.current = abort;
  }, [makeCallbacks]);

  const handleSend = useCallback(() => {
    const msg = userInput.trim();
    if (!msg) return;

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

    startStream({ message: msg, symbol, timeframe, planMode: 'plan' });
  }, [userInput, symbol, timeframe, startStream]);

  const handlePlanConfirm = useCallback(() => {
    const planTurn = turns.find((t) => t.plan);
    if (!planTurn?.plan) return;
    const savedPlan = planTurn.plan;
    confirmedPlanRef.current = savedPlan;
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

      {/* ── Input (always enabled) ── */}
      <ChatInput
        value={userInput}
        onChange={setUserInput}
        onSend={handleSend}
        disabled={generating}
        hasResult={hasCode}
      />
    </div>
  );
}
