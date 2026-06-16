import { useEffect, useRef, useState, useCallback, type ReactNode } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { useAuthStore } from '@/stores/authStore';
import { subscribeEvents, subscribeUserSummary } from '@/client/stream';
import { toCamelCase } from '@/adapters/dataAdapter';
import type { OrderUpdate, ProfitUpdate } from '@/adapters/dataAdapter';
import type { AccountStatusEvent } from '@/gen/ant/v1/stream_event_account_pb';
import {
  handleOrderUpdate,
  handleAccountStatus,
  handlePositionSnapshot,
} from '@/bridge/bridgeStreamEvents';
import { handleProfitUpdate, cleanupProfitBridge } from '@/bridge/bridgeProfitEvents';
import { handleUserSummary } from '@/bridge/bridgeUserSummary';
import { ConnectContext } from './connectContext';

/**
 * StreamProvider manages the SSE connection lifecycle and bridges stream events
 * into the TanStack Query cache. It replaces the old ConnectProvider + SSEQueryBridge
 * pair with a single provider that:
 *  - Subscribes/unsubscribes based on authentication state
 *  - Tracks real connection health (not an artificial timeout)
 *  - Bridges all SSE events to TanStack Query
 *  - Exposes connection state via ConnectContext
 *
 * Renders nothing beyond {children} — pure side-effect wrapper.
 */
export function StreamProvider({ children }: { children: ReactNode }) {
  const queryClient = useQueryClient();
  const { isAuthenticated } = useAuthStore();
  const [isConnected, setIsConnected] = useState(false);
  const [connectionState, setConnectionState] = useState<
    'connecting' | 'connected' | 'disconnected'
  >('disconnected');
  const unsubEventsRef = useRef<(() => void) | null>(null);
  const unsubSummaryRef = useRef<(() => void) | null>(null);
  const mountRef = useRef(false);

  const reconnect = useCallback(() => {
    setIsConnected(false);
    setConnectionState('disconnected');
    unsubEventsRef.current?.();
    unsubEventsRef.current = null;
    unsubSummaryRef.current?.();
    unsubSummaryRef.current = null;
    cleanupProfitBridge();
    // Re-subscription happens via the useEffect below reacting to state change.
  }, []);

  useEffect(() => {
    mountRef.current = true;

    if (!isAuthenticated) {
      unsubEventsRef.current?.();
      unsubEventsRef.current = null;
      unsubSummaryRef.current?.();
      unsubSummaryRef.current = null;
      cleanupProfitBridge();
      setIsConnected(false);
      setConnectionState('disconnected');
      return;
    }

    setConnectionState('connecting');

    // Subscribe to userSummary stream — fires on first event.
    if (!unsubSummaryRef.current) {
      unsubSummaryRef.current = subscribeUserSummary(
        (summary) => {
          const camel = toCamelCase<Record<string, unknown>>(summary);
          handleUserSummary(queryClient, camel as Parameters<typeof handleUserSummary>[1]);
          if (mountRef.current) {
            setIsConnected(true);
            setConnectionState('connected');
          }
        },
        () => {
          if (mountRef.current) {
            setIsConnected(false);
            setConnectionState('disconnected');
          }
        },
      );
    }

    // Subscribe to main events stream.
    if (!unsubEventsRef.current) {
      unsubEventsRef.current = subscribeEvents([], {
        onOrder: (order: OrderUpdate) => {
          handleOrderUpdate(queryClient, order);
          if (mountRef.current) {
            setIsConnected(true);
            setConnectionState('connected');
          }
        },
        onProfit: (profit: ProfitUpdate) => {
          handleProfitUpdate(queryClient, profit);
          if (mountRef.current) {
            setIsConnected(true);
            setConnectionState('connected');
          }
        },
        onStatus: (status: AccountStatusEvent) => {
          handleAccountStatus(queryClient, status);
        },
        onPositionSnapshot: (accountId: string, positions: OrderUpdate[]) => {
          handlePositionSnapshot(queryClient, accountId, positions);
        },
        onError: () => {
          unsubEventsRef.current = null;
          unsubSummaryRef.current?.();
          unsubSummaryRef.current = null;
          cleanupProfitBridge();
          if (mountRef.current) {
            setIsConnected(false);
            setConnectionState('disconnected');
          }
        },
      });
    }

    return () => {
      mountRef.current = false;
      // Don't unsubscribe on normal re-render — only on auth change or unmount.
    };
  }, [isAuthenticated, queryClient]);

  // Cleanup on unmount.
  useEffect(() => {
    return () => {
      unsubEventsRef.current?.();
      unsubSummaryRef.current?.();
      cleanupProfitBridge();
    };
  }, []);

  return (
    <ConnectContext.Provider value={{ isConnected, connectionState, reconnect }}>
      {children}
    </ConnectContext.Provider>
  );
}
