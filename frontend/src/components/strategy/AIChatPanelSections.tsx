// AIChatPanelSections.tsx — Display-only subcomponents + shared helpers extracted from AIChatPanel.
// These are pure presentational components: messages, clarification, backtest metrics,
// pending code banner, and input bar. They receive data/events via props only.
// Also exports intent classification helpers shared with the parent.

import { Button, Space, Tag, Typography, Input } from 'antd';
import { SendOutlined, LoadingOutlined } from '@ant-design/icons';
import type { CodeChatMessage } from '@/client/codeAssist';

const { TextArea } = Input;

export interface BacktestMetrics {
  totalReturn?: number; sharpeRatio?: number; maxDrawdown?: number;
  winRate?: number; totalTrades?: number; profitFactor?: number;
}

// ── Shared helpers: intent classification ──

/** 4-mode intent classification — keyword tables match backend classifyIntent exactly. */
export function detectMode(msg: string, hasCode: boolean, hasBacktest = false): 'generate' | 'revise' | 'repair' | 'discuss' {
  if (hasBacktest) return 'generate';
  if (!hasCode) return 'generate';
  const lower = msg.toLowerCase();
  const repairKw = ['报错','error','错误','traceback','缺少参数','missing',
    '验证失败','syntax error','syntaxerror','undefined','未定义',
    '缺少 required','参数不足','attributeerror','typeerror'];
  if (repairKw.some(k => lower.includes(k))) return 'repair';
  const discussKw = ['为什么','什么意思','怎么样','对吗','分析','解释',
    'what','why','how','explain','对不对'];
  if (discussKw.some(k => lower.includes(k))) return 'discuss';
  return 'revise';
}

export function modeLabel(t: (k: string, d?: string) => string, mode: string): string {
  const map: Record<string, string> = {
    generate: 'strategy.gen.chat.generate',
    revise: 'strategy.gen.chat.revise',
    repair: 'strategy.gen.chat.repair',
    discuss: 'strategy.gen.chat.discuss',
  };
  return t(map[mode] || mode, mode);
}

export const MODE_COLORS: Record<string, string> = {
  generate: 'blue', revise: 'green', repair: 'orange', discuss: 'purple',
};

// ── ChatMessagesView — scrollable area with message history, analysis, advice, streaming, backtest metrics ──

interface ChatMessagesViewProps {
  history: CodeChatMessage[];
  analysisText: string;
  adviceText: string;
  streamText: string;
  backtestMetrics: BacktestMetrics | null;
  mode: string;
  t: (k: string) => string;
}

export function ChatMessagesView({ history, analysisText, adviceText, streamText, backtestMetrics, mode, t }: ChatMessagesViewProps) {
  return (
    <div style={{ maxHeight: 200, overflow: 'auto', marginBottom: 6 }}>
      {history.map((m, i) => (
        <div key={i} style={{
          margin: '6px 0', padding: '6px 10px', borderRadius: 6,
          background: m.role === 'user' ? '#e6f4ff' : '#f6ffed', fontSize: 12, whiteSpace: 'pre-wrap',
        }}>
          <b style={{ color: m.role === 'user' ? '#1677ff' : '#389e0d' }}>
            {m.role === 'user' ? 'You' : 'AI'}
          </b>
          <div>{m.content}</div>
        </div>
      ))}
      {analysisText && (
        <div style={{ margin: '6px 0', padding: '8px 10px', borderRadius: 6,
          background: '#f5f5f5', fontSize: 12, color: '#595959' }}>
          🔍 {analysisText}
        </div>
      )}
      {adviceText && (
        <div style={{ margin: '6px 0', padding: '8px 10px', borderRadius: 6,
          background: '#e6f4ff', fontSize: 12, color: '#1677ff' }}>
          💡 {adviceText}
        </div>
      )}
      {streamText && (
        <div style={{ margin: '6px 0', padding: '6px 10px', borderRadius: 6,
          background: '#f6ffed', fontSize: 12, whiteSpace: 'pre-wrap', fontFamily: 'monospace' }}>
          <b style={{ color: '#389e0d' }}>AI</b>
          <div>{streamText}</div>
        </div>
      )}
      {backtestMetrics && mode === 'done' && (
        <div style={{ margin: '6px 0', padding: '10px', borderRadius: 6,
          background: '#f6ffed', border: '1px solid #b7eb8f' }}>
          <Typography.Text strong style={{ fontSize: 11, marginBottom: 4, display: 'block' }}>
            {t('strategy.gen.feedback.heading')}
          </Typography.Text>
          <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap', marginBottom: 4 }}>
            {backtestMetrics.sharpeRatio != null && (
              <span style={{ fontSize: 10, background: '#fff', padding: '1px 5px', borderRadius: 3, border: '1px solid #e8e8e8' }}>
                <b>Sharpe</b> {backtestMetrics.sharpeRatio.toFixed(2)}
              </span>
            )}
            {backtestMetrics.maxDrawdown != null && (
              <span style={{ fontSize: 10, background: '#fff', padding: '1px 5px', borderRadius: 3, border: '1px solid #e8e8e8',
                color: backtestMetrics.maxDrawdown > 0.2 ? '#cf1322' : '#595959' }}>
                <b>Max DD</b> {(backtestMetrics.maxDrawdown * 100).toFixed(1)}%
              </span>
            )}
            {backtestMetrics.winRate != null && (
              <span style={{ fontSize: 10, background: '#fff', padding: '1px 5px', borderRadius: 3, border: '1px solid #e8e8e8' }}>
                <b>Win</b> {(backtestMetrics.winRate * 100).toFixed(0)}%
              </span>
            )}
            {backtestMetrics.totalTrades != null && (
              <span style={{ fontSize: 10, background: '#fff', padding: '1px 5px', borderRadius: 3, border: '1px solid #e8e8e8' }}>
                <b>Trades</b> {backtestMetrics.totalTrades}
              </span>
            )}
            {backtestMetrics.totalReturn != null && (
              <span style={{ fontSize: 10, background: '#fff', padding: '1px 5px', borderRadius: 3, border: '1px solid #e8e8e8',
                color: backtestMetrics.totalReturn > 0 ? '#389e0d' : '#cf1322' }}>
                <b>Return</b> {(backtestMetrics.totalReturn * 100).toFixed(1)}%
              </span>
            )}
          </div>
          <Typography.Text type="secondary" style={{ fontSize: 9 }}>
            {t('strategy.gen.feedback.placeholder')}
          </Typography.Text>
        </div>
      )}
    </div>
  );
}

