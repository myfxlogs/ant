import { useState, useRef, useCallback, useEffect, lazy, Suspense } from 'react';
import { Button, Input, Tag, Typography, Select, Segmented, Popconfirm } from 'antd';
import { SendOutlined, LoadingOutlined, SettingOutlined, HistoryOutlined, FileTextOutlined, EditOutlined, CheckOutlined, CloseOutlined, DeleteOutlined } from '@ant-design/icons';
import { conversate, type ConversateCallbacks } from '@/client/strategyPlan';
import { pythonStrategyApi } from '@/client/pythonStrategy';
import { isSucceededRun } from '@/pages/strategy/StrategyTemplatePage.utils';
import ChatMessageItem, { type ChatMsg } from './ChatMessageItem';
import WorkflowBar from './WorkflowBar';
import { aiApi } from '@/client/ai';
const AISettingsModal = lazy(() => import('@/pages/strategy/components/workspace/AISettingsModal'));
import { aiGatewayApi } from '@/client/aiGateway';
import type { ToolResult, BacktestMetricsMsg } from '@/gen/ant/v1/strategy_execution_pb';
import type { BacktestRunUpdate } from '@/gen/ant/v1/backtest_run_query_pb';

const { TextArea } = Input;
type TabKey = 'chat' | 'history' | 'strategies';
interface Props { symbol?: string; timeframe?: string; sessionId?: string; accountId?: string; onApplyCode: (code: string) => void; }

