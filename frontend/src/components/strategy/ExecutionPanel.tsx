import { useState, useRef, useCallback } from 'react';
import { Button, Input, Tag, Typography, Card, Space } from 'antd';
import { CodeOutlined, ThunderboltOutlined, CheckCircleOutlined, CloseCircleOutlined, EditOutlined, LoadingOutlined, SendOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next'
import {
  EXEC_TITLE_KEY, EXEC_RUNNING_KEY, EXEC_DONE_KEY, EXEC_BACK_TO_PLAN_KEY,
  EXEC_PLAN_LABEL_KEY, EXEC_COMPLIANCE_TOOL_KEY, EXEC_BACKTEST_TOOL_KEY,
  EXEC_TOOL_RUNNING_KEY, EXEC_FEEDBACK_TITLE_KEY, EXEC_FEEDBACK_HINT_KEY,
  EXEC_FEEDBACK_PLACEHOLDER_KEY, EXEC_CHIP_LOWER_DD_KEY, EXEC_CHIP_RAISE_RETURN_KEY,
  EXEC_CHIP_TIGHTEN_SL_KEY, EXEC_CHIP_LONG_ONLY_KEY, EXEC_SEND_FEEDBACK_KEY,
  EXEC_CLEAR_KEY, EXEC_APPLY_CODE_KEY,
  PLACEHOLDER_KEY, FEEDBACK_HEADING_KEY,
} from '@/gen/ant/v1/i18n/strategy_gen_keys';
import StepProgress from './StepProgress';
import DiffView from './DiffView';
import { executePlan, diagnosePlan, type ExecuteCallbacks, type PlanCallbacks } from '@/client/strategyPlan';
import { pythonStrategyApi } from '@/client/pythonStrategy';
import { isSucceededRun } from '@/pages/strategy/StrategyTemplatePage.utils';
import type { ToolCall, ToolResult, BacktestMetricsMsg } from '@/gen/ant/v1/strategy_execution_pb';
import type { BacktestRunUpdate } from '@/gen/ant/v1/backtest_run_query_pb';

const { TextArea } = Input;

interface Props {
  plan: string;
  symbol?: string;
  timeframe?: string;
  sessionId?: string;
  previousCode?: string;
  onApply: (code: string, previousCode?: string) => void;
  onReset: () => void;
}

type Phase = 'idle' | 'generating' | 'tool_call' | 'tool_result' | 'done' | 'error';

export default function ExecutionPanel({ plan, symbol, timeframe, sessionId, previousCode, onApply, onReset }: Props) {
  const { t } = useTranslation();
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
  const [discussionReply, setDiscussionReply] = useState('');
  const [metrics, setMetrics] = useState<BacktestMetricsMsg | null>(null);
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
                watchRef.current = pythonStrategyApi.watchBacktestRun(out.run_id, (update: BacktestRunUpdate) => {
                  if (isSucceededRun(update.run) && update.metrics) {
                    setMetrics({
                      totalReturn: update.metrics.totalReturn, sharpeRatio: update.metrics.sharpeRatio,
                      maxDrawdown: update.metrics.maxDrawdown, winRate: update.metrics.winRate,
                      totalTrades: update.metrics.totalTrades, profitFactor: update.metrics.profitFactor,
                    });
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

  const [diagnosis, setDiagnosis] = useState('');
  const [pendingFeedback, setPendingFeedback] = useState('');

  const handleFeedback = useCallback(() => {
    const msg = feedback.trim();
    if (!msg) return;
    setFeedback('');
    setDiagnosis('');
    setPendingFeedback(msg);
    setPhase('generating'); setError('');

    // Phase 1: Diagnosis only — no code generation
    const abort = diagnosePlan(
      { plan, conversationId: sessionId, feedbackMessage: msg,
        currentCode: code || streamCode,
        backtestMetricsJson: metrics ? JSON.stringify(metrics) : '' },
      {
        onDelta: () => {}, // ignore streaming, show final
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
    setFeedback(msg); // put it back so user can edit
    setPhase('done');
  }, [pendingFeedback]);

  const busy = phase === 'generating' || phase === 'tool_call';

  const toolLabel = (name: string) => {
    switch (name) {
      case 'compliance_check': return t(EXEC_COMPLIANCE_TOOL_KEY, 'Compliance Check');
      case 'backtest': return t(EXEC_BACKTEST_TOOL_KEY, 'Backtest');
      default: return name;
    }
  };

  const toolIcon = (name: string) => {
    switch (name) {
      case 'compliance_check': return <CodeOutlined />;
      case 'backtest': return <ThunderboltOutlined />;
      default: return <CodeOutlined />;
    }
  };

  const chips = [
    { key: 'lower_dd', label: t(EXEC_CHIP_LOWER_DD_KEY, 'Lower Drawdown') },
    { key: 'raise_return', label: t(EXEC_CHIP_RAISE_RETURN_KEY, 'Raise Returns') },
    { key: 'tighten_sl', label: t(EXEC_CHIP_TIGHTEN_SL_KEY, 'Tighten Stop') },
    { key: 'long_only', label: t(EXEC_CHIP_LONG_ONLY_KEY, 'Long Only') },
  ];

  // Parse plan into numbered steps for the checklist
  const planSteps = plan.split('\n').filter(line => /^\d+[\.\)]\s/.test(line.trim()));

  const hasSymbol = !!(symbol && timeframe);

  return (
    <div style={{ padding: 10, border: '1px solid #f0f0f0', borderRadius: 6, background: '#fafafa' }}>
      {hasSymbol && (
        <Tag color="blue" style={{ fontSize: 11, marginBottom: 6 }}>📊 {symbol} · {timeframe}</Tag>
      )}
      <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginBottom: 8 }}>
        <ThunderboltOutlined style={{ color: '#faad14' }} />
        <Typography.Text strong style={{ fontSize: 13 }}>{t(EXEC_TITLE_KEY, 'AI Executing')}</Typography.Text>
        {busy && <Tag icon={<LoadingOutlined />} color="processing">{t(EXEC_RUNNING_KEY, 'Running')}</Tag>}
        {phase === 'done' && <Tag color="success">{t(EXEC_DONE_KEY, 'Done')}</Tag>}
        {phase === 'error' && <Tag color="error">{t(PLACEHOLDER_KEY, 'Error')}</Tag>}
        <Button size="small" type="link" onClick={onReset} style={{ marginLeft: 'auto' }}>{t(EXEC_BACK_TO_PLAN_KEY, 'Back to Plan')}</Button>
      </div>

      <Card size="small" style={{ marginBottom: 8, background: '#f6ffed', borderColor: '#b7eb8f' }}>
        <Typography.Text style={{ fontSize: 11, color: '#8c8c8c' }}>{t(EXEC_PLAN_LABEL_KEY, 'Execution Plan')}</Typography.Text>
        {planSteps.length > 0 ? (
          <div style={{ marginTop: 4 }}>
            {planSteps.map((step, i) => (
              <div key={i} style={{ fontSize: 12, padding: '1px 0', color: '#595959' }}>
                <CheckCircleOutlined style={{ color: '#52c41a', marginRight: 4, fontSize: 11 }} />
                {step.replace(/^\d+[\.\)]\s*/, '')}
              </div>
            ))}
          </div>
        ) : (
          <Typography.Paragraph ellipsis={{ rows: 2 }} style={{ fontSize: 12, margin: 0 }}>{plan}</Typography.Paragraph>
        )}
      </Card>

      <StepProgress phase={busy ? currentPhase : phase === 'done' ? 'done' : 'idle'} plan={undefined} />

      {discussionReply && (
        <div style={{ padding: '8px 10px', marginBottom: 6, borderRadius: 4, background: '#f5f5f5', border: '1px solid #d9d9d9', fontSize: 12, whiteSpace: 'pre-wrap', color: '#595959' }}>
          💬 {discussionReply}
        </div>
      )}

      {analysis && (
        <div style={{ padding: '8px 10px', marginBottom: 6, borderRadius: 4, background: '#e6f4ff', border: '1px solid #91caff', fontSize: 12, whiteSpace: 'pre-wrap', color: '#1677ff' }}>
          💡 {analysis}
        </div>
      )}

      {toolResults.map((tr, i) => (
        <div key={i} style={{
          padding: '6px 10px', marginBottom: 4, borderRadius: 4, fontSize: 11,
          // compliance: green/red is meaningful (pass/fail); backtest: neutral (ran ≠ good result)
          background: tr.name === 'backtest' ? '#f0f5ff' : (tr.success ? '#f6ffed' : '#fff2f0'),
          border: `1px solid ${tr.name === 'backtest' ? '#d6e4ff' : (tr.success ? '#b7eb8f' : '#ffa39e')}`,
        }}>
          <Space>
            {tr.name === 'backtest'
              ? <span style={{ color: '#2f54eb' }}>📊</span>
              : (tr.success ? <CheckCircleOutlined style={{ color: '#52c41a' }} /> : <CloseCircleOutlined style={{ color: '#ff4d4f' }} />)
            }
            <span>{toolIcon(tr.name)} <b>{toolLabel(tr.name)}</b></span>
            {tr.error && <span style={{ color: '#cf1322' }}>{tr.error}</span>}
          </Space>
        </div>
      ))}

      {metrics && (
        <div style={{ padding: '6px 10px', marginBottom: 6, borderRadius: 4, background: '#f6ffed', border: '1px solid #b7eb8f', fontSize: 11 }}>
          <b>📊 {t(FEEDBACK_HEADING_KEY, 'Backtest Results')}</b> · Sharpe {metrics.sharpeRatio?.toFixed(2)} · {(t as any)('strategy.gen.metrics.maxDrawdown', 'Max DD')}: {((metrics.maxDrawdown ?? 0) * 100).toFixed(1)}% · {(t as any)('strategy.gen.metrics.winRate', 'Win')}: {((metrics.winRate ?? 0) * 100).toFixed(0)}% · {(t as any)('strategy.gen.metrics.trades', 'Trades')}: {metrics.totalTrades}
        </div>
      )}

      {prevCode && code && <DiffView oldCode={prevCode} newCode={code} />}

      {streamCode && (
        <div style={{ maxHeight: 200, overflow: 'auto', padding: 8, marginBottom: 8,
          background: '#fff', borderRadius: 4, border: '1px solid #f0f0f0',
          fontSize: 11, fontFamily: 'monospace', whiteSpace: 'pre-wrap' }}>{streamCode}</div>
      )}

      {error && (
        <div style={{ padding: 6, marginBottom: 8, background: '#fff2f0', borderRadius: 4, fontSize: 11, color: '#cf1322' }}>{error}</div>
      )}

      {/* Diagnosis card — AI suggests, user confirms */}
      {diagnosis && phase === 'done' && (
        <div style={{ marginBottom: 8, padding: 10, background: '#fffbe6', borderRadius: 6, border: '1px solid #ffe58f' }}>
          <Typography.Text strong style={{ fontSize: 12, display: 'block', marginBottom: 4 }}>
            💡 AI 诊断与建议
          </Typography.Text>
          <div style={{ fontSize: 12, whiteSpace: 'pre-wrap', color: '#595959', marginBottom: 8 }}>
            {diagnosis}
          </div>
          <Space>
            <Button size="small" type="primary" icon={<CheckCircleOutlined />}
              onClick={handleConfirmDiagnosis}>
              按此修改
            </Button>
            <Button size="small" icon={<EditOutlined />}
              onClick={handleRetryFeedback}>
              调整反馈
            </Button>
            <Button size="small" type="text"
              onClick={() => { setDiagnosis(''); setPendingFeedback(''); }}>
              忽略
            </Button>
          </Space>
        </div>
      )}

      {phase === 'done' && code && (
        <div style={{ marginTop: 8, padding: 10, background: '#f0f5ff', borderRadius: 6, border: '1px solid #adc6ff' }}>
          <Typography.Text strong style={{ fontSize: 11, display: 'block', marginBottom: 4 }}>
            {t(EXEC_FEEDBACK_TITLE_KEY, 'Continue AI Conversation')}
          </Typography.Text>
          <Typography.Text type="secondary" style={{ fontSize: 10, display: 'block', marginBottom: 6 }}>
            {t(EXEC_FEEDBACK_HINT_KEY, 'Use natural language to guide the AI.')}
          </Typography.Text>
          <TextArea rows={3} value={feedback} onChange={e => setFeedback(e.target.value)}
            placeholder={t(EXEC_FEEDBACK_PLACEHOLDER_KEY, 'Try saying: "tighten stop to 1%"')}
            disabled={busy} style={{ fontSize: 13, marginBottom: 6 }}
          />
          <Space style={{ marginBottom: 6 }}>
            {chips.map(chip => (
              <Tag key={chip.key} color="purple" style={{ cursor: 'pointer', fontSize: 11 }}
                onClick={() => { setFeedback(chip.label); }}>
                💬 {chip.label}
              </Tag>
            ))}
          </Space>
          <div style={{ display: 'flex', gap: 8 }}>
            <Button type="primary" size="small" icon={<SendOutlined />} block
              onClick={handleFeedback} disabled={!feedback.trim()} loading={busy}>
              {t(EXEC_SEND_FEEDBACK_KEY, 'Send to AI')}
            </Button>
            <Button size="small" onClick={() => { setFeedback(''); }} disabled={!feedback.trim()}>
              {t(EXEC_CLEAR_KEY, 'Clear')}
            </Button>
          </div>
        </div>
      )}

      {phase === 'done' && code && (
        <Button type="primary" size="small" icon={<EditOutlined />} block
          onClick={() => onApply(code, prevCode)} style={{ marginTop: 8 }}>
          {t(EXEC_APPLY_CODE_KEY, 'Apply to Editor')}
        </Button>
      )}
    </div>
  );
}
