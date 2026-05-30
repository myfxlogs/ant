import { useEffect, useRef, useState } from 'react';
import { Alert, Button, Empty, Input, Spin, Typography, message as antMessage } from 'antd';
import { useTranslation } from 'react-i18next';
import type { StepKey } from '../flow/useDebateFlow';
import { formatElapsed } from '../helpers';
import { MessageBubble } from './MessageBubble';
import { looksLikeNextIntent } from '../helpers';
import { ChatNavBar } from './ChatNavBar';

const { Text } = Typography;

export function ChatStep(props: {
  stepKey: StepKey;
  stepLabel: string;
  state: ReturnType<ReturnType<typeof import('../flow/useDebateFlow')['useDebateFlow']>['stepState']>;
  sending: boolean;
  modelWaitActive: boolean;
  modelWaitElapsedSeconds: number;
  /** LLM deltas while async advance (kickoff) is streaming via SSE. */
  streamingPreview?: string;
  onSend: (text: string) => Promise<void>;
  onBack: () => void;
  onNext: () => void;
  isFirstChat: boolean;
  isLastAgent: boolean;
  canBack?: boolean;
}) {
  const { t } = useTranslation();
  const {
    stepLabel,
    state,
    sending,
    modelWaitActive,
    modelWaitElapsedSeconds,
    streamingPreview,
    onSend,
    onBack,
    onNext,
    isFirstChat,
    isLastAgent,
    canBack = true,
  } = props;
  const [input, setInput] = useState('');
  const listRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    const el = listRef.current;
    if (el) el.scrollTop = el.scrollHeight;
  }, [state.messages.length, streamingPreview]);

  async function handleSend() {
    const text = input.trim();
    if (!text) {
      antMessage.warning(t('ai.debate.messages.inputFirst'));
      return;
    }
    if (looksLikeNextIntent(text)) {
      setInput('');
      antMessage.info(t('ai.debate.v2.autoAdvanceHint', {
        defaultValue: 'Detected a "next" intent, advancing to the next step.',
      }));
      onNext();
      return;
    }
    setInput('');
    await onSend(text);
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', minHeight: 480 }}>
      <div style={{ marginBottom: 8 }}>
        <Text strong>{stepLabel}</Text>
        <Text type="secondary" style={{ marginLeft: 8, fontSize: 12 }}>
          {t('ai.debate.v2.chatHint', {
            defaultValue: 'Tell me your idea and intent. I will summarize my understanding; when you are happy with it, click Next.',
          })}
        </Text>
      </div>
      <div
        ref={listRef}
        style={{
          flex: 1,
          minHeight: 320,
          maxHeight: 560,
          overflowY: 'auto',
          border: '1px solid #e5e7eb',
          borderRadius: 8,
          padding: 12,
          background: '#fafafa',
        }}
      >
        {state.messages.length === 0 ? (
          <Empty
            description={isFirstChat
              ? t('ai.debate.v2.chatEmptyIntent', {
                defaultValue: 'Describe the strategy you want in natural language. The assistant will help you shape it.',
              })
              : t('ai.debate.v2.chatEmptyAgent', {
                defaultValue: 'Chat naturally with the current expert. They will give suggestions and ask questions within their own scope.',
              })}
          />
        ) : (
          state.messages.map((m) => (
            <MessageBubble
              key={m.id}
              m={m}
              waitHint={
                modelWaitActive && m.isLoading
                  ? t('ai.debate.v2.modelWaitBubble', {
                      defaultValue: 'Waiting {{time}}',
                      time: formatElapsed(modelWaitElapsedSeconds),
                    })
                  : undefined
              }
            />
          ))
        )}
        {sending && streamingPreview ? (
          <div style={{ display: 'flex', justifyContent: 'flex-start', marginBottom: 8 }}>
            <div
              style={{
                maxWidth: '85%',
                background: '#ffffff',
                border: '1px solid #e5e7eb',
                borderRadius: 8,
                padding: '8px 12px',
                whiteSpace: 'pre-wrap',
                wordBreak: 'break-word',
              }}
            >
              <Text type="secondary" style={{ fontSize: 11, display: 'block', marginBottom: 4 }}>
                {t('ai.debate.v2.streamingPreview', { defaultValue: 'Generating…' })}
              </Text>
              <Text>{streamingPreview}</Text>
            </div>
          </div>
        ) : null}
      </div>
      {modelWaitActive ? (
        <Alert
          type="info"
          showIcon
          style={{ marginTop: 8 }}
          message={t('ai.debate.v2.modelWaitBanner', {
            defaultValue: 'Model is working… Elapsed {{time}}',
            time: formatElapsed(modelWaitElapsedSeconds),
          })}
        />
      ) : null}
      <div style={{ marginTop: 8, display: 'flex', gap: 8 }}>
        <Input.TextArea
          value={input}
          onChange={(e) => setInput(e.target.value)}
          rows={2}
          placeholder={isFirstChat
            ? t('ai.debate.placeholders.intent')
            : t('ai.debate.v2.chatPlaceholder', { defaultValue: 'Say something to the current expert…' })}
          onPressEnter={(e) => {
            if (!e.shiftKey) {
              e.preventDefault();
              void handleSend();
            }
          }}
        />
        <Button type="primary" loading={sending} onClick={handleSend}>
          {t('ai.debate.v2.send', { defaultValue: 'Send' })}
        </Button>
      </div>

      <ChatNavBar sending={sending} canBack={canBack} isLastAgent={isLastAgent} onBack={onBack} onNext={onNext} />
    </div>
  );
}
