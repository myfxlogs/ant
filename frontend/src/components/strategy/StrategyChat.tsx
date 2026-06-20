import { useState, useRef, useCallback, useEffect } from 'react';
import { Button, Input, Tag, Typography, Space, Collapse, Select } from 'antd';
import { ThunderboltOutlined, SendOutlined, LoadingOutlined, CheckCircleOutlined, CloseCircleOutlined, CodeOutlined, CopyOutlined, RobotOutlined, UserOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next'
import { Light as SyntaxHighlighter } from 'react-syntax-highlighter';
import python from 'react-syntax-highlighter/dist/esm/languages/hljs/python';
import { atomOneDark } from 'react-syntax-highlighter/dist/esm/styles/hljs';
import { conversate, type ConversateCallbacks } from '@/client/strategyPlan';
import { pythonStrategyApi } from '@/client/pythonStrategy';
import { isSucceededRun } from '@/pages/strategy/StrategyTemplatePage.utils';
import DiffView from './DiffView';
import { aiApi } from '@/client/ai';
import { aiGatewayApi } from '@/client/aiGateway';
import type { ToolCall, ToolResult, BacktestMetricsMsg } from '@/gen/ant/v1/strategy_execution_pb';
import type { BacktestRunUpdate } from '@/gen/ant/v1/backtest_run_query_pb';

SyntaxHighlighter.registerLanguage('python', python);
const { TextArea } = Input;
const iconStyle = { fontSize: 14 };

type ChatMsg = { role: 'user' | 'ai'; text?: string; plan?: string; code?: string; prevCode?: string; toolResults?: ToolResult[]; metrics?: BacktestMetricsMsg | null };

interface Props { symbol?: string; timeframe?: string; sessionId?: string; onApplyCode: (code: string) => void; }

export default function StrategyChat({ symbol, timeframe, sessionId, onApplyCode }: Props) {
  const { t } = useTranslation();
  const [draft, setDraft] = useState('');
  const [busy, setBusy] = useState(false);
  const [messages, setMessages] = useState<ChatMsg[]>([]);
  const [metrics, setMetrics] = useState<BacktestMetricsMsg | null>(null);
  const [copied, setCopied] = useState(false);
  const [modelOptions, setModelOptions] = useState<Array<{ value: string; label: string }>>([]);
  const [selectedModel, setSelectedModel] = useState('');

  // Refs avoid closure staleness
  const planRef = useRef('');
  const codeRef = useRef('');
  const prevCodeRef = useRef('');
  const metricsRef = useRef<BacktestMetricsMsg | null>(null);
  const abortRef = useRef<(() => void) | null>(null);
  const chatEndRef = useRef<HTMLDivElement>(null);
  const hasSymbol = !!(symbol && timeframe);

  // Sync refs with state
  useEffect(() => { metricsRef.current = metrics; }, [metrics]);

  // Fetch model options
  useEffect(() => {
    (async () => {
      try { const r = await aiApi.getPrimary(); if (r.providerId) setSelectedModel(`${r.providerId}|${r.model || ''}`); } catch {}
      try { const list = await aiGatewayApi.listSystemModels(); setModelOptions(list.map(m => ({ value: `${m.providerId}|${m.modelName}`, label: `${m.displayName || m.modelName} (${m.providerId})` }))); } catch {}
    })();
  }, []);

  // Fetch saved templates
  const fetchTemplates = async () => {
    try {
      const { strategyTemplateApi } = await import('@/client/strategy-schedules');
      const list = await strategyTemplateApi.list();
      setTemplates(list.items || []);
    } catch {}
  };
  useEffect(() => { fetchTemplates(); }, []);

  useEffect(() => { chatEndRef.current?.scrollIntoView({ behavior: 'smooth' }); }, [messages]);

  const addMsg = (role: 'user' | 'ai', extra: Partial<ChatMsg>) => {
    setMessages(prev => [...prev, { role, ...extra }]);
  };

  const handleSend = useCallback(() => {
    const msg = draft.trim();
    if (!msg || busy) return;
    if (!hasSymbol) { addMsg('ai', { text: '请先选择交易品种和时间周期。' }); return; }
    setDraft(''); setBusy(true);
    addMsg('user', { text: msg });

    // Use refs for latest values
    const curPlan = planRef.current;
    const curCode = codeRef.current;
    const curPrevCode = prevCodeRef.current;
    const curMetrics = metricsRef.current;

    const abort = conversate(
      { message: msg, conversationId: sessionId, symbol, timeframe, plan: curPlan, currentCode: curCode, backtestMetricsJson: curMetrics ? JSON.stringify(curMetrics) : '' },
      {
        onDelta: () => {},
        onPlan: (p) => { planRef.current = p; },
        onCode: (c) => { codeRef.current = c; prevCodeRef.current = curCode; },
        onPreviousCode: (c) => { prevCodeRef.current = c; },
        onToolCall: () => {},
        onToolResult: (tr: ToolResult) => {
          // Show tool results inline in the conversation
          const icon = tr.success ? '✅' : '❌';
          const label = tr.name === 'compliance_check' ? '合规检查' : tr.name === 'backtest' ? '回测' : tr.name;
          const detail = tr.error || (tr.success ? '通过' : '');
          addMsg('ai', { text: `${icon} ${label}: ${detail}` });
          if (tr.name === 'backtest' && tr.outputJson) try {
            const out = JSON.parse(tr.outputJson);
            if (out.run_id) pythonStrategyApi.watchBacktestRun(out.run_id, (u: BacktestRunUpdate) => {
              if (isSucceededRun(u.run) && u.metrics) setMetrics({ totalReturn: u.metrics.totalReturn, sharpeRatio: u.metrics.sharpeRatio, maxDrawdown: u.metrics.maxDrawdown, winRate: u.metrics.winRate, totalTrades: u.metrics.totalTrades, profitFactor: u.metrics.profitFactor });
            });
          } catch {}
        },
        onError: (e) => {
          addMsg('ai', { text: '❌ ' + e });
          setBusy(false);
        },
        onDone: () => {
          // Build ONE message from all collected data
          const p = planRef.current;
          const c = codeRef.current;
          const pc = prevCodeRef.current;
          const chatMsg: Partial<ChatMsg> = { role: 'ai' };
          let hasContent = false;

          if (p) { chatMsg.plan = p; hasContent = true; }
          if (c) { chatMsg.code = c; chatMsg.prevCode = pc; hasContent = true; }
          if (!hasContent) { chatMsg.text = '收到，请继续。'; }

          // Only add if there's actual content
          addMsg('ai', chatMsg);
          setBusy(false);
        },
      } satisfies ConversateCallbacks,
    );
    abortRef.current = abort;
  }, [draft, busy, hasSymbol, sessionId, symbol, timeframe, t]);

  const handleLoadTemplate = async (id: string) => {
    const tpl = templates.find(t => t.id === id);
    if (tpl?.code) {
      setLoadedTemplateId(id);
      onApplyCode(tpl.code);
      addMsg('ai', { text: `📂 已加载策略: ${tpl.name}` });
    }
  };

  const handleSaveTemplate = async () => {
    const c = codeRef.current;
    if (!c) return;
    try {
      const { strategyTemplateApi } = await import('@/client/strategy-schedules');
      const name = prompt('策略名称:');
      if (!name) return;
      await strategyTemplateApi.create({ name, code: c });
      fetchTemplates();
      addMsg('ai', { text: `✅ 已保存策略: ${name}` });
    } catch {}
  };

  const handleCopy = (text: string) => { navigator.clipboard.writeText(text); setCopied(true); setTimeout(() => setCopied(false), 2000); };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%', background: '#fafbfc' }}>
      {/* Header */}
      <div style={{ padding: '8px 12px', borderBottom: '1px solid #e8e8e8', display: 'flex', alignItems: 'center', gap: 8, flexShrink: 0, background: '#fff' }}>
        <ThunderboltOutlined style={{ color: '#faad14', fontSize: 16 }} />
        <Typography.Text strong style={{ fontSize: 13, color: '#262626' }}>Strategy Code</Typography.Text>
        {hasSymbol ? <Tag color="blue" style={{ fontSize: 10, margin: 0 }}>{symbol} · {timeframe}</Tag> : <Tag color="warning" style={{ fontSize: 10, margin: 0 }}>选择品种</Tag>}
        {modelOptions.length > 0 && (
          <Select size="small" value={selectedModel || undefined}
            onChange={async (v) => { setSelectedModel(v); const [pid, model] = v.split('|'); try { await aiApi.setPrimary({ providerId: pid, model }); } catch {} }}
            style={{ width: 160, fontSize: 11 }} options={modelOptions} placeholder="选择模型" />
        )}
        {busy && <LoadingOutlined style={{ color: '#1677ff', marginLeft: 'auto' }} />}
        {templates.length > 0 && (
          <Select size="small" value={loadedTemplateId || undefined}
            onChange={(v) => handleLoadTemplate(v)}
            style={{ width: 140, fontSize: 11 }}
            options={templates.map(t => ({ value: t.id, label: t.name }))}
            placeholder="加载策略" allowClear
          />
        )}
        {codeRef.current && (
          <Button size="small" onClick={handleSaveTemplate} style={{ fontSize: 10 }}>保存</Button>
        )}
        <Button size="small" type="link" style={{ fontSize: 10, marginLeft: 'auto' }}
          onClick={async () => {
            try {
              const r = await aiGatewayApi.listSystemModels();
              setModelOptions(r.map(m => ({ value: `${m.providerId}|${m.modelName}`, label: `${m.displayName || m.modelName} (${m.providerId})` })));
            } catch {}
          }}>刷新模型</Button>
      </div>

      {/* Messages */}
      <div style={{ flex: 1, overflowY: 'auto', padding: '12px 14px', display: 'flex', flexDirection: 'column', gap: 12 }}>
        {messages.length === 0 && !busy && (
          <div style={{ textAlign: 'center', color: '#8c8c8c', marginTop: 40, fontSize: 13 }}>
            <RobotOutlined style={{ fontSize: 32, color: '#d9d9d9', display: 'block', margin: '0 auto 12' }} />
            告诉我你想创建什么样的交易策略
          </div>
        )}

        {messages.map((m, i) => (
          <div key={i} style={{ display: 'flex', gap: 8, alignItems: 'flex-start' }}>
            <div style={{ width: 28, height: 28, borderRadius: 14, flexShrink: 0, display: 'flex', alignItems: 'center', justifyContent: 'center', background: m.role === 'user' ? '#1677ff' : '#52c41a', color: '#fff' }}>
              {m.role === 'user' ? <UserOutlined style={iconStyle} /> : <RobotOutlined style={iconStyle} />}
            </div>
            <div style={{ flex: 1, minWidth: 0 }}>
              {m.text && <div style={{ fontSize: 12, lineHeight: '20px', color: '#262626', whiteSpace: 'pre-wrap' }}>{m.text}</div>}
              {m.plan && (
                <div style={{ padding: '10px 12px', borderRadius: 6, fontSize: 12, background: '#f6ffed', border: '1px solid #b7eb8f', color: '#389e0d', whiteSpace: 'pre-wrap', lineHeight: '20px' }}>
                  <div style={{ fontSize: 10, color: '#52c41a', marginBottom: 4, fontWeight: 600 }}>📋 执行计划</div>
                  {m.plan}
                </div>
              )}
              {m.code && m.prevCode && <DiffView oldCode={m.prevCode} newCode={m.code} />}
              {m.code && (
                <div style={{ marginTop: 4, position: 'relative' }}>
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '3px 10px', background: '#282c34', borderRadius: '6px 6px 0 0' }}>
                    <span style={{ fontSize: 10, color: '#abb2bf' }}><CodeOutlined /> Python</span>
                    <Button size="small" type="text" icon={<CopyOutlined />} onClick={() => handleCopy(m.code!)} style={{ color: '#abb2bf', fontSize: 11 }}>{copied ? '已复制' : '复制'}</Button>
                  </div>
                  <SyntaxHighlighter language="python" style={atomOneDark} showLineNumbers wrapLines customStyle={{ margin: 0, borderRadius: '0 0 6px 6px', fontSize: 11, padding: '8px 0', maxHeight: 300 }} lineNumberStyle={{ fontSize: 10, minWidth: '2em', color: '#636d83' }}>
                    {m.code}
                  </SyntaxHighlighter>
                  <div style={{ fontSize: 10, color: '#8c8c8c', marginTop: 4, textAlign: 'center' }}>
                    ✅ 代码已生成，合规检查和回测将自动运行。你可以继续讨论修改。
                  </div>
                </div>
              )}
            </div>
          </div>
        ))}

        {busy && messages.length > 0 && messages[messages.length - 1]?.role === 'user' && (
          <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
            <div style={{ width: 28, height: 28, borderRadius: 14, background: '#52c41a', display: 'flex', alignItems: 'center', justifyContent: 'center' }}><RobotOutlined style={{ color: '#fff', fontSize: 14 }} /></div>
            <LoadingOutlined style={{ color: '#1677ff' }} /><span style={{ fontSize: 11, color: '#8c8c8c' }}>思考中...</span>
          </div>
        )}
        <div ref={chatEndRef} />
      </div>

      {/* Input */}
      <div style={{ padding: '10px 14px', borderTop: '1px solid #e8e8e8', flexShrink: 0, background: '#fff' }}>
        <div style={{ display: 'flex', gap: 8, alignItems: 'flex-end' }}>
          <TextArea value={draft} onChange={e => setDraft(e.target.value)}
            placeholder={hasSymbol ? '描述你的策略需求，或继续对话...' : '请先在顶栏选择交易品种和周期'}
            autoSize={{ minRows: 1, maxRows: 4 }}
            onPressEnter={e => { if (!e.shiftKey) { e.preventDefault(); handleSend(); } }}
            disabled={busy || !hasSymbol} style={{ fontSize: 12, borderRadius: 8 }} />
          <Button type="primary" icon={<SendOutlined />} onClick={handleSend} loading={busy} disabled={!draft.trim() || !hasSymbol} style={{ borderRadius: 8, flexShrink: 0 }} />
        </div>
        <div style={{ fontSize: 10, color: '#8c8c8c', marginTop: 4 }}>Enter 发送 · Shift+Enter 换行</div>
      </div>
    </div>
  );
}
