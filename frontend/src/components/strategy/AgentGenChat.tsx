import { useState, useRef, useCallback } from 'react';
import { Space, Tag, Alert, Statistic, Row, Col, Progress } from 'antd';
import { LoadingOutlined, CheckCircleOutlined, CloseCircleOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { agentGenerateStrategyStream } from '@/client/agentGen';
import type { AgentBacktestResult, StrategyPlan } from '@/gen/ant/v1/agent_gateway_pb';
import type { StrategyProfile } from '@/gen/ant/v1/agent_profile_pb';
import type { BacktestAnalysis } from '@/gen/ant/v1/agent_analysis_pb';
import PlanCard from './PlanCard';
import ChatHistory, { type ChatTurn } from './ChatHistory';
import ChatInput from './ChatInput';
import {
  GENERATING_KEY,
  DONE_KEY,
  RESET_KEY,
} from '@/gen/ant/v1/i18n/strategy_gen_keys';

interface Props {
  symbol?: string;
  timeframe?: string;
  onApply: (pythonCode: string) => void;
}

type Phase = 'idle' | 'planning' | 'generating' | 'compiling' | 'backtesting' | 'analyzing' | 'done';

export default function AgentGenChat({ symbol, timeframe, onApply }: Props) {
  const { t } = useTranslation();
  const [phase, setPhase] = useState<Phase>('idle');
  const [userInput, setUserInput] = useState('');
  const [streamText, setStreamText] = useState('');
  const [pythonCode, setPythonCode] = useState('');
  const [compileError, setCompileError] = useState('');
  const [coverageScore, setCoverageScore] = useState(0);
  const [attempts, setAttempts] = useState(0);
  const [btResult, setBtResult] = useState<AgentBacktestResult | null>(null);
  const [planningProfile, setPlanningProfile] = useState<StrategyProfile | null>(null);
  const [finalProfile, setFinalProfile] = useState<StrategyProfile | null>(null);
  const [analysis, setAnalysis] = useState<BacktestAnalysis | null>(null);
  const [backtestError, setBacktestError] = useState('');
  const [error, setError] = useState('');
  const [plan, setPlan] = useState<StrategyPlan | null>(null);
  const [planRefining, setPlanRefining] = useState(false);
  const [history, setHistory] = useState<ChatTurn[]>([]);
  const abortRef = useRef<(() => void) | null>(null);
  const phaseRef = useRef<Phase>('idle');
  const lastMsgRef = useRef('');
  const streamTextRef = useRef('');
  const confirmedPlanRef = useRef<StrategyPlan | null>(null);
  const turnIdRef = useRef(0);

  const nextTurnId = () => String(++turnIdRef.current);
  const nowTime = () => new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });

  const metricsFromResult = (r: AgentBacktestResult) => [
    { label: 'Return', value: `${r.totalReturn.toFixed(1)}%`, positive: r.totalReturn >= 0 },
    { label: 'DD', value: `${r.maxDrawdown.toFixed(1)}%`, positive: r.maxDrawdown <= 0 },
    { label: 'Sharpe', value: r.sharpeRatio.toFixed(2), positive: r.sharpeRatio >= 1 },
    { label: 'Win', value: `${r.winRate.toFixed(1)}%` },
  ];

  const clearState = useCallback(() => {
    setStreamText('');
    streamTextRef.current = '';
    setPythonCode('');
    setCompileError('');
    setCoverageScore(0);
    setAttempts(0);
    setBtResult(null);
    setPlanningProfile(null);
    setFinalProfile(null);
    setAnalysis(null);
    setBacktestError('');
    setError('');
    setPlan(null);
    setPlanRefining(false);
    confirmedPlanRef.current = null;
  }, []);

  const reset = useCallback(() => {
    abortRef.current?.();
    setPhase('idle');
    phaseRef.current = 'idle';
    clearState();
  }, [clearState]);

  const makeCallbacks = useCallback(() => ({
    onPhase: (p: string) => {
      const next = p as Phase;
      setPhase(next);
      phaseRef.current = next;
    },
    onDelta: (d: string) => {
      streamTextRef.current += d;
      setStreamText((prev) => prev + d);
    },
    onPythonSource: (code: string) => {
      setPythonCode(code);
      onApply(code);
    },
    onCompileError: (err: string) => setCompileError(err),
    onBacktestError: (err: string) => setBacktestError(err),
    onCoverageScore: (score: number) => setCoverageScore(score),
    onResult: (result: AgentBacktestResult | null) => {
      setBtResult(result || null);
      if (result?.success) {
        setHistory((prev) => [...prev, {
          id: String(++turnIdRef.current),
          role: 'ai',
          message: streamTextRef.current || 'Strategy generated.',
          timestamp: nowTime(),
          metrics: metricsFromResult(result),
        }]);
      }
    },
    onProfile: (p: StrategyProfile | null) => {
      if (phaseRef.current === 'planning' || phaseRef.current === 'idle') {
        setPlanningProfile(p || null);
      } else {
        setFinalProfile(p || null);
      }
    },
    onAnalysis: (a: BacktestAnalysis | null) => setAnalysis(a || null),
    onAttempts: (n: number) => setAttempts(n),
    onError: (e: string) => setError(e),
    onPlan: (p: StrategyPlan) => {
      setPlan(p);
      setPlanRefining(false);
    },
  }), [onApply]);

  const handleSend = useCallback(() => {
    const msg = userInput.trim();
    if (!msg) return;
    setHistory((prev) => [...prev, {
      id: String(++turnIdRef.current),
      role: 'user',
      message: msg,
      timestamp: nowTime(),
    }]);
    setUserInput('');
    streamTextRef.current = '';
    clearState();
    setPhase('planning');
    phaseRef.current = 'planning';
    lastMsgRef.current = msg;

    const abort = agentGenerateStrategyStream(
      { message: msg, symbol, timeframe, planMode: 'plan' },
      makeCallbacks(),
    );
    abortRef.current = abort;
  }, [userInput, symbol, timeframe, onApply, clearState, makeCallbacks]);

  const handlePlanConfirm = useCallback(() => {
    if (!plan) return;
    const savedPlan = plan;
    clearState();
    confirmedPlanRef.current = savedPlan;
    setPlan(savedPlan);
    setPhase('generating');
    phaseRef.current = 'generating';

    const abort = agentGenerateStrategyStream(
      { message: lastMsgRef.current, symbol, timeframe, planMode: 'generate', confirmedPlan: savedPlan },
      makeCallbacks(),
    );
    abortRef.current = abort;
  }, [plan, symbol, timeframe, onApply, clearState, makeCallbacks]);

  const handlePlanRefine = useCallback((feedback: string) => {
    setPlanRefining(true);
    const abort = agentGenerateStrategyStream(
      { message: lastMsgRef.current, symbol, timeframe, planMode: 'plan', planFeedback: feedback },
      makeCallbacks(),
    );
    abortRef.current = abort;
  }, [symbol, timeframe, onApply, makeCallbacks]);

  const isBusy = phase !== 'idle' && phase !== 'done';

  const phaseLabels: Record<string, string> = {
    planning: t('strategy.gen.planning', 'Planning...'),
    generating: t(GENERATING_KEY),
    compiling: t('strategy.gen.compiling', 'Compiling...'),
    backtesting: t('strategy.gen.backtesting', 'Backtesting...'),
    analyzing: t('strategy.gen.analyzing', 'Analyzing...'),
  };

  const phaseTag = () => {
    if (phase === 'done') {
      return (compileError || backtestError) && !btResult
        ? <Tag icon={<CloseCircleOutlined />} color="error">{t('strategy.gen.failed', 'Failed')}</Tag>
        : <Tag icon={<CheckCircleOutlined />} color="success">{t(DONE_KEY)}</Tag>;
    }
    const label = phaseLabels[phase];
    return label ? <Tag icon={<LoadingOutlined />} color="processing">{label}</Tag> : null;
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%', overflow: 'hidden' }}>
      {/* ── Layer 1: PlanCard (fixed top, not scrolling) ── */}
      {plan && !pythonCode && (
        <div style={{ flexShrink: 0, padding: '12px 16px 0' }}>
          <PlanCard plan={plan} onConfirm={handlePlanConfirm} onRefine={handlePlanRefine} refining={planRefining} />
        </div>
      )}

      {/* ── Layer 2: Chat History (scrollable) ── */}
      <ChatHistory turns={history} />

      {/* ── Layer 3: Current stream output (max 100px) ── */}
      {(streamText || phase !== 'idle') && (
        <div style={{ flexShrink: 0, maxHeight: 100, overflowY: 'auto', padding: '8px 16px', fontSize: 12 }}>
          {phase !== 'idle' && phase !== 'done' && (
            <Space size={4} wrap style={{ marginBottom: 4 }}>
              {phaseTag()}
              {attempts > 1 && <Tag color="orange">{t('strategy.gen.attempts', 'Attempts')}: {attempts}/3</Tag>}
              {coverageScore > 0 && <Tag color="blue">{t('strategy.gen.coverage', 'Coverage')}: {(coverageScore * 100).toFixed(0)}%</Tag>}
            </Space>
          )}
          {compileError && (!btResult || phase === 'compiling') && (
            <Alert type="error" showIcon style={{ marginBottom: 4 }}
              message={t('strategy.gen.compileError', 'Compile Error')}
              description={<pre style={{ fontSize: 11, whiteSpace: 'pre-wrap', margin: 0 }}>{compileError}</pre>}
            />
          )}
          {backtestError && (!btResult || phase === 'backtesting') && (
            <Alert type="error" showIcon style={{ marginBottom: 4 }}
              message={t('strategy.gen.backtestError', 'Backtest Error')}
              description={<pre style={{ fontSize: 11, whiteSpace: 'pre-wrap', margin: 0 }}>{backtestError}</pre>}
            />
          )}
          {streamText && (
            <div style={{
              padding: 4, marginBottom: 4,
              background: 'var(--ant-color-bg-base)', borderRadius: 4,
              fontSize: 12, fontFamily: 'monospace', whiteSpace: 'pre-wrap',
            }}>
              {streamText}
            </div>
          )}
          {btResult?.success && (
            <Row gutter={8} style={{ marginBottom: 4 }}>
              <Col span={6}><Statistic title={t('strategy.gen.totalReturn', 'Total Return')} value={btResult.totalReturn} precision={2} suffix="%" valueStyle={{ fontSize: 14 }} /></Col>
              <Col span={6}><Statistic title={t('strategy.gen.maxDrawdown', 'Max Drawdown')} value={btResult.maxDrawdown} precision={2} suffix="%" valueStyle={{ fontSize: 14 }} /></Col>
              <Col span={6}><Statistic title={t('strategy.gen.sharpe', 'Sharpe Ratio')} value={btResult.sharpeRatio} precision={2} valueStyle={{ fontSize: 14 }} /></Col>
              <Col span={6}><Statistic title={t('strategy.gen.winRate', 'Win Rate')} value={btResult.winRate} precision={1} suffix="%" valueStyle={{ fontSize: 14 }} /></Col>
            </Row>
          )}
          {coverageScore > 0 && (
            <Progress percent={coverageScore * 100} size="small" format={(p) => `${t('strategy.gen.coverage', 'Coverage')}: ${(p || 0).toFixed(0)}%`} style={{ marginBottom: 4 }} />
          )}
          {planningProfile && (
            <Alert type="info" showIcon style={{ marginBottom: 4 }}
              message={t('strategy.gen.planningProfile', 'Strategy Profile (Planning)')}
              description={`${planningProfile.strategyType || ''} — ${planningProfile.description || ''}`}
            />
          )}
          {finalProfile && (
            <Alert type="info" showIcon style={{ marginBottom: 4 }}
              message={finalProfile.strategyType || t('strategy.gen.profile', 'Strategy Profile')}
              description={finalProfile.description || ''}
            />
          )}
          {analysis?.summary && (
            <Alert type="success" showIcon style={{ marginBottom: 4 }}
              message={t('strategy.gen.analysis', 'Backtest Analysis')}
              description={analysis.summary}
            />
          )}
          {error && (
            <Alert type="warning" showIcon closable style={{ marginBottom: 4 }}
              message={error} onClose={() => setError('')} />
          )}
          {phase !== 'idle' && !isBusy && (
            <a onClick={reset} style={{ fontSize: 12, cursor: 'pointer' }}>{t(RESET_KEY)}</a>
          )}
        </div>
      )}

      {/* ── Layer 4: Input (fixed bottom) ── */}
      <ChatInput
        value={userInput}
        onChange={setUserInput}
        onSend={handleSend}
        disabled={isBusy}
        hasResult={!!pythonCode}
      />
    </div>
  );
}
