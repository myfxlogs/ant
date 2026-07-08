import { useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import { aiApi } from '@/client/ai';
import {
  LOADED_STRATEGY_KEY, SAVED_STRATEGY_KEY, STRATEGY_NAME_PROMPT_KEY,
} from '@/gen/ant/v1/i18n/strategy_ai_chat_keys';
import type { ChatMsg } from './ChatMessageItem';
import type { Conversation, Template } from './strategyChatUtils';

interface UseConversationHandlersArgs {
  sessionId?: string;
  onApplyCode: (code: string) => void;
  addMsg: (role: 'user' | 'ai', extra: Partial<ChatMsg>) => void;
  setMessages: React.Dispatch<React.SetStateAction<ChatMsg[]>>;
  setTab: (tab: 'chat' | 'history' | 'strategies') => void;
  templates: Template[];
  setTemplates: React.Dispatch<React.SetStateAction<Template[]>>;
  conversations: Conversation[];
  setConversations: React.Dispatch<React.SetStateAction<Conversation[]>>;
  activeConvId: string;
  setActiveConvId: (id: string) => void;
  editingConvId: string | null;
  setEditingConvId: (id: string | null) => void;
  editTitle: string;
  setEditTitle: (title: string) => void;
  planRef: React.MutableRefObject<string>;
  codeRef: React.MutableRefObject<string>;
  prevCodeRef: React.MutableRefObject<string>;
  titleGeneratedRef: React.MutableRefObject<boolean>;
  firstUserMsgRef: React.MutableRefObject<string>;
  bumpCodeGen: () => void;
  fetchTemplates: () => Promise<void>;
  fetchConversations: () => Promise<void>;
}

export function useConversationHandlers({
  sessionId, onApplyCode, addMsg, setMessages, setTab,
  templates, setTemplates, conversations, setConversations,
  activeConvId, setActiveConvId, editingConvId, setEditingConvId,
  editTitle, setEditTitle, planRef, codeRef, prevCodeRef,
  titleGeneratedRef, firstUserMsgRef, bumpCodeGen,
  fetchTemplates, fetchConversations,
}: UseConversationHandlersArgs) {
  const { t } = useTranslation();

  const handleLoadTemplate = useCallback(async (id: string) => {
    const tpl = templates.find(t => t.id === id);
    if (tpl?.code) { onApplyCode(tpl.code); addMsg('ai', { text: `${t(LOADED_STRATEGY_KEY)}: ${tpl.name}` }); }
  }, [templates, onApplyCode, addMsg, t]);

  const handleSendToAI = useCallback((code: string, name: string) => {
    codeRef.current = code; prevCodeRef.current = '';
    bumpCodeGen(); onApplyCode(code);
    addMsg('ai', { text: `${t(LOADED_STRATEGY_KEY)}: ${name}`, code, prevCode: codeRef.current });
    setTab('chat');
  }, [onApplyCode, addMsg, t, codeRef, prevCodeRef, bumpCodeGen, setTab]);

  const handleRenameTemplate = useCallback(async (id: string, name: string) => {
    if (!name || name === templates.find(t => t.id === id)?.name) return;
    try {
      const { strategyTemplateApi } = await import('@/client/strategy-schedules');
      await strategyTemplateApi.update({ id, name });
      fetchTemplates();
    } catch {}
  }, [templates, fetchTemplates]);

  const handleDeleteTemplate = useCallback(async (id: string) => {
    try {
      const { strategyTemplateApi } = await import('@/client/strategy-schedules');
      await strategyTemplateApi.delete(id);
      fetchTemplates();
    } catch {}
  }, [fetchTemplates]);

  const handleSaveTemplate = useCallback(async () => {
    const c = codeRef.current; if (!c) return;
    try {
      const { strategyTemplateApi } = await import('@/client/strategy-schedules');
      const name = prompt(t(STRATEGY_NAME_PROMPT_KEY)); if (!name) return;
      await strategyTemplateApi.create({ name, code: c });
      fetchTemplates(); addMsg('ai', { text: `${t(SAVED_STRATEGY_KEY)}: ${name}` });
    } catch {}
  }, [codeRef, t, fetchTemplates, addMsg]);

  // Lazy-create: only generate a new UUID. The actual DB row is created
  // when the first message is sent (generator_agent.go CreateWithID).
  // This aligns with Claude Code: conversations only appear in history
  // after the first exchange, not on "New" button click.
  const handleNewConv = useCallback(() => {
    setActiveConvId(crypto.randomUUID()); setMessages([]);
    planRef.current = ''; codeRef.current = ''; prevCodeRef.current = '';
    titleGeneratedRef.current = false; firstUserMsgRef.current = '';
    setTab('chat'); fetchConversations();
  }, [setActiveConvId, setMessages, planRef, codeRef, prevCodeRef, titleGeneratedRef, firstUserMsgRef, setTab, fetchConversations]);

  const handleLoadConv = useCallback(async (id: string) => {
    try {
      const detail = await aiApi.getConversation(id); setActiveConvId(id);
      setMessages((detail.messages || []).map(m => ({ role: m.role === 'user' ? 'user' : 'ai', text: m.content })));
      titleGeneratedRef.current = true;
      setTab('chat');
    } catch {}
  }, [setActiveConvId, setMessages, titleGeneratedRef, setTab]);

  const handleStartRename = useCallback((convId: string, currentTitle: string) => {
    setEditingConvId(convId);
    setEditTitle(currentTitle);
  }, [setEditingConvId, setEditTitle]);

  const handleCancelRename = useCallback(() => {
    setEditingConvId(null); setEditTitle('');
  }, [setEditingConvId, setEditTitle]);

  const handleConfirmRename = useCallback(async (convId: string) => {
    const title = editTitle.trim();
    if (title && title !== conversations.find(c => c.id === convId)?.title) {
      try { await aiApi.updateConversationTitle(convId, title); } catch {}
      setConversations(prev => prev.map(c => c.id === convId ? { ...c, title } : c));
    }
    setEditingConvId(null); setEditTitle('');
  }, [editTitle, conversations, setConversations, setEditingConvId, setEditTitle]);

  const handleDeleteConv = useCallback(async (convId: string) => {
    try { await aiApi.deleteConversation(convId); } catch {}
    setConversations(prev => prev.filter(c => c.id !== convId));
    if (activeConvId === convId) { setActiveConvId(''); setMessages([]); }
  }, [setConversations, activeConvId, setActiveConvId, setMessages]);

  return {
    handleLoadTemplate, handleSendToAI, handleRenameTemplate, handleDeleteTemplate,
    handleSaveTemplate, handleNewConv, handleLoadConv,
    handleStartRename, handleCancelRename, handleConfirmRename, handleDeleteConv,
  };
}
