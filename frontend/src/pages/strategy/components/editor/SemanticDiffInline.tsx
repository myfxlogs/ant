import { Card, Typography } from 'antd';
import type { SemanticDiff } from '@/gen/ant/v1/agent_gateway_pb';

const { Text } = Typography;

const diffIcon: Record<string, string> = { added: '+', modified: '~', removed: '-', remaining: '=' };
const diffColor: Record<string, string> = { added: '#52c41a', modified: '#1677ff', removed: '#cf1322', remaining: '#faad14' };

export default function SemanticDiffInline({ diff }: { diff: SemanticDiff | null }) {
  if (!diff || (!diff.changes?.length && !diff.effectSummary)) return null;
  return (
    <Card size="small" style={{ marginBottom: 8 }}>
      {diff.changes?.map((c, i) => (
        <div key={i} style={{ display: 'flex', gap: 6, marginBottom: 2, fontSize: 12 }}>
          <span style={{ color: diffColor[c.kind] || '#999' }}>{diffIcon[c.kind] || '~'}</span>
          <Text style={{ fontSize: 12, color: '#595959' }}>{c.description}</Text>
        </div>
      ))}
      {diff.effectSummary && (
        <div style={{ borderTop: '1px solid #f0f0f0', paddingTop: 4, marginTop: 4 }}>
          <Text type="secondary" style={{ fontSize: 11 }}>{diff.effectSummary}</Text>
        </div>
      )}
    </Card>
  );
}
