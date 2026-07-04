import { useState, useRef, useCallback, useEffect, lazy, Suspense } from 'react';
import { Button, Drawer, Tag, Select, Tooltip } from 'antd';
import { HistoryOutlined, SettingOutlined, FileTextOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import StrategyList from './StrategyList';
import StrategyChatHistory from './StrategyChatHistory';
import AgentGenChat from './AgentGenChat';
import { useConversationHandlers } from './useConversationHandlers';
import type { Conversation, Template } from './strategyChatUtils';
import { aiApi } from '@/client/ai';
const AISettingsModal = lazy(() => import('@/pages/strategy/components/workspace/AISettingsModal'));
import { aiGatewayApi } from '@/client/aiGateway';
import { AI_GATEWAY_SETTINGS_KEY, NEW_CONVERSATION_KEY, SELECT_MODEL_KEY, SELECT_SYMBOL_KEY } from '@/gen/ant/v1/i18n/strategy_ai_chat_keys';

import type { ValidateExtendedResult } from '@/client/codeAssist';

interface Props { symbol?: string; timeframe?: string; sessionId?: string; accountId?: string; onApplyCode: (code: string) => void; onValidateResult?: (result: ValidateExtendedResult) => void; onRunBacktest?: () => void; backtestStatus?: string; }

export default function StrategyChat({ symbol, timeframe, sessionId, accountId, onApplyCode, onValidateResult, onRunBacktest, backtestStatus }: Props) {
  const { t } = useTranslation();
  const [modelOptions, setModelOptions] = useState<Array<{ value: string; label: string }>>([]);
  const [selectedModel, setSelectedModel] = useState('');
  const [templates, setTemplates] = useState<Template[]>([]);
  const [loadedTemplateId, setLoadedTemplateId] = useState('');
  const [aiSettingsOpen, setAiSettingsOpen] = useState(false);
  const [historyOpen, setHistoryOpen] = useState(false);
  const [strategiesOpen, setStrategiesOpen] = useState(false);
  const [conversations, setConversations] = useState<Conversation[]>([]);
  const [activeConvId, setActiveConvId] = useState('');
  const [editingConvId, setEditingConvId] = useState<string | null>(null);
  const [editTitle, setEditTitle] = useState('');
  const codeRef = useRef('');
  const [codeGenKey, setCodeGenKey] = useState(0);
  const bumpCodeGen = () => setCodeGenKey(k => k + 1);
  const hasSymbol = !!(symbol && timeframe);

  // Fetch model options
  useEffect(() => {
    (async () => {
      try { const r = await aiApi.getPrimary(); if (r.providerId) setSelectedModel(`${r.providerId}|${r.model || ''}`); } catch {}
      try { const list = await aiGatewayApi.listSystemModels(); setModelOptions(list.map(m => ({ value: `${m.providerId}|${m.modelName}`, label: `${m.displayName || m.modelName} (${m.providerId})` }))); } catch {}
    })();
  }, []);

  const fetchTemplates = async () => {
    try {
      const { strategyTemplateApi } = await import('@/client/strategy-schedules');
      const list = await strategyTemplateApi.list();
      setTemplates(list || []);
    } catch {}
  };
  useEffect(() => { fetchTemplates(); }, []);

  const fetchConversations = async () => {
    try { const list = await aiApi.listConversations(); setConversations(list.map(c => ({ id: c.id, title: c.title || t(NEW_CONVERSATION_KEY), created_at: c.createdAt?.toISOString() || '' }))); } catch {}
  };
  useEffect(() => { fetchConversations(); }, []);

  const noop = useCallback(() => {}, []);
  const {
    handleLoadTemplate, handleSendToAI, handleRenameTemplate, handleDeleteTemplate,
    handleSaveTemplate, handleNewConv, handleLoadConv,
    handleStartRename, handleCancelRename, handleConfirmRename, handleDeleteConv,
  } = useConversationHandlers({
    sessionId, onApplyCode, addMsg: noop, setMessages: noop, setTab: noop,
    templates, conversations, setConversations,
    activeConvId, setActiveConvId, editingConvId, setEditingConvId,
    editTitle, setEditTitle, planRef: useRef(''), codeRef, prevCodeRef: useRef(''),
    titleGeneratedRef: useRef(false), firstUserMsgRef: useRef(''), bumpCodeGen,
    fetchTemplates, fetchConversations,
  });

  const handleSendToAIWrapper = (code: string, name: string) => {
    setLoadedTemplateId(templates.find(t => t.code === code)?.id || '');
    handleSendToAI(code, name);
    setStrategiesOpen(false);
  };

  const handleLoadConvWrapper = (id: string) => {
    handleLoadConv(id);
    setHistoryOpen(false);
  };

  const handleNewConvWrapper = () => {
    handleNewConv();
    setHistoryOpen(false);
  };

  const symbolTag = hasSymbol
    ? <Tag color="blue" style={{ fontSize: 12, margin: 0 }}>{symbol} · {timeframe}</Tag>
    : <Tag color="warning" style={{ fontSize: 12, margin: 0 }}>{t(SELECT_SYMBOL_KEY)}</Tag>;

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      {/* Compact sub-bar: model selector + action buttons (no title — tab bar handles that) */}
      <div style={{ padding: '4px 8px', borderBottom: '1px solid var(--ant-color-border)', display: 'flex', alignItems: 'center', gap: 4, flexShrink: 0, background: 'var(--ant-color-bg-container)' }}>
        {symbolTag}
        {modelOptions.length > 0 && (
          <Select size="small" value={selectedModel || undefined}
            onChange={async (v) => { setSelectedModel(v); const [pid, model] = v.split('|'); try { await aiApi.setPrimary({ providerId: pid, model }); } catch {} }}
            style={{ width: 140, fontSize: 11 }} options={modelOptions} placeholder={t(SELECT_MODEL_KEY)} />
        )}
        <div style={{ flex: 1 }} />
        <Tooltip title={t('strategy.aiChat.historyTab', 'History')}>
          <Button size="small" type="text" icon={<HistoryOutlined />} onClick={() => setHistoryOpen(true)} />
        </Tooltip>
        <Tooltip title={t('strategy.aiChat.strategiesTab', 'Strategies')}>
          <Button size="small" type="text" icon={<FileTextOutlined />} onClick={() => setStrategiesOpen(true)} />
        </Tooltip>
        <Button size="small" type="text" icon={<SettingOutlined />} onClick={() => setAiSettingsOpen(true)} title={t(AI_GATEWAY_SETTINGS_KEY)} />
      </div>

      {/* Main content: AgentGenChat as the single unified interface */}
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
        <AgentGenChat symbol={symbol} timeframe={timeframe} onApply={onApplyCode} />
      </div>

      {/* History drawer */}
      <Drawer
        title={t('strategy.aiChat.historyTab', 'History')}
        open={historyOpen}
        onClose={() => setHistoryOpen(false)}
        width={360}
        styles={{ body: { padding: 0 } }}
      >
        <StrategyChatHistory
          conversations={conversations} activeConvId={activeConvId}
          editingConvId={editingConvId} editTitle={editTitle}
          onNewConv={handleNewConvWrapper} onLoadConv={handleLoadConvWrapper}
          onStartRename={handleStartRename} onConfirmRename={handleConfirmRename}
          onCancelRename={handleCancelRename} onDeleteConv={handleDeleteConv}
          onEditTitleChange={setEditTitle}
        />
      </Drawer>

      {/* Strategies drawer */}
      <Drawer
        title={t('strategy.aiChat.strategiesTab', 'Strategies')}
        open={strategiesOpen}
        onClose={() => setStrategiesOpen(false)}
        width={360}
        styles={{ body: { padding: 0 } }}
      >
        <StrategyList templates={templates} loadedId={loadedTemplateId}
          hasCode={!!codeRef.current} onLoad={handleLoadTemplate} onSave={handleSaveTemplate}
          onRename={handleRenameTemplate} onDelete={handleDeleteTemplate}
          onSendToAI={handleSendToAIWrapper}
        />
      </Drawer>

      <Suspense fallback={null}>
        <AISettingsModal open={aiSettingsOpen} onClose={() => setAiSettingsOpen(false)} />
      </Suspense>
    </div>
  );
}
