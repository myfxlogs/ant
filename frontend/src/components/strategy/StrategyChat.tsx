import { useState, useRef, useCallback, useEffect, lazy, Suspense } from 'react';
import { Button, Drawer, Tag, Select, Tooltip } from 'antd';
import { HistoryOutlined, SettingOutlined, FileTextOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import StrategyList from './StrategyList';
import StrategyChatHistory from './StrategyChatHistory';
import AgentGenChat from './AgentGenChat';
import { useConversationHandlers } from './useConversationHandlers';
import type { Conversation, Template } from './strategyChatUtils';
import type { ChatTurn } from './ChatHistory';
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
  const [activeConvId, setActiveConvId] = useState(crypto.randomUUID());
  const initialTurnsRef = useRef<ChatTurn[]>([]);
  const [editingConvId, setEditingConvId] = useState<string | null>(null);
  const [editTitle, setEditTitle] = useState('');
  const codeRef = useRef('');
  const bumpCodeGen = () => {};

  const hasSymbol = !!(symbol && timeframe);

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
    handleSendToAI(code, name);
    setStrategiesOpen(false);
  };

  // Load full conversation history — every message, not just code-bearing turns.
  // Aligns with Claude Code: resume shows the complete transcript.
  const handleLoadConvWrapper = async (id: string) => {
    const turns: ChatTurn[] = [];
    try {
      const detail = await aiApi.getConversation(id);
      const { AgentGenerateStrategyChunk } = await import('@/gen/ant/v1/agent_gateway_pb');
      for (const m of (detail.messages || [])) {
        if (m.role === 'user') {
          turns.push({ id: crypto.randomUUID(), role: 'user', message: m.content });
        } else {
          let turn: ChatTurn | null = null;
          if (m.turnData && m.turnData.length > 0) {
            try {
              const chunk = AgentGenerateStrategyChunk.fromBinary(m.turnData);
              turn = {
                id: crypto.randomUUID(), role: 'ai', message: '',
                phase: 'done' as const,
                streamText: chunk.delta || m.content,
                generatedCode: chunk.pythonSource || undefined,
                compileError: chunk.compileError || undefined,
                backtestError: chunk.backtestError || undefined,
                metrics: chunk.result?.success ? [
                  { label: 'Return', value: `${chunk.result.totalReturn.toFixed(1)}%`, positive: chunk.result.totalReturn >= 0 },
                  { label: 'Max DD', value: `${chunk.result.maxDrawdown.toFixed(1)}%`, positive: chunk.result.maxDrawdown <= 0 },
                  { label: 'Sharpe', value: chunk.result.sharpeRatio.toFixed(2), positive: chunk.result.sharpeRatio >= 1 },
                  { label: 'Win', value: `${chunk.result.winRate.toFixed(1)}%` },
                ] : undefined,
              } as ChatTurn;
            } catch { /* fall through to plain text */ }
          }
          if (!turn) {
            turn = { id: crypto.randomUUID(), role: 'ai', message: m.content, phase: 'done', streamText: m.content };
          }
          turns.push(turn);
        }
      }
    } catch {}
    initialTurnsRef.current = turns;
    setActiveConvId(id);
    setHistoryOpen(false);
  };

  const handleNewConvWrapper = () => {
    initialTurnsRef.current = [];
    handleNewConv();
    setHistoryOpen(false);
  };

  const symbolTag = hasSymbol
    ? <Tag color="blue" style={{ fontSize: 12, margin: 0 }}>{symbol} · {timeframe}</Tag>
    : <Tag color="warning" style={{ fontSize: 12, margin: 0 }}>{t(SELECT_SYMBOL_KEY)}</Tag>;

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
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

      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
        <AgentGenChat
          key={activeConvId}
          conversationId={activeConvId}
          symbol={symbol} timeframe={timeframe}
          onApply={onApplyCode}
          onDone={fetchConversations}
          initialTurnsRef={initialTurnsRef}
        />
      </div>

      <Drawer title={t('strategy.aiChat.historyTab', 'History')} open={historyOpen} onClose={() => setHistoryOpen(false)} width={360} styles={{ body: { padding: 0 } }}>
        <StrategyChatHistory
          conversations={conversations} activeConvId={activeConvId}
          editingConvId={editingConvId} editTitle={editTitle}
          onNewConv={handleNewConvWrapper} onLoadConv={handleLoadConvWrapper}
          onStartRename={handleStartRename} onConfirmRename={handleConfirmRename}
          onCancelRename={handleCancelRename} onDeleteConv={handleDeleteConv}
          onEditTitleChange={setEditTitle}
        />
      </Drawer>

      <Drawer title={t('strategy.aiChat.strategiesTab', 'Strategies')} open={strategiesOpen} onClose={() => setStrategiesOpen(false)} width={360} styles={{ body: { padding: 0 } }}>
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
