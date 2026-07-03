import { Card, Tag, Typography, Space, Empty } from 'antd';
import { PlusCircleOutlined, EditOutlined, MinusCircleOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import type { SemanticDiff, SemanticChange } from '@/gen/ant/v1/agent_gateway_pb';

interface Props {
  diff: SemanticDiff | null;
}

const kindIcon: Record<string, React.ReactNode> = {
  added: <PlusCircleOutlined style={{ color: '#52c41a' }} />,
  modified: <EditOutlined style={{ color: '#1677ff' }} />,
  removed: <MinusCircleOutlined style={{ color: '#cf1322' }} />,
};

const kindColor: Record<string, string> = {
  added: 'success',
  modified: 'processing',
  removed: 'error',
};

export default function SemanticDiffCard({ diff }: Props) {
  const { t } = useTranslation();

  if (!diff || (!diff.changes?.length && !diff.effectSummary)) {
    return null;
  }

  return (
    <Card
      size="small"
      style={{ marginBottom: 8, borderColor: '#d9d9d9' }}
      title={
        <Space size={4}>
          <Typography.Text strong style={{ fontSize: 13 }}>
            {t('agent.semantic_diff.title', 'Strategy Changes')}
          </Typography.Text>
        </Space>
      }
    >
      {diff.changes?.length > 0 && (
        <div style={{ marginBottom: 6 }}>
          {diff.changes.map((change: SemanticChange, i: number) => (
            <div key={i} style={{ display: 'flex', alignItems: 'flex-start', gap: 6, marginBottom: 3 }}>
              <span style={{ fontSize: 12, marginTop: 1 }}>{kindIcon[change.kind] || <EditOutlined />}</span>
              <Tag color={kindColor[change.kind] || 'default'} style={{ fontSize: 10, flexShrink: 0 }}>
                {change.kind}
              </Tag>
              <Typography.Text style={{ fontSize: 12, color: '#595959' }}>
                {change.description}
              </Typography.Text>
            </div>
          ))}
        </div>
      )}

      {diff.effectSummary && (
        <div style={{ paddingTop: 6, borderTop: '1px solid #f0f0f0' }}>
          <Typography.Text type="secondary" style={{ fontSize: 11 }}>
            {t('agent.semantic_diff.effect', 'Effect')}
          </Typography.Text>
          <Typography.Paragraph style={{ fontSize: 12, marginTop: 2, color: '#595959', marginBottom: 0 }}>
            {diff.effectSummary}
          </Typography.Paragraph>
        </div>
      )}
    </Card>
  );
}
