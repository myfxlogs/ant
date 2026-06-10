import { useState, useEffect } from 'react';
import { aiClient } from '@/client/connect';
import type { CodeChatMessage } from '@/client/codeAssist';

/**
 * Resolves the AI chat session on workspace mount and migrates the session key
 * when a strategy draft is saved (draft:* → strategy:<id>).
 */
export function useWorkspaceSession(
  userId: string | undefined,
  symbol: string,
  timeframe: string,
  lastSavedId: string,
) {
  const [sessionId, setSessionId] = useState('');
  const [chatHistory, setChatHistory] = useState<CodeChatMessage[]>([]);

  // Resolve AI session on mount — enables cross-device chat history sync.
  useEffect(() => {
    if (!userId) return;
    const strategyKey = `draft:${userId}:${symbol || ''}:${timeframe || ''}`;
    aiClient.resolveSession({ strategyKey }).then(res => {
      setSessionId(res.sessionId);
      setChatHistory(res.messages || []);
    }).catch(() => {
      // Session resolve is best-effort; AI chat works without persistence.
    });
  }, [userId, symbol, timeframe]);

  // Migrate session key when strategy is saved: draft:* → strategy:<id>
  useEffect(() => {
    if (!lastSavedId || !sessionId) return;
    const newKey = `strategy:${lastSavedId}`;
    aiClient.updateSessionStrategyKey({ sessionId, strategyKey: newKey }).then(() => {
      aiClient.resolveSession({ strategyKey: newKey }).then(res => {
        setSessionId(res.sessionId);
        setChatHistory(res.messages || []);
      }).catch(() => {});
    }).catch(() => {
      // Session key migration is best-effort; next resolve will pick up the new key.
    });
  }, [lastSavedId, sessionId]);

  return { sessionId, chatHistory };
}
