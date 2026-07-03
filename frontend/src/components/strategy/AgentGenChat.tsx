import { useState, useRef, useCallback } from 'react';
import { Input, Button, Space, Tag, Typography, Alert, Statistic, Row, Col, Progress } from 'antd';
import { ThunderboltOutlined, SendOutlined, LoadingOutlined, CheckCircleOutlined, CloseCircleOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { agentGenerateStrategyStream } from '@/client/agentGen';
import type { AgentBacktestResult, StrategyPlan } from '@/gen/ant/v1/agent_gateway_pb';
import type { StrategyProfile } from '@/gen/ant/v1/agent_profile_pb';
import type { BacktestAnalysis } from '@/gen/ant/v1/agent_analysis_pb';
import PlanCard from './PlanCard';
import {
  GENERATING_KEY,
  DONE_KEY,
  RESET_KEY,
  PLACEHOLDER_KEY,
  REGENERATE_KEY,
  SEND_KEY,
  TITLE_KEY,
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
  const abortRef = useRef<(() => void) | null>(null);
  const phaseRef = useRef<Phase>('idle');
  const lastMsgRef = useRef('');
  const confirmedPlanRef = useRef<StrategyPlan | null>(null);

  const clearState = useCallback(() => {
    setStreamText('');
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

  const handleSend = useCallback(() => {
    const msg = userInput.trim();
    if (!msg) return;
    setUserInput('');
    clearState();
    setPhase('planning');
    phaseRef.current = 'planning';
    lastMsgRef.current = msg;

    const abort = agentGenerateStrategyStream(
      {
        message: msg,
        symbol,
        timeframe,
        planMode: 'plan',
      },
      makeCallbacks(),
    );
    abortRef.current = abort;
  }, [userInput, symbol, timeframe, onApply]);

  const handlePlanConfirm = useCallback(() => {
    if (!plan) return;
    confirmedPlanRef.current = plan;
    clearState();
    setPlan(plan);
    setPhase('generating');
    phaseRef.current = 'generating';

    const abort = agentGenerateStrategyStream(
      {
        message: lastMsgRef.current,
        symbol,
        timeframe,
        planMode: 'generate',
        confirmedPlan: plan,
      },
      makeCallbacks(),
    );
    abortRef.current = abort;
  }, [plan, symbol, timeframe, onApply]);

  const handlePlanRefine = useCallback((feedback: string) => {
    setPlanRefining(true);
    const abort = agentGenerateStrategyStream(
      {
        message: lastMsgRef.current,
        symbol,
        timeframe,
        planMode: 'plan',
        planFeedback: feedback,
      },
      makeCallbacks(),
    );
    abortRef.current = abort;
  }, [symbol, timeframe, onApply]);

  const makeCallbacks = useCallback(() => ({
    onPhase: (p: string) => {
      const next = p as Phase;
      setPhase(next);
      phaseRef.current = next;
    },
    onDelta: (d: string) => setStreamText((prev) => prev + d),
    onPythonSource: (code: string) => {
      setPythonCode(code);
      onApply(code);
    },
    onCompileError: (err: string) => setCompileError(err),
    onBacktestError: (err: string) => setBacktestError(err),
    onCoverageScore: (score: number) => setCoverageScore(score),
    onResult: (result: AgentBacktestResult | null) => setBtResult(result || null),
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
    <div style={{ padding: 10, background: '#fafafa', borderRadius: 6, border: '1px solid #e8e8e8' }}>
      {/* Header */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginBottom: 8 }}>
        <ThunderboltOutlined style={{ color: '#faad14' }} />
        <Typography.Text strong style={{ fontSize: 13 }}>
          {t(TITLE_KEY, 'AI Strategy Generation')}
        </Typography.Text>
        {phase !== 'idle' && !isBusy && (
          <Button size="small" type="link" onClick={reset} style={{ marginLeft: 'auto' }}>
            {t(RESET_KEY)}
          </Button>
        )}
      </div>

      {/* Status tags */}
      {phase !== 'idle' && (
        <Space size={4} wrap style={{ marginBottom: 8 }}>
          {phaseTag()}
          {attempts > 1 && <Tag color="orange">{t('strategy.gen.attempts', 'Attempts')}: {attempts}/3</Tag>}
          {coverageScore > 0 && <Tag color="blue">{t('strategy.gen.coverage', 'Coverage')}: {(coverageScore * 100).toFixed(0)}%</Tag>}
        </Space>
      )}

      {/* Compile error */}
      {compileError && (!btResult || phase === 'compiling') && (
        <Alert type="error" showIcon style={{ marginBottom: 8 }}
          message={t('strategy.gen.compileError', 'Compile Error')}
          description={<pre style={{ fontSize: 11, whiteSpace: 'pre-wrap', margin: 0 }}>{compileError}</pre>}
        />
      )}

      {/* Backtest error */}
      {backtestError && (!btResult || phase === 'backtesting') && (
        <Alert type="error" showIcon style={{ marginBottom: 8 }}
          message={t('strategy.gen.backtestError', 'Backtest Error')}
          description={<pre style={{ fontSize: 11, whiteSpace: 'pre-wrap', margin: 0 }}>{backtestError}</pre>}
        />
      )}

      {/* Streamed output */}
      {streamText && (
        <div style={{
          maxHeight: 200, overflow: 'auto', padding: 8, marginBottom: 8,
          background: '#fff', borderRadius: 4, border: '1px solid #f0f0f0',
          fontSize: 12, fontFamily: 'monospace', whiteSpace: 'pre-wrap',
        }}>
          {streamText}
        </div>
      )}

      {/* Generated Python code */}
      {pythonCode && (
        <div style={{
          maxHeight: 250, overflow: 'auto', padding: 8, marginBottom: 8,
          background: '#f6f8fa', borderRadius: 4, border: '1px solid #d0d7de',
          fontSize: 12, fontFamily: 'monospace', whiteSpace: 'pre-wrap',
        }}>
          <Typography.Text type="secondary" style={{ fontSize: 11, display: 'block', marginBottom: 4 }}>
            {t('strategy.gen.pythonSource', 'Generated Python Strategy Code')}
          </Typography.Text>
          {pythonCode}
        </div>
      )}

      {/* Backtest metrics */}
      {btResult?.success && (
        <Row gutter={8} style={{ marginBottom: 8 }}>
          <Col span={6}>
            <Statistic title={t('strategy.gen.totalReturn', 'Total Return')} value={btResult.totalReturn} precision={2} suffix="%" valueStyle={{ fontSize: 14 }} />
          </Col>
          <Col span={6}>
            <Statistic title={t('strategy.gen.maxDrawdown', 'Max Drawdown')} value={btResult.maxDrawdown} precision={2} suffix="%" valueStyle={{ fontSize: 14 }} />
          </Col>
          <Col span={6}>
            <Statistic title={t('strategy.gen.sharpe', 'Sharpe Ratio')} value={btResult.sharpeRatio} precision={2} valueStyle={{ fontSize: 14 }} />
          </Col>
          <Col span={6}>
            <Statistic title={t('strategy.gen.winRate', 'Win Rate')} value={btResult.winRate} precision={1} suffix="%" valueStyle={{ fontSize: 14 }} />
          </Col>
        </Row>
      )}

      {/* Coverage progress */}
      {coverageScore > 0 && (
        <Progress
          percent={coverageScore * 100}
          size="small"
          format={(p) => `${t('strategy.gen.coverage', 'Coverage')}: ${(p || 0).toFixed(0)}%`}
          style={{ marginBottom: 8 }}
        />
      )}

      {/* Planning profile (from NL) */}
      {planningProfile && (
        <Alert type="info" showIcon style={{ marginBottom: 8 }}
          message={t('strategy.gen.planningProfile', 'Strategy Profile (Planning)')}
          description={`${planningProfile.strategyType || ''} — ${planningProfile.description || ''}`}
        />
      )}

      {/* Final profile (from source+coverage) */}
      {finalProfile && (
        <Alert type="info" showIcon style={{ marginBottom: 8 }}
          message={finalProfile.strategyType || t('strategy.gen.profile', 'Strategy Profile')}
          description={finalProfile.description || ''}
        />
      )}

      {/* Analysis */}
      {analysis?.summary && (
        <Alert type="success" showIcon style={{ marginBottom: 8 }}
          message={t('strategy.gen.analysis', 'Backtest Analysis')}
          description={analysis.summary}
        />
      )}

      {/* Error */}
      {error && (
        <Alert type="warning" showIcon closable style={{ marginBottom: 8 }}
          message={error} onClose={() => setError('')}
        />
      )}

      {/* Plan Mode card (ADR-0025 §3) */}
      {plan && phase === 'done' && !pythonCode && (
        <PlanCard
          plan={plan}
          onConfirm={handlePlanConfirm}
          onRefine={handlePlanRefine}
          refining={planRefining}
        />
      )}

      {/* Input area */}
      {!isBusy && (
        <>
          <Input.TextArea
            rows={3}
            value={userInput}
            onChange={(e) => setUserInput(e.target.value)}
            onPressEnter={(e) => { e.preventDefault(); handleSend(); }}
            placeholder={t(PLACEHOLDER_KEY)}
            style={{ fontSize: 13, marginBottom: 8 }}
          />
          <Button type="primary" icon={<SendOutlined />} size="small"
            onClick={handleSend} disabled={!userInput.trim()} block>
            {pythonCode ? t(REGENERATE_KEY) : t(SEND_KEY)}
          </Button>
        </>
      )}
    </div>
  );
}
