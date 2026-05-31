import { Space, Spin, Typography } from 'antd';
import type { ChatMessage } from '../flow/useDebateFlow';

const { Text } = Typography;

export function MessageBubble({ m, waitHint }: { m: ChatMessage; waitHint?: string }) {
  // System bridge messages are only for prompting the LLM, never shown to end users.
  // Let the Agent start with its own introduction.
  if (m.kind === 'kickoff') {
    return null;
  }
  const isUser = m.role === 'user';
  return (
    <div style={{ display: 'flex', justifyContent: isUser ? 'flex-end' : 'flex-start', marginBottom: 8 }}>
      <div
        style={{
          maxWidth: '85%',
          background: isUser ? '#fff4d6' : '#ffffff',
          border: '1px solid #e5e7eb',
          borderRadius: 8,
          padding: '8px 12px',
          whiteSpace: 'pre-wrap',
          wordBreak: 'break-word',
        }}
      >
        {m.isLoading ? (
          <Space direction="vertical" size={4} align="start">
            <Spin size="small" />
            {waitHint ? (
              <Text type="secondary" style={{ fontSize: 12 }}>
                {waitHint}
              </Text>
            ) : null}
          </Space>
        ) : (
          <Text>{m.content}</Text>
        )}
      </div>
    </div>
  );
}
