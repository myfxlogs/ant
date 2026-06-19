import { Steps, Typography } from 'antd';
import { LoadingOutlined, CheckCircleOutlined, BulbOutlined, CodeOutlined, SafetyOutlined, BarChartOutlined } from '@ant-design/icons';

interface Props {
  phase: string;
  plan?: string;
}

const steps = [
  { key: 'analyzing', icon: <BulbOutlined />, title: 'Analyze' },
  { key: 'planning', icon: <BulbOutlined />, title: 'Plan' },
  { key: 'generating', icon: <CodeOutlined />, title: 'Generate' },
  { key: 'compliance', icon: <SafetyOutlined />, title: 'Compliance' },
  { key: 'backtest', icon: <BarChartOutlined />, title: 'Backtest' },
];

const stepOrder = ['analyzing', 'planning', 'generating', 'compliance', 'backtest', 'done'];

function stepIndex(phase: string): number {
  const idx = stepOrder.indexOf(phase);
  return idx >= 0 ? idx : stepOrder.length;
}

export default function StepProgress({ phase, plan }: Props) {
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
        items={steps.map((s, i) => ({
          title: s.title,
          icon: i < current ? <CheckCircleOutlined /> : i === current ? <LoadingOutlined /> : s.icon,
        }))}
      />
    </div>
  );
}
