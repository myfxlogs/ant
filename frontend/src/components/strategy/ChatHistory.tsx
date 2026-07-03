import { useEffect, useRef } from 'react';
import { useTranslation } from 'react-i18next';

export interface ChatTurn {
  id: string;
  role: 'user' | 'ai';
  message: string;
  timestamp?: string;
  metrics?: { label: string; value: string; positive?: boolean }[];
}

interface Props {
  turns: ChatTurn[];
}

export default function ChatHistory({ turns }: Props) {
  const { t } = useTranslation();
  const endRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    endRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [turns.length]);

  if (turns.length === 0) return null;

  return (
    <div style={{ flex: '1 1 auto', overflowY: 'auto', padding: '4px 16px', minHeight: 60, maxHeight: 280, borderBottom: '1px solid var(--ant-color-border)' }}>
      {turns.map((turn) => (
        <div key={turn.id} style={{ margin: '10px 0' }}>
          <div style={{ fontSize: 10, color: '#484f58', marginBottom: 3 }}>
            {turn.role === 'user' ? 'You' : 'Agent'}
            {turn.timestamp ? ` · ${turn.timestamp}` : ''}
          </div>
          <div style={{
            background: turn.role === 'user' ? 'var(--ant-color-bg-elevated)' : 'var(--ant-color-bg-base)',
            border: turn.role === 'ai' ? '1px solid var(--ant-color-border)' : 'none',
            borderRadius: turn.role === 'user' ? '8px 8px 0 8px' : '8px 8px 8px 0',
            padding: '8px 12px',
            fontSize: 12,
            color: 'var(--ant-color-text)',
            whiteSpace: 'pre-wrap',
          }}>
            {turn.message}
          </div>
          {turn.metrics && turn.metrics.length > 0 && (
            <div style={{ display: 'flex', gap: 12, marginTop: 4, padding: '0 4px' }}>
              {turn.metrics.map((m, i) => (
                <span key={i} style={{ fontSize: 11, color: 'var(--ant-color-text-secondary)' }}>
                  <span style={{
                    fontWeight: 600,
                    fontSize: 12,
                    marginRight: 2,
                    color: m.positive === true ? '#3fb950' : m.positive === false ? '#f85149' : undefined,
                  }}>{m.value}</span>
                  {m.label}
                </span>
              ))}
            </div>
          )}
        </div>
      ))}
      <div ref={endRef} />
    </div>
  );
}
