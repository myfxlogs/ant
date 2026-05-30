import { useEffect, useRef } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { useAuthStore } from '@/stores/authStore';
import { useConnect } from '@/providers/useConnect';
import { subscribeEvents, subscribeUserSummary } from '@/client/stream';
import { toCamelCase } from '@/adapters/dataAdapter';
import type { OrderUpdate, ProfitUpdate } from '@/adapters/dataAdapter';
import type { AccountStatusEvent } from '@/gen/ant/v1/stream_event_account_pb';
import type { StreamEvent } from '@/gen/ant/v1/stream_pb';
import {
  handleOrderUpdate,
  handleProfitUpdate,
  handleAccountStatus,
  handlePositionSnapshot,
} from './bridgeStreamEvents';
import { handleUserSummary } from './bridgeUserSummary';

/**
 * SSEQueryBridge subscribes to SSE event streams and writes data
 * directly into the TanStack Query cache, replacing the previous
 * Zustand-based store writes in ConnectProvider.
 *
 * Renders nothing — pure side-effect component.
 */
export function SSEQueryBridge() {
  const queryClient = useQueryClient();
  const { isAuthenticated } = useAuthStore();
  const { isConnected } = useConnect();
  const unsubEventsRef = useRef<(() => void) | null>(null);
  const unsubSummaryRef = useRef<(() => void) | null>(null);

  useEffect(() => {
    if (!isAuthenticated || !isConnected) {
      unsubEventsRef.current?.();
      unsubEventsRef.current = null;
      unsubSummaryRef.current?.();
      unsubSummaryRef.current = null;
      return;
    }

    // Subscribe to userSummary stream
    if (!unsubSummaryRef.current) {
      unsubSummaryRef.current = subscribeUserSummary((summary) => {
        const camel = toCamelCase<Record<string, unknown>>(summary);
        handleUserSummary(queryClient, camel as Parameters<typeof handleUserSummary>[1]);
      });
    }

    // Subscribe to main events stream
    if (!unsubEventsRef.current) {
      unsubEventsRef.current = subscribeEvents([], {
        onOrder: (order: OrderUpdate) => {
          handleOrderUpdate(queryClient, order);
        },
        onProfit: (profit: ProfitUpdate) => {
          handleProfitUpdate(queryClient, profit);
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
        },
      });
    }

    return () => {
      unsubEventsRef.current?.();
      unsubEventsRef.current = null;
      unsubSummaryRef.current?.();
      unsubSummaryRef.current = null;
    };
  }, [isAuthenticated, isConnected, queryClient]);

  return null;
}
