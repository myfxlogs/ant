import { Steps, Progress, Button, Card } from 'antd';
import { LoadingOutlined, CloseCircleOutlined, CheckCircleOutlined } from '@ant-design/icons';
import type { TFunction } from 'i18next';

type Stage = 'idle' | 'generating' | 'compiling' | 'backtesting' | 'evaluating' | 'publishing' | 'completed' | 'failed';

const STAGE_ORDER: Stage[] = ['generating', 'compiling', 'backtesting', 'evaluating', 'publishing', 'completed'];

interface AutoGenerateProgressProps {
  stage: string;
  progress: number;
  delta: string;
  onCancel: () => void;
  t: TFunction;
}

function stageToStepIndex(stage: string): number {
  const idx = STAGE_ORDER.indexOf(stage as Stage);
  return idx < 0 ? 0 : idx;
}

export default function AutoGenerateProgress({ stage, progress, delta, onCancel, t }: AutoGenerateProgressProps) {
  const currentIdx = stageToStepIndex(stage);
  const steps = STAGE_ORDER.map(s => {
    const idx = STAGE_ORDER.indexOf(s);
    let status: 'wait' | 'process' | 'finish' | 'error' = 'wait';
    if (stage === 'failed') {
      if (idx < currentIdx) status = 'finish';
      else if (idx === currentIdx) status = 'error';
    } else if (idx < currentIdx) {
      status = 'finish';
    } else if (idx === currentIdx) {
      status = 'process';
    }
    return {
      title: t(`marketplace.autogen.stages.${s}`),
      status,
    };
  });

  const currentIcon = (status: 'wait' | 'process' | 'finish' | 'error') => {
    if (status === 'process') return <LoadingOutlined />;
    if (status === 'error') return <CloseCircleOutlined />;
    if (status === 'finish') return <CheckCircleOutlined />;
    return undefined;
  };

  return (
    <div>
      <Steps
        current={currentIdx}
        items={steps.map(s => ({ title: s.title, status: s.status, icon: currentIcon(s.status) }))}
        size="small"
        style={{ marginBottom: 16 }}
      />
      <Progress percent={Math.round(progress * 100)} status="active" style={{ marginBottom: 16 }} />
      {delta && (
        <Card size="small" style={{ maxHeight: 200, overflow: 'auto', marginBottom: 16, background: 'var(--color-bg-secondary)' }}>
          <pre style={{ whiteSpace: 'pre-wrap', fontSize: 12, margin: 0 }}>{delta}</pre>
        </Card>
      )}
      <Button onClick={onCancel} danger>{t('marketplace.autogen.cancel')}</Button>
    </div>
  );
}
