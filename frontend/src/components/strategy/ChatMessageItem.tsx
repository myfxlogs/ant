import { Button } from 'antd';
import { CodeOutlined, CopyOutlined, UserOutlined } from '@ant-design/icons';
import { Light as SyntaxHighlighter } from 'react-syntax-highlighter';
import python from 'react-syntax-highlighter/dist/esm/languages/hljs/python';
import { atomOneDark } from 'react-syntax-highlighter/dist/esm/styles/hljs';
import { useTranslation } from 'react-i18next';
import DiffView from './DiffView';
import type { ToolResult, BacktestMetricsMsg } from '@/gen/ant/v1/strategy_execution_pb';

SyntaxHighlighter.registerLanguage('python', python);
const iconStyle = { fontSize: 14 };

export type ChatMsg = { role: 'user' | 'ai'; text?: string; plan?: string; code?: string; prevCode?: string; toolResults?: ToolResult[]; metrics?: BacktestMetricsMsg | null };

export default function ChatMessageItem({ msg, copied, onCopy }: {
  msg: ChatMsg;
  copied: boolean;
  onCopy: (text: string) => void;
}) {
  const { t } = useTranslation();
  return (
    <div style={{ display: 'flex', gap: 8, alignItems: 'flex-start' }}>
      <div style={{ width: 28, height: 28, borderRadius: 14, flexShrink: 0, display: 'flex', alignItems: 'center', justifyContent: 'center', background: msg.role === 'user' ? '#1677ff' : '#52c41a', color: '#fff' }}>
        {msg.role === 'user' ? <UserOutlined style={iconStyle} /> : <span style={{ fontSize: 16 }}>🤖</span>}
      </div>
      <div style={{ flex: 1, minWidth: 0 }}>
        {msg.text && <div style={{ fontSize: 13, lineHeight: '22px', color: '#262626', whiteSpace: 'pre-wrap' }}>{msg.text}</div>}
        {msg.plan && (
          <div style={{ padding: '10px 12px', borderRadius: 6, fontSize: 12, background: '#f6ffed', border: '1px solid #b7eb8f', color: '#389e0d', whiteSpace: 'pre-wrap', lineHeight: '20px' }}>
            <div style={{ fontSize: 12, color: '#52c41a', marginBottom: 4, fontWeight: 600 }}>📋 {t('strategy.chat.executionPlan', { defaultValue: 'Execution Plan' })}</div>
            {msg.plan}
          </div>
        )}
        {msg.code && msg.prevCode && <DiffView oldCode={msg.prevCode} newCode={msg.code} />}
        {msg.code && (
          <div style={{ marginTop: 4, position: 'relative' }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '3px 10px', background: '#282c34', borderRadius: '6px 6px 0 0' }}>
              <span style={{ fontSize: 12, color: '#abb2bf' }}><CodeOutlined /> Python</span>
              <Button size="small" type="text" icon={<CopyOutlined />} onClick={() => onCopy(msg.code!)} style={{ color: '#abb2bf', fontSize: 12 }}>{copied ? t('common.copied', { defaultValue: 'Copied' }) : t('common.copy', { defaultValue: 'Copy' })}</Button>
            </div>
            <SyntaxHighlighter language="python" style={atomOneDark} showLineNumbers wrapLines customStyle={{ margin: 0, borderRadius: '0 0 6px 6px', fontSize: 12, padding: '8px 0', maxHeight: 300 }} lineNumberStyle={{ fontSize: 11, minWidth: '2em', color: '#636d83' }}>
              {msg.code}
            </SyntaxHighlighter>
            <div style={{ fontSize: 11, color: '#8c8c8c', marginTop: 4, textAlign: 'center' }}>
              ✅ {t('strategy.chat.codeGenerated', { defaultValue: 'Code generated. Use the buttons below to run strategy review and backtest.' })}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
