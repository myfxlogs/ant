import { useState, useCallback } from 'react';
import { Button, Input, Space, Typography, Tag, Card } from 'antd';
import { CheckCircleOutlined, EditOutlined, ThunderboltOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import type { StrategyPlan } from '@/gen/ant/v1/agent_gateway_pb';
import {
  PLAN_CONFIRM_BTN_KEY,
  PLAN_EDIT_KEY,
  PLAN_EDIT_CANCEL_KEY,
  PLAN_HINT_KEY,
  PLAN_INPUT_PLACEHOLDER_KEY,
  PLAN_SEND_BTN_KEY,
  PLAN_CARD_TITLE_KEY,
} from '@/gen/ant/v1/i18n/strategy_gen_keys';

interface Props {
  plan: StrategyPlan;
  onConfirm: () => void;
  onRefine: (feedback: string) => void;
  refining?: boolean;
}

export default function PlanCard({ plan, onConfirm, onRefine, refining }: Props) {
  const { t } = useTranslation();
  const [editing, setEditing] = useState(false);
  const [feedback, setFeedback] = useState('');

  const handleRefine = useCallback(() => {
    const msg = feedback.trim();
    if (!msg) return;
    setFeedback('');
    setEditing(false);
    onRefine(msg);
  }, [feedback, onRefine]);

  const fields: Array<{ label: string; value: string }> = [
    { label: t('strategy.gen.planType', 'Type'), value: plan.type },
    { label: t('strategy.gen.planEntry', 'Entry'), value: plan.entry },
    { label: t('strategy.gen.planExit', 'Exit'), value: plan.exit },
    { label: t('strategy.gen.planRisk', 'Risk'), value: plan.risk },
    { label: t('strategy.gen.planMarket', 'Market'), value: plan.market },
  ];

  return (
    <Card
      size="small"
      style={{ marginBottom: 8, background: 'var(--ant-color-bg-elevated)', borderColor: 'var(--ant-color-border)' }}
      title={
        <Space size={4}>
          <ThunderboltOutlined style={{ color: '#faad14' }} />
          <Typography.Text strong style={{ fontSize: 13 }}>
            {t(PLAN_CARD_TITLE_KEY, 'AI Strategy Plan')}
          </Typography.Text>
        </Space>
      }
    >
      {fields.map((f, i) => f.value && (
        <div key={i} style={{ marginBottom: 4 }}>
          <Typography.Text type="secondary" style={{ fontSize: 11, fontWeight: 600 }}>
            {f.label}:
          </Typography.Text>
          <Typography.Text style={{ fontSize: 12, marginLeft: 6 }}>
            {f.value}
          </Typography.Text>
        </div>
      ))}

      {!editing ? (
        <Space size={8} style={{ marginTop: 8 }}>
          <Button
            type="primary"
            size="small"
            icon={<CheckCircleOutlined />}
            onClick={onConfirm}
          >
            {t(PLAN_CONFIRM_BTN_KEY, 'Confirm & Generate')}
          </Button>
          <Button
            size="small"
            icon={<EditOutlined />}
            onClick={() => setEditing(true)}
          >
            {t(PLAN_EDIT_KEY, 'Modify')}
          </Button>
        </Space>
      ) : (
        <div style={{ marginTop: 8 }}>
          <Typography.Text type="secondary" style={{ fontSize: 11, display: 'block', marginBottom: 4 }}>
            {t(PLAN_HINT_KEY, 'Describe what to change')}
          </Typography.Text>
          <Space.Compact style={{ width: '100%' }}>
            <Input
              value={feedback}
              onChange={(e) => setFeedback(e.target.value)}
              placeholder={t(PLAN_INPUT_PLACEHOLDER_KEY, 'e.g. add RSI < 70 filter')}
              onPressEnter={handleRefine}
              disabled={refining}
              style={{ fontSize: 12 }}
            />
            <Button
              type="primary"
              size="small"
              onClick={handleRefine}
              loading={refining}
              disabled={!feedback.trim()}
            >
              {t(PLAN_SEND_BTN_KEY, 'Update Plan')}
            </Button>
            <Button
              size="small"
              onClick={() => { setEditing(false); setFeedback(''); }}
            >
              {t(PLAN_EDIT_CANCEL_KEY, 'Cancel')}
            </Button>
          </Space.Compact>
        </div>
      )}

      {refining && !editing && (
        <Tag color="processing" style={{ marginTop: 8 }}>
          {t('strategy.gen.planAnalyzing', 'Updating plan...')}
        </Tag>
      )}
    </Card>
  );
}
