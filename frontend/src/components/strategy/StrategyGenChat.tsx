import { useState, useRef, useCallback } from 'react';
import { Input, Button, Space, Tag, Typography, Alert } from 'antd';
import { ThunderboltOutlined, SendOutlined, LoadingOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { generateStrategyStream } from '@/client/strategyGen';

interface Props {
  symbol?: string;
  timeframe?: string;
  onApply: (code: string) => void;
}

type Phase = 'idle' | 'clarifying' | 'generating' | 'compliance' | 'backtest' | 'done';

export default function StrategyGenChat({ symbol, timeframe, onApply }: Props) {
  const { t } = useTranslation();
  const [phase, setPhase] = useState<Phase>('idle');
  const [userInput, setUserInput] = useState('');
  const [streamText, setStreamText] = useState('');
  const [questions, setQuestions] = useState<string[]>([]);
  const [templateName, setTemplateName] = useState('');
  const [backtestId, setBacktestId] = useState('');
  const [error, setError] = useState('');
  const [clarifyRound, setClarifyRound] = useState(0);
  const [genCode, setGenCode] = useState('');
  const abortRef = useRef<(() => void) | null>(null);
  const MAX_CLARIFY = 3;

  const reset = useCallback(() => {
    abortRef.current?.();
    setPhase('idle');
    setStreamText('');
    setQuestions([]);
    setTemplateName('');
    setBacktestId('');
    setError('');
  }, []);

  const runGeneration = useCallback((message: string, round: number) => {
    setPhase('generating');
    setStreamText('');
    setError('');
    setGenCode('');

    const abort = generateStrategyStream(
      { message, symbol, timeframe, clarificationRound: round },
      {
        onPhase: (p) => {
          if (p === 'clarifying') setPhase('clarifying');
          else if (p === 'generating') setPhase('generating');
          else if (p === 'compliance') setPhase('compliance');
          else if (p === 'backtest') setPhase('backtest');
          else if (p === 'done') setPhase('done');
        },
        onDelta: (d) => setStreamText((prev) => prev + d),
        onQuestions: (q) => setQuestions(q),
        onTemplate: (n) => setTemplateName(n),
        onCode: (c) => { setGenCode(c); onApply(c); },
        onBacktestId: (id) => setBacktestId(id),
        onError: (e) => setError(e),
        onDone: () => setPhase('done'),
      },
    );
    abortRef.current = abort;
  }, [symbol, timeframe, onApply]);

  const handleSend = useCallback(() => {
    const msg = userInput.trim();
    if (!msg) return;
    setUserInput('');
    setClarifyRound(0);
    runGeneration(msg, 0);
  }, [userInput, runGeneration]);

  const handleClarifyAnswer = useCallback((answer: string) => {
    const nextRound = clarifyRound + 1;
    setClarifyRound(nextRound);
    setQuestions([]);
    runGeneration(answer, nextRound);
  }, [clarifyRound, runGeneration]);

  const isBusy = phase === 'generating' || phase === 'clarifying';

  // ── Clarification questions ──
  if (questions.length > 0) {
    return (
      <div style={{ padding: 12, background: '#fffbe6', borderRadius: 6, border: '1px solid #ffe58f' }}>
        <Typography.Text strong style={{ fontSize: 13 }}>
          {t('strategy.gen.clarifyTitle', '需要确认几个细节：')}
        </Typography.Text>
        <Space direction="vertical" size={6} style={{ width: '100%', marginTop: 8 }}>
          {questions.map((q, i) => (
            <Button key={i} block size="small" type="dashed"
              onClick={() => handleClarifyAnswer(q)}
              disabled={clarifyRound >= MAX_CLARIFY}>
              {q}
            </Button>
          ))}
          {clarifyRound >= MAX_CLARIFY && (
            <Button block size="small" type="primary" onClick={() => {
              setQuestions([]);
              runGeneration('使用默认参数', clarifyRound);
            }}>
              {t('strategy.gen.useDefaults', '使用默认设置继续')}
            </Button>
          )}
        </Space>
      </div>
    );
  }

  // ── Main chat widget ──
  return (
    <div style={{ padding: 10, background: '#fafafa', borderRadius: 6, border: '1px solid #e8e8e8' }}>
      {/* Header */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginBottom: 8 }}>
        <ThunderboltOutlined style={{ color: '#faad14' }} />
        <Typography.Text strong style={{ fontSize: 13 }}>
          {t('strategy.gen.title', 'AI 策略生成')}
        </Typography.Text>
        {phase !== 'idle' && !isBusy && (
          <Button size="small" type="link" onClick={reset} style={{ marginLeft: 'auto' }}>
            {t('strategy.gen.reset', '重新开始')}
          </Button>
        )}
      </div>

      {/* Status tags */}
      {phase !== 'idle' && (
        <Space size={4} wrap style={{ marginBottom: 8 }}>
          {templateName && <Tag color="blue">{t('strategy.gen.template', '模板')}: {templateName}</Tag>}
          {isBusy && <Tag icon={<LoadingOutlined />} color="processing">{t('strategy.gen.generating', '生成中')}</Tag>}
          {phase === 'compliance' && <Tag color="orange">{t('strategy.gen.validating', '合规检查')}</Tag>}
          {backtestId && <Tag color="green">{t('strategy.gen.backtestStarted', '回测已启动')}</Tag>}
          {phase === 'done' && !error && <Tag color="success">{t('strategy.gen.done', '完成')}</Tag>}
        </Space>
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

      {/* Backtest result */}
      {backtestId && (
        <Alert type="info" showIcon style={{ marginBottom: 8 }}
          message={t('strategy.gen.backtestMsg', '回测任务已创建')}
          description={`Run ID: ${backtestId.slice(0, 8)}...`}
        />
      )}

      {/* Error */}
      {error && (
        <Alert type="warning" showIcon closable style={{ marginBottom: 8 }}
          message={error} onClose={() => setError('')}
        />
      )}

      {/* Input area */}
      {!isBusy && (
        <Input.TextArea
          rows={3}
          value={userInput}
          onChange={(e) => setUserInput(e.target.value)}
          onPressEnter={(e) => { e.preventDefault(); handleSend(); }}
          placeholder={t('strategy.gen.placeholder', '描述你想创建的交易策略，例如："做一个 EURUSD 的布林带均值回归策略，1小时周期"')}
          style={{ fontSize: 13, marginBottom: 8 }}
          disabled={isBusy}
        />
      )}

      {!isBusy && (
        <Button type="primary" icon={<SendOutlined />} size="small"
          onClick={handleSend} disabled={!userInput.trim()} block>
          {genCode ? t('strategy.gen.regenerate', '重新生成') : t('strategy.gen.send', '生成策略')}
        </Button>
      )}
    </div>
  );
}