/** Ask the LLM to generate a short conversation title from the first user message. */
async function generateTitle(firstMsg: string): Promise<string> {
  try {
    const prompt = `你是一个标题生成器。根据用户的第一条消息，生成一个简短的对话标题（3-8个字）。只返回标题本身，不要加引号、句号或任何额外文字。\n\n用户消息: "${firstMsg}"\n\n标题:`;
    const result = await aiApi.chat({ message: prompt });
    const title = (result.message || '').replace(/^["'「『]|["'」』]$/g, '').trim();
    return title.slice(0, 30) || firstMsg.slice(0, 20);
  } catch {
    return firstMsg.slice(0, 20);
  }
}
export default function StrategyChat({ symbol, timeframe, sessionId, accountId, onApplyCode }: Props) {
  const [draft, setDraft] = useState('');
  const [busy, setBusy] = useState(false);
  const [messages, setMessages] = useState<ChatMsg[]>([]);
  const [metrics, setMetrics] = useState<BacktestMetricsMsg | null>(null);
  const [copied, setCopied] = useState(false);
  const [modelOptions, setModelOptions] = useState<Array<{ value: string; label: string }>>([]);
  const [selectedModel, setSelectedModel] = useState('');
  const [templates, setTemplates] = useState<Array<{ id: string; name: string; code: string }>>([]);
  const [loadedTemplateId, setLoadedTemplateId] = useState('');
  const [aiSettingsOpen, setAiSettingsOpen] = useState(false);
  const [conversations, setConversations] = useState<Array<{ id: string; title: string; created_at: string }>>([]);
  const [activeConvId, setActiveConvId] = useState('');
  const [tab, setTab] = useState<TabKey>('chat');
  const [editingConvId, setEditingConvId] = useState<string | null>(null);
  const [editTitle, setEditTitle] = useState('');
  const planRef = useRef(''), codeRef = useRef(''), prevCodeRef = useRef('');
  const metricsRef = useRef<BacktestMetricsMsg | null>(null);
  const abortRef = useRef<(() => void) | null>(null);
  const chatEndRef = useRef<HTMLDivElement>(null);
  const titleGeneratedRef = useRef(false);
  const firstUserMsgRef = useRef('');
  const hasSymbol = !!(symbol && timeframe);
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
  // Fetch conversation list
  const fetchConversations = async () => {
    try { const list = await aiApi.listConversations(); setConversations(list.map(c => ({ id: c.id, title: c.title || '新对话', created_at: c.createdAt?.toISOString() || '' }))); } catch {}
  };
  useEffect(() => { fetchConversations(); }, []);
  // Auto-greet
  const greeted = useRef(false);
  useEffect(() => {
    if (hasSymbol && !greeted.current && !busy && messages.length === 0) {
      greeted.current = true;
      firstUserMsgRef.current = `你好，介绍一下当前 ${symbol} ${timeframe} 的市场概况`;
      setBusy(true);
      addMsg('user', { text: firstUserMsgRef.current });
      abortRef.current = runConversate(firstUserMsgRef.current, '', true);
    }
  }, [hasSymbol, symbol, timeframe, sessionId, busy, messages.length]);

  useEffect(() => { chatEndRef.current?.scrollIntoView({ behavior: 'smooth' }); }, [messages]);
  const addMsg = (role: 'user' | 'ai', extra: Partial<ChatMsg>) => {
    setMessages(prev => [...prev, { role, ...extra }]);
  };
  // Try to auto-name the conversation after the first exchange
  const tryAutoName = useCallback(async (convId: string, firstMsg: string) => {
    if (titleGeneratedRef.current || !convId || !firstMsg) return;
    titleGeneratedRef.current = true;
    const title = await generateTitle(firstMsg);
    if (title) {
      try { await aiApi.updateConversationTitle(convId, title); } catch {}
      setConversations(prev => prev.map(c => c.id === convId ? { ...c, title } : c));
      setActiveConvId(convId);
    }
  }, []);
  const runConversate = (msg: string, curCode: string, isFirst: boolean) => {
    const curPlan = planRef.current, curPrevCode = prevCodeRef.current;
    const curMetrics = metricsRef.current;
    return conversate(
      { message: msg, conversationId: sessionId, symbol, timeframe, plan: curPlan, currentCode: curCode, backtestMetricsJson: curMetrics ? JSON.stringify(curMetrics) : '' },
      {
        onDelta: () => {},
        onPlan: (p) => { planRef.current = p; },
        onCode: (c) => { codeRef.current = c; prevCodeRef.current = curCode; },
        onPreviousCode: (c) => { prevCodeRef.current = c; },
        onToolCall: () => {},
        onToolResult: (tr: ToolResult) => {
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
        onError: (e) => { addMsg('ai', { text: '❌ ' + e }); setBusy(false); },
        onDone: () => {
          const p = planRef.current, c = codeRef.current, pc = prevCodeRef.current;
          const chatMsg: Partial<ChatMsg> = { role: 'ai' };
          let hasContent = false;
          if (p) { chatMsg.plan = p; hasContent = true; }
          if (c) { chatMsg.code = c; chatMsg.prevCode = pc; hasContent = true; }
          if (!hasContent) chatMsg.text = '收到，请继续。';
          addMsg('ai', chatMsg);
          setBusy(false);
          // Auto-name after first exchange
          if (isFirst && sessionId) {
            const firstMsg = firstUserMsgRef.current || msg;
            tryAutoName(sessionId, firstMsg);
          }
        },
      } satisfies ConversateCallbacks,
    );
  };
  const handleSend = useCallback(() => {
    const msg = draft.trim();
    if (!msg || busy) return;
    if (!hasSymbol) { addMsg('ai', { text: '请先选择交易品种和时间周期。' }); return; }
    const isFirst = messages.length === 0 && !titleGeneratedRef.current;
    if (isFirst) firstUserMsgRef.current = msg;
    setDraft(''); setBusy(true);
    addMsg('user', { text: msg });
    abortRef.current = runConversate(msg, codeRef.current, isFirst);
  }, [draft, busy, hasSymbol, sessionId, symbol, timeframe, messages.length]);
  const handleLoadTemplate = async (id: string) => {
    const tpl = templates.find(t => t.id === id);
    if (tpl?.code) { setLoadedTemplateId(id); onApplyCode(tpl.code); addMsg('ai', { text: `📂 已加载策略: ${tpl.name}` }); }
  };
  const handleSaveTemplate = async () => {
    const c = codeRef.current; if (!c) return;
    try {
      const { strategyTemplateApi } = await import('@/client/strategy-schedules');
      const name = prompt('策略名称:'); if (!name) return;
      await strategyTemplateApi.create({ name, code: c });
      fetchTemplates(); addMsg('ai', { text: `✅ 已保存策略: ${name}` });
    } catch {}
  };
  const handleNewConv = async () => {
    try {
      const conv = await aiApi.createConversation('新对话');
      setActiveConvId(conv.id); setMessages([]);
      planRef.current = ''; codeRef.current = ''; prevCodeRef.current = '';
      titleGeneratedRef.current = false; firstUserMsgRef.current = '';
      setTab('chat'); fetchConversations();
    } catch {}
  };
  const handleLoadConv = async (id: string) => {
    try {
      const detail = await aiApi.getConversation(id); setActiveConvId(id);
      setMessages((detail.messages || []).map(m => ({ role: m.role === 'user' ? 'user' : 'ai', text: m.content })));
      titleGeneratedRef.current = true; // prevent auto-rename on loaded conversations
      setTab('chat');
    } catch {}
  };
  const handleCopy = (text: string) => { navigator.clipboard.writeText(text); setCopied(true); setTimeout(() => setCopied(false), 2000); };
  // ── Conversation rename ──
  const handleStartRename = (convId: string, currentTitle: string) => {
    setEditingConvId(convId);
    setEditTitle(currentTitle);
  };
  const handleCancelRename = () => { setEditingConvId(null); setEditTitle(''); };
  const handleConfirmRename = async (convId: string) => {
    const title = editTitle.trim();
    if (title && title !== conversations.find(c => c.id === convId)?.title) {
      try { await aiApi.updateConversationTitle(convId, title); } catch {}
      setConversations(prev => prev.map(c => c.id === convId ? { ...c, title } : c));
    }
    setEditingConvId(null); setEditTitle('');
  };
  const handleDeleteConv = async (convId: string) => {
    try { await aiApi.deleteConversation(convId); } catch {}
    setConversations(prev => prev.filter(c => c.id !== convId));
    if (activeConvId === convId) { setActiveConvId(''); setMessages([]); }
  };
  const getPlaceholder = () => {
    if (!hasSymbol) return '请先在顶栏选择交易品种和周期';
    if (messages.length === 0) return '描述你想要的交易策略，例如：\"写一个基于RSI和MACD的趋势跟踪策略\"';
    const lastAi = [...messages].reverse().find(m => m.role === 'ai');
    if (lastAi?.code) return '描述你的修改需求，或继续讨论优化方案...';
    if (lastAi?.plan) return '你可以要求修改计划，或输入 \"生成代码\" 开始实现';
    return '继续描述你的策略需求...';
  };
  const symbolTag = hasSymbol
    ? <Tag color="blue" style={{ fontSize: 12, margin: 0 }}>{symbol} · {timeframe}</Tag>
    : <Tag color="warning" style={{ fontSize: 12, margin: 0 }}>选择品种</Tag>;

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%', background: '#fafbfc' }}>
      {/* Header */}
      <div style={{ padding: '8px 12px', borderBottom: '1px solid #e8e8e8', display: 'flex', alignItems: 'center', gap: 8, flexShrink: 0, background: '#fff' }}>
        <span style={{ fontSize: 16 }}>🤖</span>
        <Typography.Text strong style={{ fontSize: 14, color: '#262626' }}>Strategy Code</Typography.Text>
        {symbolTag}
        {modelOptions.length > 0 && (
          <Select size="small" value={selectedModel || undefined}
            onChange={async (v) => { setSelectedModel(v); const [pid, model] = v.split('|'); try { await aiApi.setPrimary({ providerId: pid, model }); } catch {} }}
            style={{ width: 160, fontSize: 12 }} options={modelOptions} placeholder="选择模型" />
        )}
        <Button size="small" type="text" icon={<SettingOutlined />} onClick={() => setAiSettingsOpen(true)} style={{ fontSize: 12 }} title="AI 网关设置" />
        <div style={{ flex: 1 }} />
        {busy && <LoadingOutlined style={{ color: '#1677ff' }} />}
      </div>

      {/* Tabs */}
      <div style={{ flexShrink: 0, padding: '6px 12px', background: '#fff', borderBottom: '1px solid #f0f0f0' }}>
        <Segmented value={tab} onChange={(v) => setTab(v as TabKey)}
          options={[{ label: '💬 对话', value: 'chat' }, { label: '📜 历史', value: 'history' }, { label: '📋 策略', value: 'strategies' }]}
          block style={{ fontSize: 12 }} />
      </div>

      {/* Tab content */}
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
        {tab === 'chat' && (<>
          <div style={{ flex: 1, overflowY: 'auto', padding: '12px 14px', display: 'flex', flexDirection: 'column', gap: 12 }}>
            {messages.length === 0 && !busy && (
              <div style={{ textAlign: 'center', color: '#8c8c8c', marginTop: 40, fontSize: 14 }}>
                <div style={{ fontSize: 32, marginBottom: 12 }}>🤖</div>
                告诉我你想创建什么样的交易策略
              </div>
            )}
            {messages.map((m, i) => <ChatMessageItem key={i} msg={m} copied={copied} onCopy={handleCopy} />)}
            {busy && messages.length > 0 && messages[messages.length - 1]?.role === 'user' && (
              <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
                <div style={{ width: 28, height: 28, borderRadius: 14, background: '#52c41a', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 14 }}>🤖</div>
                <LoadingOutlined style={{ color: '#1677ff' }} /><span style={{ fontSize: 11, color: '#8c8c8c' }}>思考中...</span>
              </div>
            )}
            <div ref={chatEndRef} />
          </div>
          <div style={{ padding: '10px 14px', borderTop: '1px solid #e8e8e8', flexShrink: 0, background: '#fff' }}>
            <div style={{ display: 'flex', gap: 8, alignItems: 'flex-end' }}>
              <TextArea value={draft} onChange={e => setDraft(e.target.value)}
                placeholder={getPlaceholder()}
                autoSize={{ minRows: 1, maxRows: 4 }}
                onPressEnter={e => { if (!e.shiftKey) { e.preventDefault(); handleSend(); } }}
                disabled={busy || !hasSymbol} style={{ fontSize: 13, borderRadius: 8 }} />
              <Button type="primary" icon={<SendOutlined />} onClick={handleSend} loading={busy} disabled={!draft.trim() || !hasSymbol} style={{ borderRadius: 8, flexShrink: 0 }} />
            </div>
            <div style={{ fontSize: 11, color: '#8c8c8c', marginTop: 4 }}>Enter 发送 · Shift+Enter 换行</div>
            <WorkflowBar
              codeRef={codeRef} busy={busy} hasSymbol={hasSymbol} accountId={accountId}
              symbol={symbol} timeframe={timeframe} templates={templates}
              addMsg={addMsg} setMetrics={setMetrics} fetchTemplates={fetchTemplates}
            />
          </div>
        </>)}

        {tab === 'history' && (
          <div style={{ flex: 1, overflowY: 'auto', padding: '12px 14px' }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 10 }}>
              <Typography.Text style={{ fontSize: 13, fontWeight: 600 }}><HistoryOutlined style={{ marginRight: 6 }} />历史对话</Typography.Text>
              <Button size="small" type="primary" onClick={handleNewConv}>+ 新建对话</Button>
            </div>
            {conversations.length > 0 ? (
              <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
                {conversations.map(conv => (
                  <div key={conv.id}
                    style={{ padding: '8px 10px', cursor: editingConvId === conv.id ? 'default' : 'pointer', borderRadius: 8, fontSize: 12,
                      background: conv.id === activeConvId ? '#e6f4ff' : '#fafafa',
                      border: conv.id === activeConvId ? '1px solid #91caff' : '1px solid #f0f0f0',
                      display: 'flex', justifyContent: 'space-between', alignItems: 'center', gap: 8, transition: 'all 0.15s' }}>
                    {editingConvId === conv.id ? (
                      <div style={{ display: 'flex', gap: 4, flex: 1, alignItems: 'center' }}>
                        <Input size="small" value={editTitle}
                          onChange={e => setEditTitle(e.target.value)}
                          onPressEnter={() => handleConfirmRename(conv.id)}
                          style={{ flex: 1, fontSize: 12 }} autoFocus />
                        <Button size="small" type="text" icon={<CheckOutlined />} onClick={() => handleConfirmRename(conv.id)}
                          style={{ color: '#52c41a', padding: '0 4px' }} />
                        <Button size="small" type="text" icon={<CloseOutlined />} onClick={handleCancelRename}
                          style={{ color: '#ff4d4f', padding: '0 4px' }} />
                      </div>
                    ) : (
                      <>
                        <span onClick={() => handleLoadConv(conv.id)}
                          style={{ color: '#262626', fontWeight: conv.id === activeConvId ? 600 : 400, flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                          {conv.id === activeConvId && <span style={{ color: '#1677ff', marginRight: 4 }}>●</span>}{conv.title}
                        </span>
                        <span style={{ color: '#8c8c8c', fontSize: 10, flexShrink: 0 }}>{conv.created_at?.slice(0, 10)}</span>
                        <Button size="small" type="text" icon={<EditOutlined style={{ fontSize: 11 }} />}
                          onClick={(e) => { e.stopPropagation(); handleStartRename(conv.id, conv.title); }}
                          style={{ color: '#8c8c8c', padding: '0 2px', flexShrink: 0 }}
                          title="重命名" />
                        <Popconfirm title="确定删除此对话？" okText="删除" cancelText="取消"
                          okButtonProps={{ danger: true }}
                          onConfirm={() => handleDeleteConv(conv.id)}>
                          <Button size="small" type="text" icon={<DeleteOutlined style={{ fontSize: 11 }} />}
                            onClick={(e) => e.stopPropagation()}
                            style={{ color: '#ff4d4f', padding: '0 2px', flexShrink: 0 }}
                            title="删除" />
                        </Popconfirm>
                      </>
                    )}
                  </div>
                ))}
              </div>
            ) : <div style={{ fontSize: 13, color: '#8c8c8c', textAlign: 'center', padding: '40px 0' }}>暂无历史对话</div>}
          </div>
        )}

        {tab === 'strategies' && (
          <div style={{ flex: 1, overflowY: 'auto', padding: '12px 14px' }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 10 }}>
              <Typography.Text style={{ fontSize: 13, fontWeight: 600 }}><FileTextOutlined style={{ marginRight: 6 }} />策略模板</Typography.Text>
              {codeRef.current && <Button size="small" type="primary" onClick={handleSaveTemplate}>保存当前策略</Button>}
            </div>
            {templates.length > 0 ? (
              <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
                {templates.map(tpl => (
                  <div key={tpl.id} onClick={() => handleLoadTemplate(tpl.id)}
                    style={{ padding: '10px 12px', cursor: 'pointer', borderRadius: 8, fontSize: 12,
                      background: tpl.id === loadedTemplateId ? '#f6ffed' : '#fafafa',
                      border: tpl.id === loadedTemplateId ? '1px solid #b7eb8f' : '1px solid #f0f0f0',
                      display: 'flex', justifyContent: 'space-between', alignItems: 'center', transition: 'all 0.15s' }}>
                    <span style={{ color: '#262626', fontWeight: tpl.id === loadedTemplateId ? 600 : 400 }}>
                      {tpl.id === loadedTemplateId && <span style={{ color: '#52c41a', marginRight: 4 }}>●</span>}{tpl.name}
                    </span>
                    <span style={{ color: '#8c8c8c', fontSize: 10 }}>{tpl.code ? `${tpl.code.split('\n').length} 行` : ''}</span>
                  </div>
                ))}
              </div>
            ) : <div style={{ fontSize: 13, color: '#8c8c8c', textAlign: 'center', padding: '40px 0' }}>暂无已保存的策略模板</div>}
          </div>
        )}
      </div>

      <Suspense fallback={null}>
        <AISettingsModal open={aiSettingsOpen} onClose={() => setAiSettingsOpen(false)} />
      </Suspense>
    </div>
  );
}