// ── ChatClarificationView — shows clarification questions when LLM needs more details ──

interface ChatClarificationViewProps {
  questions: string[];
  clarifyRound: number;
  onAnswer: (answer: string) => void;
  onUseDefaults: () => void;
  t: (k: string, d?: string) => string;
}

export function ChatClarificationView({ questions, clarifyRound, onAnswer, onUseDefaults, t }: ChatClarificationViewProps) {
  return (
    <div style={{ padding: 12, background: '#fffbe6', borderRadius: 6, border: '1px solid #ffe58f' }}>
      <Typography.Text strong style={{ fontSize: 13 }}>
        {t('strategy.gen.clarifyTitle', '需要确认几个细节：')}
      </Typography.Text>
      <Space direction="vertical" size={6} style={{ width: '100%', marginTop: 8 }}>
        {questions.map((q, i) => (
          <Button key={i} block size="small" type="dashed" onClick={() => onAnswer(q)}
            disabled={clarifyRound >= 3}>{q}</Button>
        ))}
        {clarifyRound >= 3 && (
          <Button block size="small" type="primary" onClick={onUseDefaults}>
            {t('strategy.gen.useDefaults', '使用默认设置继续')}
          </Button>
        )}
      </Space>
    </div>
  );
}

// ── ChatPendingCodeBanner — shown when autoApply=false, AI returned code but user must review ──

interface ChatPendingCodeBannerProps {
  pendingCode: string;
  onApply: (code: string) => void;
  onDismiss: () => void;
  t: (k: string, d?: string) => string;
}

export function ChatPendingCodeBanner({ pendingCode, onApply, onDismiss, t }: ChatPendingCodeBannerProps) {
  return (
    <div style={{ padding: 8, marginBottom: 8, background: '#f6ffed', borderRadius: 6,
      border: '1px solid #b7eb8f', display: 'flex', alignItems: 'center', gap: 8 }}>
      <span style={{ fontSize: 12, flex: 1 }}>
        AI generated code — review the chat above before applying.
      </span>
      <Space size={6}>
        <Button size="small" onClick={() => { onApply(pendingCode); }}>
          Apply Code
        </Button>
        <Button size="small" onClick={onDismiss}>Dismiss</Button>
      </Space>
    </div>
  );
}

// ── ChatInputBar — input textarea + mode tag + send button ──

interface ChatInputBarProps {
  draft: string;
  busy: boolean;
  hasCode: boolean;
  hasBacktest: boolean;
  modeTag: string;
  modeColor: string;
  onDraftChange: (v: string) => void;
  onSend: () => void;
  t: (k: string) => string;
}

export function ChatInputBar({ draft, busy, hasCode, hasBacktest, modeTag, modeColor, onDraftChange, onSend, t }: ChatInputBarProps) {
  return (
    <>
      <TextArea rows={2} value={draft} onChange={e => onDraftChange(e.target.value)}
        disabled={busy}
        placeholder={
          !hasCode
            ? t('strategy.gen.placeholder', '描述你想创建的交易策略，例如："做一个 EURUSD 的布林带均值回归策略"')
            : t('strategy.codeAssist.reviseInputPlaceholder', 'e.g. Replace SMA(20) with EMA(50) and add a 1% stop-loss.')
        }
        onPressEnter={e => { if (!e.shiftKey) { e.preventDefault(); onSend(); } }}
        style={{ fontSize: 13, marginBottom: 8 }}
      />
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8 }}>
        <Tag color={modeColor}>{modeTag}</Tag>
        <Button type="primary" icon={<SendOutlined />} loading={busy}
          onClick={onSend} disabled={!draft.trim()}>
          {t('strategy.codeAssist.reviseSend', 'Send to AI')}
        </Button>
      </div>
    </>
  );
}
