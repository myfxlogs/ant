import { useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import { message } from 'antd';
import { aiApi } from '@/client/ai';
import {
  LOADED_STRATEGY_KEY, SAVED_STRATEGY_KEY, STRATEGY_NAME_PROMPT_KEY,
} from '@/gen/ant/v1/i18n/strategy_ai_chat_keys';
import type { Conversation, Template } from './strategyChatUtils';

interface UseConversationHandlersArgs {
  onApplyCode: (code: string) => void;
  templates: Template[];
  conversations: Conversation[];
  setConversations: React.Dispatch<React.SetStateAction<Conversation[]>>;
  activeConvId: string;
  setActiveConvId: (id: string) => void;
  setEditingConvId: (id: string | null) => void;
  editTitle: string;
  setEditTitle: (title: string) => void;
  codeRef: React.MutableRefObject<string>;
  fetchTemplates: () => Promise<void>;
  fetchConversations: () => Promise<void>;
}

export function useConversationHandlers({
  onApplyCode,
  templates, conversations, setConversations,
  activeConvId, setActiveConvId, setEditingConvId,
  editTitle, setEditTitle, codeRef,
  fetchTemplates, fetchConversations,
}: UseConversationHandlersArgs) {
  const { t } = useTranslation();

  const handleLoadTemplate = useCallback(async (id: string) => {
    const tpl = templates.find(t => t.id === id);
    if (tpl?.code) { onApplyCode(tpl.code); message.info(`${t(LOADED_STRATEGY_KEY)}: ${tpl.name}`); }
  }, [templates, onApplyCode, t]);

  const handleSendToAI = useCallback((code: string, name: string) => {
    codeRef.current = code;
    onApplyCode(code);
    message.info(`${t(LOADED_STRATEGY_KEY)}: ${name}`);
  }, [onApplyCode, t, codeRef]);

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
      await strategyTemplateApi.create({ name, description: name, code: c });
      fetchTemplates(); message.success(`${t(SAVED_STRATEGY_KEY)}: ${name}`);
    } catch {}
  }, [codeRef, t, fetchTemplates]);

  // Lazy-create: only generate a new UUID. The actual DB row is created
  // when the first message is sent (generator_agent.go CreateWithID).
  // This aligns with Claude Code: conversations only appear in history
  // after the first exchange, not on "New" button click.
  const handleNewConv = useCallback(() => {
    setActiveConvId(crypto.randomUUID());
    codeRef.current = '';
    fetchConversations();
  }, [setActiveConvId, codeRef, fetchConversations]);

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
    if (activeConvId === convId) { setActiveConvId(''); }
  }, [setConversations, activeConvId, setActiveConvId]);

  return {
    handleLoadTemplate, handleSendToAI, handleRenameTemplate, handleDeleteTemplate,
    handleSaveTemplate, handleNewConv,
    handleStartRename, handleCancelRename, handleConfirmRename, handleDeleteConv,
  };
}
