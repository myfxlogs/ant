import React, { useEffect, useMemo, useRef, useState } from 'react';
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
    const userMsg = instr;
    setDraft('');

    const stop = codeAssistApi.reviseStream(
      { code, instruction: instr, history, locale: i18n.language },
      {
        onDelta: (delta) => { setStreamingText((prev) => prev + delta); },
        onResult: (python) => {
          setLoading(false);
          setHistory([
            ...history,
            { role: 'user' as const, content: userMsg },
            { role: 'assistant' as const, content: streamingRef.current || python },
          ]);
          streamingRef.current = '';
          setStreamingText('');
          if (python) {
            onApply(python);
            message.success(t(CODE_UPDATED_KEY, { defaultValue: 'Code updated.' }));
          }
        },
        onError: (e: unknown) => {
          setLoading(false);
          streamingRef.current = '';
          setStreamingText('');
          message.error(String((e as Error)?.message || e || 'failed'));
        },
      },
    );
    stopRef.current = stop;
  };

  const messagesView = useMemo(
    () => history.map((m, i) => (
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
    )),
    [history, t],
  );

  return (
    <div style={{ border: '1px solid #f0f0f0', borderRadius: 6, padding: 8, background: '#fff' }}>
      <Space style={{ marginBottom: 6 }}>
        <RobotOutlined />
        <span>{t(AI_REVISE_TITLE_KEY, { defaultValue: 'AI assistant' })}</span>
      </Space>
      <div style={{ maxHeight: 200, overflow: 'auto', marginBottom: 6 }}>
        {messagesView}
        {streamingText && (
          <div style={{
            margin: '6px 0', padding: '6px 10px', borderRadius: 6,
            background: '#f6ffed', fontSize: 12, whiteSpace: 'pre-wrap',
          }}>
            <b style={{ color: '#389e0d' }}>AI</b>
            <div>{streamingText}</div>
          </div>
        )}
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
