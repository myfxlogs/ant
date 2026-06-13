import { useState, useRef, useEffect, useCallback } from 'react';
import { Button, Space, Tag, Typography, message } from 'antd';
import { ThunderboltOutlined, LoadingOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { generateStrategyStream } from '@/client/strategyGen';
import { codeAssistApi, type CodeChatMessage } from '@/client/codeAssist';
import { pythonStrategyApi } from '@/client/pythonStrategy';
import type { BacktestRunUpdate } from '@/gen/ant/v1/backtest_run_query_pb';
import { isSucceededRun } from '@/pages/strategy/StrategyTemplatePage.utils';
import {
  ChatMessagesView,
  ChatClarificationView,
  ChatPendingCodeBanner,
  ChatInputBar,
  detectMode,
  modeLabel,
  MODE_COLORS,
  type BacktestMetrics,
} from './AIChatPanelSections';

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
  const [backtestMetrics, setBacktestMetrics] = useState<BacktestMetrics | null>(null);
  const [analysisText, setAnalysisText] = useState('');
  const [adviceText, setAdviceText] = useState('');
  const [lastGeneratedCode, setLastGeneratedCode] = useState('');
  const abortRef = useRef<(() => void) | null>(null);
  const streamRef = useRef('');
  const watchRef = useRef<(() => void) | null>(null);

  useEffect(() => () => { abortRef.current?.(); watchRef.current?.(); }, []);

  useEffect(() => {
    if (initialPrompt) { setDraft(initialPrompt); }
  }, [initialPrompt]);

  const reset = useCallback(() => {
    abortRef.current?.();
    watchRef.current?.();
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
          watchRef.current?.();
          const stop = pythonStrategyApi.watchBacktestRun(id, (update: BacktestRunUpdate) => {
            if (isSucceededRun(update.run) && update.metrics) {
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
          watchRef.current = stop;
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

  // Compute mode tag for input bar
  const currentMode = draft.trim()
    ? detectMode(draft, !!code.trim(), !!backtestMetrics)
    : code.trim() ? 'revise' : 'generate';

  // Clarification mode — early return
  if (questions.length > 0) {
    return (
      <ChatClarificationView
        questions={questions}
        clarifyRound={clarifyRound}
        onAnswer={handleClarifyAnswer}
        onUseDefaults={() => { setQuestions([]); handleGenerate('使用默认参数', clarifyRound); }}
        t={t}
      />
    );
  }

  return (
    <div style={{ border: '1px solid #f0f0f0', borderRadius: 6, padding: 10, background: '#fafafa' }}>
      {/* Header */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginBottom: 8 }}>
        <ThunderboltOutlined style={{ color: '#faad14' }} />
        <Typography.Text strong style={{ fontSize: 13 }}>{t('strategy.aiChat.title')}</Typography.Text>
        <Space size={4} wrap style={{ marginLeft: 8 }}>
          {!code.trim() && !backtestMetrics && <Tag color="blue">{t('strategy.gen.title', '策略生成')}</Tag>}
          {!!code.trim() && !backtestMetrics && <Tag color="green">{t('strategy.aiChat.revise')}</Tag>}
          {backtestMetrics && <Tag color="purple">{t('strategy.aiChat.feedback')}</Tag>}
          {isBusy && <Tag icon={<LoadingOutlined />} color="processing">{t('strategy.aiChat.streaming')}</Tag>}
          {phase === 'analyzing' && isBusy && <Tag color="orange">{t('strategy.aiChat.analyzing')}</Tag>}
          {phase && !isBusy && phase !== 'done' && <Tag>{phase}</Tag>}
          {backtestId && <Tag color="success">backtest: {backtestId.slice(0, 8)}</Tag>}
        </Space>
        {mode !== 'idle' && !isBusy && (
          <Button size="small" type="link" onClick={reset} style={{ marginLeft: 'auto' }}>{t('strategy.aiChat.reset')}</Button>
        )}
      </div>

      {/* Messages + streaming + backtest metrics */}
      <ChatMessagesView
        history={history}
        analysisText={analysisText}
        adviceText={adviceText}
        streamText={streamText}
        backtestMetrics={backtestMetrics}
        mode={mode}
        t={t}
      />

      {/* Error */}
      {error && (
        <div style={{ padding: 4, marginBottom: 6, background: '#fff2f0', borderRadius: 4,
          fontSize: 11, color: '#cf1322' }}>{error}</div>
      )}

      {/* Pending code — shown when autoApply=false */}
      {pendingCode && !autoApply && (
        <ChatPendingCodeBanner
          pendingCode={pendingCode}
          onApply={(c) => { onApply(c); setPendingCode(null); message.success(t('strategy.codeAssist.codeUpdated', 'Code updated.')); }}
          onDismiss={() => setPendingCode(null)}
          t={t}
        />
      )}

      {/* Input bar */}
      <ChatInputBar
        draft={draft}
        busy={isBusy}
        hasCode={!!code.trim()}
        hasBacktest={!!backtestMetrics}
        modeTag={modeLabel(t, currentMode)}
        modeColor={MODE_COLORS[currentMode]}
        onDraftChange={setDraft}
        onSend={handleSend}
        t={t}
      />
    </div>
  );
}
