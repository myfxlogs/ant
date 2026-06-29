import { useState, useRef, useCallback, useEffect, lazy, Suspense } from 'react';
import { Button, Input, Tag, Typography, Select, Segmented } from 'antd';
import { SendOutlined, LoadingOutlined, SettingOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import ChatMessageItem, { type ChatMsg } from './ChatMessageItem';
import WorkflowBar from './WorkflowBar';
import StrategyList from './StrategyList';
import StrategyChatHistory from './StrategyChatHistory';
import { useConversationHandlers } from './useConversationHandlers';
import { generateTitle, runConversate as runConversateFn, type TabKey, type Conversation, type Template } from './strategyChatUtils';
import { aiApi } from '@/client/ai';
const AISettingsModal = lazy(() => import('@/pages/strategy/components/workspace/AISettingsModal'));
import { aiGatewayApi } from '@/client/aiGateway';
import type { BacktestMetricsMsg } from '@/gen/ant/v1/strategy_execution_pb';
import { AI_GATEWAY_SETTINGS_KEY, CHAT_TAB_KEY, INPUT_HINT_KEY, NEW_CONVERSATION_KEY, PLACEHOLDER_CONTINUE_KEY, PLACEHOLDER_DESCRIBE_STRATEGY_KEY, PLACEHOLDER_MODIFY_KEY, PLACEHOLDER_REVISE_PLAN_KEY, PLACEHOLDER_START_KEY, SELECT_MODEL_KEY, SELECT_SYMBOL_KEY, STRATEGIES_TAB_KEY, THINKING_KEY } from '@/gen/ant/v1/i18n/strategy_ai_chat_keys';

const { TextArea } = Input;
import type { ValidateExtendedResult } from '@/client/codeAssist';

interface Props { symbol?: string; timeframe?: string; sessionId?: string; accountId?: string; onApplyCode: (code: string) => void; onValidateResult?: (result: ValidateExtendedResult) => void; onRunBacktest?: () => void; backtestStatus?: string; }

export default function StrategyChat({ symbol, timeframe, sessionId, accountId, onApplyCode, onValidateResult, onRunBacktest, backtestStatus }: Props) {
  const { t } = useTranslation();
  const [draft, setDraft] = useState('');
  const [busy, setBusy] = useState(false);
  const [messages, setMessages] = useState<ChatMsg[]>([]);
  const [metrics, setMetrics] = useState<BacktestMetricsMsg | null>(null);
  const [copied, setCopied] = useState(false);
  const [modelOptions, setModelOptions] = useState<Array<{ value: string; label: string }>>([]);
  const [selectedModel, setSelectedModel] = useState('');
  const [templates, setTemplates] = useState<Template[]>([]);
  const [loadedTemplateId, setLoadedTemplateId] = useState('');
  const [aiSettingsOpen, setAiSettingsOpen] = useState(false);
  const [conversations, setConversations] = useState<Conversation[]>([]);
  const [activeConvId, setActiveConvId] = useState('');
  const [tab, setTab] = useState<TabKey>('chat');
  const [editingConvId, setEditingConvId] = useState<string | null>(null);
  const [editTitle, setEditTitle] = useState('');
  const planRef = useRef(''), codeRef = useRef(''), prevCodeRef = useRef('');
  const [codeGenKey, setCodeGenKey] = useState(0);
  const bumpCodeGen = () => setCodeGenKey(k => k + 1);
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
      setTemplates(list || []);
    } catch {}
  };
  useEffect(() => { fetchTemplates(); }, []);
  // Fetch conversation list
  const fetchConversations = async () => {
    try { const list = await aiApi.listConversations(); setConversations(list.map(c => ({ id: c.id, title: c.title || t(NEW_CONVERSATION_KEY), created_at: c.createdAt?.toISOString() || '' }))); } catch {}
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
  const runConversate = (msg: string, curCode: string, isFirst: boolean) =>
    runConversateFn({
      msg, curCode, isFirst, sessionId, symbol, timeframe,
      planRef, codeRef, prevCodeRef, metricsRef, firstUserMsgRef,
      bumpCodeGen, addMsg, setMetrics, setBusy, tryAutoName, t,
    });
  const handleSend = useCallback(() => {
    const msg = draft.trim();
    if (!msg || busy) return;
const isFirst = messages.length === 0 && !titleGeneratedRef.current;
    if (isFirst) firstUserMsgRef.current = msg;
    setDraft(''); setBusy(true);
    addMsg('user', { text: msg });
    abortRef.current = runConversate(msg, codeRef.current, isFirst);
  }, [draft, busy, hasSymbol, sessionId, symbol, timeframe, messages.length]);
  const {
    handleLoadTemplate, handleSendToAI, handleRenameTemplate, handleDeleteTemplate,
    handleSaveTemplate, handleNewConv, handleLoadConv,
    handleStartRename, handleCancelRename, handleConfirmRename, handleDeleteConv,
  } = useConversationHandlers({
    sessionId, onApplyCode, addMsg, setMessages, setTab,
    templates, conversations, setConversations,
    activeConvId, setActiveConvId, editingConvId, setEditingConvId,
    editTitle, setEditTitle, planRef, codeRef, prevCodeRef,
    titleGeneratedRef, firstUserMsgRef, bumpCodeGen,
    fetchTemplates, fetchConversations,
  });
  const handleCopy = (text: string) => { navigator.clipboard.writeText(text); setCopied(true); setTimeout(() => setCopied(false), 2000); };
  const handleSendToAIWrapper = (code: string, name: string) => {
    setLoadedTemplateId(templates.find(t => t.code === code)?.id || '');
    handleSendToAI(code, name);
  };
  const getPlaceholder = () => {
    if (!hasSymbol && messages.length === 0) return t(PLACEHOLDER_START_KEY);
    if (messages.length === 0) return t(PLACEHOLDER_DESCRIBE_STRATEGY_KEY);
    const lastAi = [...messages].reverse().find(m => m.role === 'ai');
    if (lastAi?.code) return t(PLACEHOLDER_MODIFY_KEY);
    if (lastAi?.plan) return t(PLACEHOLDER_REVISE_PLAN_KEY);
    return t(PLACEHOLDER_CONTINUE_KEY);
  };
  const symbolTag = hasSymbol
    ? <Tag color="blue" style={{ fontSize: 12, margin: 0 }}>{symbol} · {timeframe}</Tag>
    : <Tag color="warning" style={{ fontSize: 12, margin: 0 }}>{t(SELECT_SYMBOL_KEY)}</Tag>;

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
            style={{ width: 160, fontSize: 12 }} options={modelOptions} placeholder={t(SELECT_MODEL_KEY)} />
        )}
        <Button size="small" type="text" icon={<SettingOutlined />} onClick={() => setAiSettingsOpen(true)} style={{ fontSize: 12 }} title={t(AI_GATEWAY_SETTINGS_KEY)} />
        <div style={{ flex: 1 }} />
        {busy && <LoadingOutlined style={{ color: '#1677ff' }} />}
      </div>

      {/* Tabs */}
      <div style={{ flexShrink: 0, padding: '6px 12px', background: '#fff', borderBottom: '1px solid #f0f0f0' }}>
        <Segmented value={tab} onChange={(v) => setTab(v as TabKey)}
          options={[{ label: t(CHAT_TAB_KEY), value: 'chat' }, { label: t(HISTORY_TAB_KEY), value: 'history' }, { label: t(STRATEGIES_TAB_KEY), value: 'strategies' }]}
          block style={{ fontSize: 12 }} />
      </div>

      {/* Tab content */}
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
        {tab === 'chat' && (<>
          <div style={{ flex: 1, overflowY: 'auto', padding: '12px 14px', display: 'flex', flexDirection: 'column', gap: 12 }}>
            {messages.length === 0 && !busy && (
              <div style={{ textAlign: 'center', color: '#8c8c8c', marginTop: 40, fontSize: 14 }}>
                <div style={{ fontSize: 32, marginBottom: 12 }}>🤖</div>
                {t(PLACEHOLDER_DESCRIBE_STRATEGY_KEY)}
              </div>
            )}
            {messages.map((m, i) => <ChatMessageItem key={i} msg={m} copied={copied} onCopy={handleCopy} />)}
            {busy && messages.length > 0 && messages[messages.length - 1]?.role === 'user' && (
              <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
                <div style={{ width: 28, height: 28, borderRadius: 14, background: '#52c41a', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 14 }}>🤖</div>
                <LoadingOutlined style={{ color: '#1677ff' }} /><span style={{ fontSize: 11, color: '#8c8c8c' }}>{t(THINKING_KEY)}</span>
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
                disabled={busy} style={{ fontSize: 13, borderRadius: 8 }} />
              <Button type="primary" icon={<SendOutlined />} onClick={handleSend} loading={busy} disabled={!draft.trim() || busy} style={{ borderRadius: 8, flexShrink: 0 }} />
            </div>
            <div style={{ fontSize: 11, color: '#8c8c8c', marginTop: 4 }}>{t(INPUT_HINT_KEY)}</div>
            <WorkflowBar
              codeRef={codeRef} busy={busy} hasSymbol={hasSymbol} accountId={accountId}
              symbol={symbol} timeframe={timeframe} templates={templates} codeGenKey={codeGenKey}
              addMsg={addMsg} fetchTemplates={fetchTemplates}
              onValidateResult={onValidateResult}
              onRunBacktest={onRunBacktest}
              backtestStatus={backtestStatus}
            />
          </div>
        </>)}

        {tab === 'history' && (
          <StrategyChatHistory
            conversations={conversations} activeConvId={activeConvId}
            editingConvId={editingConvId} editTitle={editTitle}
            onNewConv={handleNewConv} onLoadConv={handleLoadConv}
            onStartRename={handleStartRename} onConfirmRename={handleConfirmRename}
            onCancelRename={handleCancelRename} onDeleteConv={handleDeleteConv}
            onEditTitleChange={setEditTitle}
          />
        )}

        {tab === 'strategies' && (
          <StrategyList templates={templates} loadedId={loadedTemplateId}
            hasCode={!!codeRef.current} onLoad={handleLoadTemplate} onSave={handleSaveTemplate}
            onRename={handleRenameTemplate} onDelete={handleDeleteTemplate}
            onSendToAI={handleSendToAIWrapper} />
        )}
      </div>

      <Suspense fallback={null}>
        <AISettingsModal open={aiSettingsOpen} onClose={() => setAiSettingsOpen(false)} />
      </Suspense>
    </div>
  );
}
