import { Steps, Typography } from 'antd';
import { LoadingOutlined, CheckCircleOutlined, BulbOutlined, CodeOutlined, SafetyOutlined, BarChartOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';

interface Props {
  phase: string;
  plan?: string;
}

const stepKeys = [
  { key: 'analyzing', icon: <BulbOutlined />, titleKey: 'ai.wizard.steps.analyze', defaultTitle: 'Analyze' },
  { key: 'planning', icon: <BulbOutlined />, titleKey: 'ai.wizard.steps.plan', defaultTitle: 'Plan' },
  { key: 'generating', icon: <CodeOutlined />, titleKey: 'ai.wizard.steps.generate', defaultTitle: 'Generate' },
  { key: 'compliance', icon: <SafetyOutlined />, titleKey: 'ai.wizard.steps.compliance', defaultTitle: 'Compliance' },
  { key: 'backtest', icon: <BarChartOutlined />, titleKey: 'ai.wizard.steps.backtest', defaultTitle: 'Backtest' },
];

const stepOrder = ['analyzing', 'planning', 'generating', 'compliance', 'backtest', 'done'];

function stepIndex(phase: string): number {
  const idx = stepOrder.indexOf(phase);
  return idx >= 0 ? idx : stepOrder.length;
}

export default function StepProgress({ phase, plan }: Props) {
  const { t } = useTranslation();
  if (phase === 'idle' || phase === 'done') return null;

  const current = stepIndex(phase);

  return (
    <div style={{ marginBottom: 10 }}>
      {plan && (
        <div style={{
          padding: '6px 10px', marginBottom: 8, borderRadius: 6,
          background: '#f6ffed', border: '1px solid #b7eb8f', fontSize: 12, color: '#389e0d',
        }}>
          <b>📋 Plan:</b> {plan}
        </div>
      )}
      <Steps
        size="small"
        current={current}
        status={phase === 'done' ? 'finish' : 'process'}
        items={stepKeys.map((s, i) => ({
          title: t(s.titleKey as any, { defaultValue: s.defaultTitle }),
          icon: i < current ? <CheckCircleOutlined /> : i === current ? <LoadingOutlined /> : s.icon,
        }))}
      />
    </div>
  );
}
