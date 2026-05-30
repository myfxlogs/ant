import { Space, Spin, Typography } from 'antd';
import type { ChatMessage } from '../flow/useDebateFlow';

const { Text } = Typography;

export function MessageBubble({ m, waitHint }: { m: ChatMessage; waitHint?: string }) {
  // 系统衔接消息仅用于提示 LLM，不展示给最终用户——保持对话干净，
  // 直接由 Agent 的自我介绍开场即可。
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
