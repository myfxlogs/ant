import React, { useEffect, useRef, useState } from 'react';
import { Button, Input, Space, message } from 'antd';
import { RobotOutlined, SendOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next'
import { AI_REVISE_TITLE_KEY, CODE_UPDATED_KEY, ENTER_INSTRUCTION_KEY, GENERATE_PLACEHOLDER_KEY, REVISE_INPUT_PLACEHOLDER_KEY, REVISE_SEND_KEY } from '@/gen/ant/v1/i18n/strategy_code_assist_keys';

;
import { codeAssistApi, type CodeChatMessage } from '@/client/codeAssist';

const { TextArea } = Input;

export interface AICodeReviseChatProps {
  code: string;
  onApply: (newCode: string) => void;
  initialInstruction?: string;
}

export const AICodeReviseChat: React.FC<AICodeReviseChatProps> = ({ code, onApply, initialInstruction }) => {
  const { t, i18n } = useTranslation();
  const [history, setHistory] = useState<CodeChatMessage[]>([]);
  const [draft, setDraft] = useState(initialInstruction || '');
  const [loading, setLoading] = useState(false);
  const [streamingText, setStreamingText] = useState('');
  const streamingRef = useRef('');
  const stopRef = useRef<(() => void) | null>(null);

  useEffect(() => () => stopRef.current?.(), []);

  // Sync initialInstruction into draft when prop changes (e.g. validation errors → AI fix).
  const [prevInstruction, setPrevInstruction] = useState(initialInstruction);
  if (initialInstruction !== prevInstruction) {
    setPrevInstruction(initialInstruction);
    setDraft(initialInstruction || '');
  }

  const chatEndRef = useRef<HTMLDivElement>(null);

  const send = () => {
    const instr = draft.trim();
    if (!instr) {
      message.warning(t(ENTER_INSTRUCTION_KEY, { defaultValue: 'Please describe what you want to change.' }));
      return;
    }
    stopRef.current?.();
    setLoading(true);
    setStreamingText('');
    streamingRef.current = '';
    setDraft('');

    // Optimistic: show user message immediately, don't wait for AI response.
    const userMsg: CodeChatMessage = { role: 'user', content: instr };
    setHistory(prev => [...prev, userMsg]);

    const stop = codeAssistApi.reviseStream(
      { code, instruction: instr, history: [...history, userMsg], locale: i18n.language },
      {
        onDelta: (delta) => {
          streamingRef.current += delta;
          setStreamingText(streamingRef.current);
        },
        onResult: (python) => {
          setLoading(false);
          const finalContent = streamingRef.current || python;
          setHistory(prev => [...prev, { role: 'assistant' as const, content: finalContent }]);
          streamingRef.current = '';
          setStreamingText('');
          const codeToApply = python || finalContent;
          if (codeToApply.trim()) {
            onApply(codeToApply);
            message.success(t(CODE_UPDATED_KEY, { defaultValue: 'Code updated.' }));
          }
        },
        onError: (e: unknown) => {
          setLoading(false);
          streamingRef.current = '';
          setStreamingText('');
          if ((e as unknown)?.code == null) {
            message.error(String((e as Error)?.message || e || t('common.unknownError', { defaultValue: 'Unknown error' })));
          }
        },
      },
    );
    stopRef.current = stop;
  };

  // Auto-scroll to bottom when new messages arrive.
  useEffect(() => {
    chatEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [history, streamingText]);

  return (
    <div style={{ border: '1px solid #f0f0f0', borderRadius: 6, padding: 8, background: '#fff', display: 'flex', flexDirection: 'column', height: '100%' }}>
      <Space style={{ marginBottom: 6, flex: 'none' }}>
        <RobotOutlined />
        <span>{t(AI_REVISE_TITLE_KEY, { defaultValue: 'AI assistant' })}</span>
        {loading && <span style={{ fontSize: 11, color: '#1677ff' }}>●</span>}
      </Space>
      <div style={{ flex: 1, overflow: 'auto', marginBottom: 6, minHeight: 0 }}>
        {history.length === 0 && !streamingText && (
          <div style={{ textAlign: 'center', padding: 20, color: '#8c8c8c', fontSize: 12 }}>
            {t('strategy.codeAssist.aiHint', { defaultValue: 'Describe the changes you want, e.g. "Add a 2% stop-loss" or "Replace SMA with EMA"' })}
          </div>
        )}
        {history.map((m, i) => (
          <div key={i} style={{
            margin: '6px 0', padding: '6px 10px', borderRadius: 6,
            background: m.role === 'user' ? '#e6f4ff' : '#f6ffed',
            fontSize: 12, whiteSpace: 'pre-wrap',
          }}>
            <b style={{ color: m.role === 'user' ? '#1677ff' : '#389e0d' }}>
              {m.role === 'user' ? t('common.you', { defaultValue: 'You' }) : 'AI'}
            </b>
            <div>{m.content}</div>
          </div>
        ))}
        {streamingText && (
          <div style={{
            margin: '6px 0', padding: '6px 10px', borderRadius: 6,
            background: '#f6ffed', fontSize: 12, whiteSpace: 'pre-wrap',
          }}>
            <b style={{ color: '#389e0d' }}>AI</b>
            <div>{streamingText}</div>
          </div>
        )}
        <div ref={chatEndRef} />
      </div>
      <TextArea rows={2} value={draft} onChange={(e) => setDraft(e.target.value)}
        placeholder={code.trim()
          ? t(REVISE_INPUT_PLACEHOLDER_KEY, { defaultValue: 'e.g. Replace SMA(20) with EMA(50) and add a 1% stop-loss.' })
          : t(GENERATE_PLACEHOLDER_KEY, { defaultValue: 'Describe a strategy, e.g. Bollinger Bands mean-reversion for EURUSD with 2% stop-loss' })}
      />
      <div style={{ marginTop: 6, textAlign: 'right' }}>
        <Button type="primary" icon={<SendOutlined />} loading={loading} onClick={() => { send(); }}>
          {t(REVISE_SEND_KEY, { defaultValue: 'Send to AI' })}
        </Button>
      </div>
    </div>
  );
};
