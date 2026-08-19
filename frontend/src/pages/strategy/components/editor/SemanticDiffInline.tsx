import { Card, Typography } from 'antd';
import type { SemanticDiff } from '@/gen/ant/v1/agent_gateway_pb';

const { Text } = Typography;

const diffIcon: Record<string, string> = { added: '+', modified: '~', removed: '-', remaining: '=' };
const diffColor: Record<string, string> = { added: 'var(--color-success)', modified: 'var(--color-info)', removed: '#cf1322', remaining: 'var(--color-warning)' };

export default function SemanticDiffInline({ diff }: { diff: SemanticDiff | null }) {
  if (!diff || (!diff.changes?.length && !diff.effectSummary)) return null;
  return (
    <Card size="small" style={{ marginBottom: 8 }}>
      {diff.changes?.map((c, i) => (
        <div key={i} style={{ display: 'flex', gap: 6, marginBottom: 2, fontSize: 12 }}>
          <span style={{ color: diffColor[c.kind] || '#999' }}>{diffIcon[c.kind] || '~'}</span>
          <Text style={{ fontSize: 12, color: 'var(--color-text-secondary)' }}>{c.description}</Text>
        </div>
      ))}
      {diff.effectSummary && (
        <div style={{ borderTop: '1px solid var(--color-border)', paddingTop: 4, marginTop: 4 }}>
          <Text type="secondary" style={{ fontSize: 11 }}>{diff.effectSummary}</Text>
        </div>
      )}
    </Card>
  );
}
