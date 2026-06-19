import { useState, useRef, useCallback } from 'react';
import { Button, Card, Input, Space, Tag, Typography, Spin } from 'antd';
import { ThunderboltOutlined, CheckCircleOutlined, EditOutlined, SendOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next'
import {
  PLACEHOLDER_KEY, PLAN_TITLE_KEY, PLAN_ANALYZING_KEY, PLAN_ERROR_TAG_KEY,
  PLAN_RESET_KEY, PLAN_CARD_TITLE_KEY, PLAN_EDIT_KEY, PLAN_EDIT_CANCEL_KEY,
  PLAN_CONFIRM_BTN_KEY, PLAN_SEND_BTN_KEY, PLAN_SYMBOL_WARN_KEY,
  PLAN_SYMBOL_OK_KEY, PLAN_PREREQUISITE_MSG_KEY,
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
  const [editing, setEditing] = useState(false);
  const [editDraft, setEditDraft] = useState('');
  const abortRef = useRef<(() => void) | null>(null);

  const reset = useCallback(() => {
    abortRef.current?.();
    setPhase('idle'); setPlan(''); setStreamText(''); setError('');
    setEditing(false); setEditDraft('');
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

  const handleConfirm = () => {
    const finalPlan = editing ? editDraft.trim() : plan;
    if (finalPlan) onPlanConfirmed(finalPlan);
  };

  const hasSymbol = !!(symbol && timeframe);

  if (phase === 'plan_ready' || editing) {
    const displayPlan = editing ? editDraft : plan;
    return (
      <div style={{ padding: 10 }}>
        <Card size="small" style={{ borderRadius: 8, borderColor: '#b7eb8f', background: '#f6ffed' }}
          title={<span><CheckCircleOutlined style={{ color: '#52c41a' }} /> {t(PLAN_CARD_TITLE_KEY, 'AI Execution Plan')}</span>}
          actions={[
            <Button key="edit" size="small" icon={<EditOutlined />}
              onClick={() => { setEditing(!editing); if (!editing) setEditDraft(plan); }}>
              {editing ? t(PLAN_EDIT_CANCEL_KEY, 'Cancel') : t(PLAN_EDIT_KEY, 'Edit')}
            </Button>,
            <Button key="confirm" type="primary" size="small" icon={<ThunderboltOutlined />}
              onClick={handleConfirm}>
              {t(PLAN_CONFIRM_BTN_KEY, 'Confirm & Generate Code')}
            </Button>,
          ]}
        >
          {editing ? (
            <TextArea rows={6} value={editDraft} onChange={e => setEditDraft(e.target.value)}
              style={{ fontSize: 13, fontFamily: 'monospace' }} />
          ) : (
            <Typography.Paragraph style={{ fontSize: 13, whiteSpace: 'pre-wrap', margin: 0 }}>
              {displayPlan}
            </Typography.Paragraph>
          )}
        </Card>
      </div>
    );
  }

  return (
    <div style={{ padding: 10, border: '1px solid #f0f0f0', borderRadius: 6, background: '#fafafa' }}>
      {/* Symbol indicator */}
      <div style={{ marginBottom: 6, display: 'flex', alignItems: 'center', gap: 6 }}>
        {hasSymbol ? (
          <Tag color="blue" style={{ fontSize: 11 }}>📊 {t(PLAN_SYMBOL_OK_KEY, '{symbol} · {timeframe}').replace('{symbol}', symbol!).replace('{timeframe}', timeframe!)}</Tag>
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
