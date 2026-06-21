import { useState, useRef, useCallback } from 'react';
import { Button, Input, Space, Tag, Typography, Spin } from 'antd';
import { ThunderboltOutlined, SendOutlined, LoadingOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next'
import {
  PLACEHOLDER_KEY, PLAN_TITLE_KEY, PLAN_ANALYZING_KEY, PLAN_ERROR_TAG_KEY,
  PLAN_RESET_KEY, PLAN_SEND_BTN_KEY, PLAN_SYMBOL_WARN_KEY,
  PLAN_PREREQUISITE_MSG_KEY,
  PLAN_HINT_KEY, PLAN_INPUT_PLACEHOLDER_KEY,
} from '@/gen/ant/v1/i18n/strategy_gen_keys';
import { analyzePlan, type PlanCallbacks } from '@/client/strategyPlan';

const { TextArea } = Input;

interface Props {
  symbol?: string;
  timeframe?: string;
  sessionId?: string;
  onPlanConfirmed: (plan: string) => void;
}

type Phase = 'idle' | 'analyzing' | 'plan_ready' | 'error';

export default function PlanPanel({ symbol, timeframe, sessionId, onPlanConfirmed }: Props) {
  const { t } = useTranslation();
  const [phase, setPhase] = useState<Phase>('idle');
  const [draft, setDraft] = useState('');
  const [plan, setPlan] = useState('');
  const [streamText, setStreamText] = useState('');
  const [error, setError] = useState('');
  const [refining, setRefining] = useState(false);
  const [planDraft, setPlanDraft] = useState('');
  const abortRef = useRef<(() => void) | null>(null);

  const reset = useCallback(() => {
    abortRef.current?.();
    setPhase('idle'); setPlan(''); setStreamText(''); setError('');
    setRefining(false); setPlanDraft('');
  }, []);

  const handleSend = useCallback(() => {
    const msg = draft.trim();
    if (!msg) return;
    if (!symbol || !timeframe) {
      setError(t(PLAN_PREREQUISITE_MSG_KEY, 'Please select a trading symbol and timeframe first.'));
      return;
    }
    setDraft(''); setPhase('analyzing'); setStreamText(''); setError(''); setPlan('');

    const abort = analyzePlan(
      { message: msg, conversationId: sessionId, symbol, timeframe },
      {
        onDelta: (d) => setStreamText(p => p + d),
        onPlan: (p) => { setPlan(p); setPhase('plan_ready'); },
        onError: (e) => { setError(e); setPhase('error'); },
        onDone: () => {},
      } satisfies PlanCallbacks,
    );
    abortRef.current = abort;
  }, [draft, sessionId, symbol, timeframe, t]);

  const handleRefine = useCallback(() => {
    const msg = planDraft.trim();
    if (!msg) return;
    // Natural language: user can discuss or say "generate"
    if (/生成代码|可以|好|行|ok|yes|go|继续|确认/.test(msg)) {
      onPlanConfirmed(plan);
      return;
    }
    setPlanDraft(''); setRefining(true); setError('');
    const abort = analyzePlan(
      { message: `关于这个策略计划：${plan}\n\n用户的反馈：${msg}\n\n请根据反馈修改计划，直接输出修改后的计划。`, conversationId: sessionId, symbol, timeframe },
      {
        onDelta: () => {},
        onPlan: (p) => { setPlan(p); setRefining(false); },
        onError: (e) => { setError(e); setRefining(false); },
        onDone: () => {},
      } satisfies PlanCallbacks,
    );
    abortRef.current = abort;
  }, [planDraft, plan, sessionId, symbol, timeframe, onPlanConfirmed]);

  const hasSymbol = !!(symbol && timeframe);

  if (phase === 'plan_ready') {
    return (
      <div style={{ padding: 10, border: '1px solid #f0f0f0', borderRadius: 6, background: '#fafafa' }}>
        <div style={{ padding: '8px 10px', marginBottom: 8, borderRadius: 6,
          background: '#f6ffed', border: '1px solid #b7eb8f' }}>
          <Typography.Text strong style={{ fontSize: 11, color: '#389e0d', display: 'block', marginBottom: 4 }}>
            {t(PLAN_TITLE_KEY)}
          </Typography.Text>
          <Typography.Paragraph style={{ fontSize: 12, whiteSpace: 'pre-wrap', margin: 0, color: '#262626' }}>
            {plan}
          </Typography.Paragraph>
        </div>

        {refining && (
          <div style={{ padding: 6, textAlign: 'center', color: '#8c8c8c', fontSize: 12 }}>
            <LoadingOutlined style={{ marginRight: 4 }} />{t(PLAN_ANALYZING_KEY)}
          </div>
        )}

        {error && (
          <div style={{ padding: 6, marginBottom: 6, background: '#fffbe6', borderRadius: 4, fontSize: 11, color: '#ad6800' }}>
            {error}
          </div>
        )}

        <div style={{ fontSize: 11, color: '#8c8c8c', marginBottom: 6 }}>
          {t(PLAN_HINT_KEY)}
        </div>
        <Space.Compact style={{ width: '100%' }}>
          <Input value={planDraft} onChange={e => setPlanDraft(e.target.value)}
            placeholder={t(PLAN_INPUT_PLACEHOLDER_KEY)}
            onPressEnter={handleRefine}
            disabled={refining}
            style={{ fontSize: 12 }}
          />
          <Button type="primary" icon={<SendOutlined />}
            onClick={handleRefine} loading={refining}
            disabled={!planDraft.trim()}>{t(PLAN_SEND_BTN_KEY)}</Button>
        </Space.Compact>
      </div>
    );
  }

  return (
    <div style={{ padding: 10, border: '1px solid #f0f0f0', borderRadius: 6, background: '#fafafa' }}>
      <div style={{ marginBottom: 6, display: 'flex', alignItems: 'center', gap: 6 }}>
        {hasSymbol ? (
          <Tag color="blue" style={{ fontSize: 11 }}>📊 {symbol} · {timeframe}</Tag>
        ) : (
          <Tag color="warning" style={{ fontSize: 11 }}>{t(PLAN_SYMBOL_WARN_KEY, 'Please select symbol and timeframe above')}</Tag>
        )}
      </div>

      <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginBottom: 8 }}>
        <ThunderboltOutlined style={{ color: '#faad14' }} />
        <Typography.Text strong style={{ fontSize: 13 }}>{t(PLAN_TITLE_KEY, 'AI Strategy Planner')}</Typography.Text>
        {phase === 'analyzing' && <Tag color="processing" icon={<Spin size="small" />}>{t(PLAN_ANALYZING_KEY, 'Analyzing')}</Tag>}
        {phase === 'error' && <Tag color="error">{t(PLAN_ERROR_TAG_KEY, 'Error')}</Tag>}
        {phase !== 'idle' && (
          <Button size="small" type="link" onClick={reset} style={{ marginLeft: 'auto' }}>{t(PLAN_RESET_KEY, 'Start Over')}</Button>
        )}
      </div>

      {streamText && (
        <div style={{ maxHeight: 200, overflow: 'auto', padding: 8, marginBottom: 8,
          background: '#fff', borderRadius: 4, border: '1px solid #f0f0f0',
          fontSize: 12, whiteSpace: 'pre-wrap', color: '#595959' }}>
          {streamText}
        </div>
      )}

      {error && (
        <div style={{ padding: 8, marginBottom: 8, background: '#fffbe6', borderRadius: 4, border: '1px solid #ffe58f', fontSize: 12, color: '#ad6800' }}>
          ⚠️ {error}
        </div>
      )}

      <Space.Compact style={{ width: '100%' }}>
        <TextArea rows={2} value={draft} onChange={e => setDraft(e.target.value)}
          disabled={phase === 'analyzing'}
          placeholder={t(PLACEHOLDER_KEY, 'Describe the trading strategy you want to create…')}
          onPressEnter={e => { if (!e.shiftKey) { e.preventDefault(); handleSend(); } }}
          style={{ fontSize: 13 }}
        />
      </Space.Compact>
      <Button type="primary" icon={<SendOutlined />} size="small" block
        onClick={handleSend} disabled={!draft.trim() || phase === 'analyzing'}
        loading={phase === 'analyzing'} style={{ marginTop: 8 }}>
        {t(PLAN_SEND_BTN_KEY, 'Analyze & Generate Plan')}
      </Button>
    </div>
  );
}
