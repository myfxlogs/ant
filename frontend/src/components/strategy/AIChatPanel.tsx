import { useState, useRef, useEffect, useMemo, useCallback } from 'react';
import { Button, Input, Space, Tag, Typography, message } from 'antd';
import { ThunderboltOutlined, SendOutlined, LoadingOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { generateStrategyStream } from '@/client/strategyGen';
import { codeAssistApi, type CodeChatMessage } from '@/client/codeAssist';
import { pythonStrategyApi } from '@/client/pythonStrategy';

interface BacktestMetrics {
  totalReturn?: number; sharpeRatio?: number; maxDrawdown?: number;
  winRate?: number; totalTrades?: number; profitFactor?: number;
}

const { TextArea } = Input;

interface Props {
  code: string;
  onApply: (code: string) => void;
  symbol?: string;
  timeframe?: string;
  initialPrompt?: string | null;
  /** When false, AI responses are shown in chat but do NOT replace the code editor. Default true. */
  autoApply?: boolean;
  sessionId?: string;
  chatHistory?: CodeChatMessage[];
}

// 4-mode intent classification — keyword tables match backend classifyIntent exactly.
// Phase 3: hasBacktest forces 'generate' (feedback mode) regardless of keywords.
function detectMode(msg: string, hasCode: boolean, hasBacktest = false): 'generate' | 'revise' | 'repair' | 'discuss' {
  if (hasBacktest) return 'generate';
  if (!hasCode) return 'generate';
  const lower = msg.toLowerCase();
  // Repair (highest priority — error keywords)
  const repairKw = ['报错','error','错误','traceback','缺少参数','missing',
    '验证失败','syntax error','syntaxerror','undefined','未定义',
    '缺少 required','参数不足','attributeerror','typeerror'];
  if (repairKw.some(k => lower.includes(k))) return 'repair';
  // Discuss (question/analysis keywords)
  const discussKw = ['为什么','什么意思','怎么样','对吗','分析','解释',
    'what','why','how','explain','对不对'];
  if (discussKw.some(k => lower.includes(k))) return 'discuss';
  return 'revise';
}

function modeLabel(t: (k: string, d?: string) => string, mode: string): string {
  const map: Record<string, string> = {
    generate: 'strategy.gen.chat.generate',
    revise: 'strategy.gen.chat.revise',
    repair: 'strategy.gen.chat.repair',
    discuss: 'strategy.gen.chat.discuss',
  };
  return t(map[mode] || mode, mode);
}

const MODE_COLORS: Record<string, string> = {
  generate: 'blue', revise: 'green', repair: 'orange', discuss: 'purple',
};

export default function AIChatPanel({ code, onApply, symbol, timeframe, initialPrompt, autoApply = true, sessionId, chatHistory }: Props) {
  const { t, i18n } = useTranslation();
  const [draft, setDraft] = useState('');
  const [mode, setMode] = useState<'idle' | 'clarifying' | 'streaming' | 'done'>('idle');
  const [streamText, setStreamText] = useState('');
  const [questions, setQuestions] = useState<string[]>([]);
  const [genCode, setGenCode] = useState('');
  const [pendingCode, setPendingCode] = useState<string | null>(null);
  const [error, setError] = useState('');
  const [phase, setPhase] = useState('');
  const [backtestId, setBacktestId] = useState('');
  const [clarifyRound, setClarifyRound] = useState(0);
  const [history, setHistory] = useState<CodeChatMessage[]>(chatHistory || []);
  // Phase 3: feedback loop state
  const [backtestMetrics, setBacktestMetrics] = useState<BacktestMetrics | null>(null);
  const [analysisText, setAnalysisText] = useState('');
  const [adviceText, setAdviceText] = useState('');
  const [lastGeneratedCode, setLastGeneratedCode] = useState('');
  const abortRef = useRef<(() => void) | null>(null);
  const streamRef = useRef('');

  useEffect(() => () => abortRef.current?.(), []);

  // Auto-trigger on AI Optimize prompt.
  useEffect(() => {
    if (initialPrompt) { setDraft(initialPrompt); }
  }, [initialPrompt]);

  const reset = useCallback(() => {
    abortRef.current?.();
    setMode('idle'); setStreamText(''); setQuestions([]);
    setGenCode(''); setError(''); setPhase(''); setBacktestId('');
    setBacktestMetrics(null); setAnalysisText(''); setAdviceText('');
    setLastGeneratedCode('');
  }, []);

  const handleGenerate = useCallback((msg: string, round: number, isFeedback = false) => {
    setMode('streaming'); setStreamText(''); setError(''); setGenCode('');
    setAnalysisText(''); setAdviceText('');
    const input: Parameters<typeof generateStrategyStream>[0] = {
      message: msg, symbol, timeframe, clarificationRound: round,
      conversationId: sessionId || '',
    };
    if (isFeedback && backtestMetrics) {
      input.previousCode = lastGeneratedCode || code;
      input.backtestMetricsJson = JSON.stringify(backtestMetrics);
      input.feedbackMessage = msg;
    }
    const abort = generateStrategyStream(
      input,
      {
        onPhase: (p) => {
          if (p === 'analyzing') setPhase('analyzing');
          else if (p === 'clarifying') setMode('clarifying');
          else if (p === 'generating') setMode('streaming');
          else setPhase(p);
        },
        onDelta: (d) => { setStreamText(p => p + d); streamRef.current += d; },
        onQuestions: (q) => setQuestions(q),
        onTemplate: (n) => setPhase('template: ' + n),
        onCode: (c) => {
          setGenCode(c);
          setLastGeneratedCode(c);
          if (autoApply) onApply(c);
          else setPendingCode(c);
        },
        onBacktestId: (id) => {
          setBacktestId(id);
          // Watch for backtest completion to get metrics for next feedback round
          pythonStrategyApi.watchBacktestRun(id, (update: any) => {
            if ((update.status === 'SUCCEEDED' || update.status === 'COMPLETED') && update.metrics) {
              setBacktestMetrics({
                sharpeRatio: update.metrics.sharpeRatio,
                maxDrawdown: update.metrics.maxDrawdown,
                winRate: update.metrics.winRate,
                totalTrades: update.metrics.totalTrades,
                totalReturn: update.metrics.totalReturn,
                profitFactor: update.metrics.profitFactor,
              });
            }
          });
        },
        onAnalysis: (a) => setAnalysisText(a),
        onAdvice: (a) => setAdviceText(a),
        onError: (e) => setError(e),
        onDone: () => setMode('done'),
      },
    );
    abortRef.current = abort;
  }, [symbol, timeframe, sessionId, onApply, backtestMetrics, lastGeneratedCode, code, autoApply]);

  const handleRevise = useCallback((msg: string) => {
    setMode('streaming'); setStreamText(''); setError('');
    streamRef.current = '';
    const abort = codeAssistApi.reviseStream(
      { code, instruction: msg, history, locale: i18n.language, sessionId },
      {
        onDelta: (d) => { setStreamText(p => p + d); streamRef.current += d; },
        onResult: (python) => {
          setMode('done');
          setHistory(prev => [...prev,
            { role: 'user', content: msg },
            { role: 'assistant', content: streamRef.current || python },
          ]);
          streamRef.current = ''; setStreamText('');
          if (python) {
            if (autoApply) {
              onApply(python);
              message.success(t('strategy.codeAssist.codeUpdated', 'Code updated.'));
            } else {
              setPendingCode(python);
            }
          }
        },
        onError: (e) => {
          setMode('done'); setError(String((e as Error)?.message || e));
        },
      },
    );
    abortRef.current = abort;
  }, [code, history, i18n.language, sessionId, onApply, t]);

  const handleSend = useCallback(() => {
    const msg = draft.trim();
    if (!msg) return;
    setDraft(''); setClarifyRound(0); setQuestions([]);
    const hasBacktest = !!backtestMetrics;
    const intent = detectMode(msg, !!code.trim(), hasBacktest);
    if (intent === 'generate') handleGenerate(msg, 0, hasBacktest);
    else handleRevise(msg);
  }, [draft, code, handleGenerate, handleRevise, backtestMetrics]);

  const handleClarifyAnswer = useCallback((answer: string) => {
    const next = clarifyRound + 1;
    setClarifyRound(next); setQuestions([]);
    handleGenerate(answer, next);
  }, [clarifyRound, handleGenerate]);

  const isBusy = mode === 'streaming';

  // Message history view
  const messagesView = useMemo(() => history.map((m, i) => (
    <div key={i} style={{ margin: '6px 0', padding: '6px 10px', borderRadius: 6,
      background: m.role === 'user' ? '#e6f4ff' : '#f6ffed', fontSize: 12, whiteSpace: 'pre-wrap' }}>
      <b style={{ color: m.role === 'user' ? '#1677ff' : '#389e0d' }}>
        {m.role === 'user' ? 'You' : 'AI'}
      </b>
      <div>{m.content}</div>
    </div>
  )), [history]);

  // Clarification mode
  if (questions.length > 0) {
    return (
      <div style={{ padding: 12, background: '#fffbe6', borderRadius: 6, border: '1px solid #ffe58f' }}>
        <Typography.Text strong style={{ fontSize: 13 }}>
          {t('strategy.gen.clarifyTitle', '需要确认几个细节：')}
        </Typography.Text>
        <Space direction="vertical" size={6} style={{ width: '100%', marginTop: 8 }}>
          {questions.map((q, i) => (
            <Button key={i} block size="small" type="dashed" onClick={() => handleClarifyAnswer(q)}
              disabled={clarifyRound >= 3}>{q}</Button>
          ))}
          {clarifyRound >= 3 && (
            <Button block size="small" type="primary" onClick={() => { setQuestions([]); handleGenerate('使用默认参数', clarifyRound); }}>
              {t('strategy.gen.useDefaults', '使用默认设置继续')}
            </Button>
          )}
        </Space>
      </div>
    );
  }

  return (
    <div style={{ border: '1px solid #f0f0f0', borderRadius: 6, padding: 10, background: '#fafafa' }}>
      {/* Header */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginBottom: 8 }}>
        <ThunderboltOutlined style={{ color: '#faad14' }} />
        <Typography.Text strong style={{ fontSize: 13 }}>AI Chat</Typography.Text>
        <Space size={4} wrap style={{ marginLeft: 8 }}>
          {!code.trim() && !backtestMetrics && <Tag color="blue">{t('strategy.gen.title', '策略生成')}</Tag>}
          {!!code.trim() && !backtestMetrics && <Tag color="green">revise</Tag>}
          {backtestMetrics && <Tag color="purple">🔄 feedback</Tag>}
          {isBusy && <Tag icon={<LoadingOutlined />} color="processing">streaming</Tag>}
          {phase === 'analyzing' && isBusy && <Tag color="orange">analyzing</Tag>}
          {phase && !isBusy && phase !== 'done' && <Tag>{phase}</Tag>}
          {backtestId && <Tag color="success">backtest: {backtestId.slice(0, 8)}</Tag>}
        </Space>
        {mode !== 'idle' && !isBusy && (
          <Button size="small" type="link" onClick={reset} style={{ marginLeft: 'auto' }}>reset</Button>
        )}
      </div>

      {/* Messages + streaming */}
      <div style={{ maxHeight: 200, overflow: 'auto', marginBottom: 6 }}>
        {messagesView}
        {analysisText && (
          <div style={{ margin: '6px 0', padding: '8px 10px', borderRadius: 6,
            background: '#f5f5f5', fontSize: 12, color: '#595959' }}>
            🔍 {analysisText}
          </div>
        )}
        {adviceText && (
          <div style={{ margin: '6px 0', padding: '8px 10px', borderRadius: 6,
            background: '#e6f4ff', fontSize: 12, color: '#1677ff' }}>
            💡 {adviceText}
          </div>
        )}
        {streamText && (
          <div style={{ margin: '6px 0', padding: '6px 10px', borderRadius: 6,
            background: '#f6ffed', fontSize: 12, whiteSpace: 'pre-wrap', fontFamily: 'monospace' }}>
            <b style={{ color: '#389e0d' }}>AI</b>
            <div>{streamText}</div>
          </div>
        )}
        {backtestMetrics && mode === 'done' && (
          <div style={{ margin: '6px 0', padding: '10px', borderRadius: 6,
            background: '#f6ffed', border: '1px solid #b7eb8f' }}>
            <Typography.Text strong style={{ fontSize: 11, marginBottom: 4, display: 'block' }}>
              {t('strategy.gen.feedback.heading')}
            </Typography.Text>
            <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap', marginBottom: 4 }}>
              {backtestMetrics.sharpeRatio != null && (
                <span style={{ fontSize: 10, background: '#fff', padding: '1px 5px', borderRadius: 3, border: '1px solid #e8e8e8' }}>
                  <b>Sharpe</b> {backtestMetrics.sharpeRatio.toFixed(2)}
                </span>
              )}
              {backtestMetrics.maxDrawdown != null && (
                <span style={{ fontSize: 10, background: '#fff', padding: '1px 5px', borderRadius: 3, border: '1px solid #e8e8e8',
                  color: backtestMetrics.maxDrawdown > 0.2 ? '#cf1322' : '#595959' }}>
                  <b>Max DD</b> {(backtestMetrics.maxDrawdown * 100).toFixed(1)}%
                </span>
              )}
              {backtestMetrics.winRate != null && (
                <span style={{ fontSize: 10, background: '#fff', padding: '1px 5px', borderRadius: 3, border: '1px solid #e8e8e8' }}>
                  <b>Win</b> {(backtestMetrics.winRate * 100).toFixed(0)}%
                </span>
              )}
              {backtestMetrics.totalTrades != null && (
                <span style={{ fontSize: 10, background: '#fff', padding: '1px 5px', borderRadius: 3, border: '1px solid #e8e8e8' }}>
                  <b>Trades</b> {backtestMetrics.totalTrades}
                </span>
              )}
              {backtestMetrics.totalReturn != null && (
                <span style={{ fontSize: 10, background: '#fff', padding: '1px 5px', borderRadius: 3, border: '1px solid #e8e8e8',
                  color: backtestMetrics.totalReturn > 0 ? '#389e0d' : '#cf1322' }}>
                  <b>Return</b> {(backtestMetrics.totalReturn * 100).toFixed(1)}%
                </span>
              )}
            </div>
            <Typography.Text type="secondary" style={{ fontSize: 9 }}>
              {t('strategy.gen.feedback.placeholder')}
            </Typography.Text>
          </div>
        )}
      </div>

      {/* Error */}
      {error && (
        <div style={{ padding: 4, marginBottom: 6, background: '#fff2f0', borderRadius: 4,
          fontSize: 11, color: '#cf1322' }}>{error}</div>
      )}

      {/* Pending code — shown when autoApply=false, AI returned code but it was not applied */}
      {pendingCode && !autoApply && (
        <div style={{ padding: 8, marginBottom: 8, background: '#f6ffed', borderRadius: 6,
          border: '1px solid #b7eb8f', display: 'flex', alignItems: 'center', gap: 8 }}>
          <span style={{ fontSize: 12, flex: 1 }}>
            AI generated code — review the chat above before applying.
          </span>
          <Space size={6}>
            <Button size="small" onClick={() => { onApply(pendingCode); setPendingCode(null); message.success(t('strategy.codeAssist.codeUpdated', 'Code updated.')); }}>
              Apply Code
            </Button>
            <Button size="small" onClick={() => setPendingCode(null)}>Dismiss</Button>
          </Space>
        </div>
      )}

      {/* Input */}
      <TextArea rows={2} value={draft} onChange={e => setDraft(e.target.value)}
        disabled={isBusy}
        placeholder={
          !code.trim()
            ? t('strategy.gen.placeholder', '描述你想创建的交易策略，例如："做一个 EURUSD 的布林带均值回归策略"')
            : t('strategy.codeAssist.reviseInputPlaceholder', 'e.g. Replace SMA(20) with EMA(50) and add a 1% stop-loss.')
        }
        onPressEnter={e => { if (!e.shiftKey) { e.preventDefault(); handleSend(); } }}
        style={{ fontSize: 13, marginBottom: 8 }}
      />
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8 }}>
        {draft.trim() && MODE_COLORS[detectMode(draft, !!code.trim())] && (() => {
          const m = detectMode(draft, !!code.trim());
          return <Tag color={MODE_COLORS[m]}>{modeLabel(t, m)}</Tag>;
        })()}
        {!draft.trim() && MODE_COLORS[code.trim() ? 'revise' : 'generate'] && (() => {
          const m = code.trim() ? 'revise' : 'generate';
          return <Tag color={MODE_COLORS[m]}>{modeLabel(t, m)}</Tag>;
        })()}
        <Button type="primary" icon={<SendOutlined />} loading={isBusy}
          onClick={handleSend} disabled={!draft.trim()}>
          {t('strategy.codeAssist.reviseSend', 'Send to AI')}
        </Button>
      </div>
    </div>
  );
}
