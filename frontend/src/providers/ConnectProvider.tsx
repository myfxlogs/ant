import { useEffect, useRef, useState, useCallback, type ReactNode } from 'react';
import { useAuthStore } from '@/stores/authStore';
import { ConnectContext } from './connectContext';

/**
 * ConnectProvider manages the SSE connection lifecycle:
 * - Subscribes/unsubscribes based on authentication state
 * - Provides connection state (isConnected, connectionState, reconnect)
 * - Data mapping is handled by SSEQueryBridge (writes to TanStack Query cache)
 */
export function ConnectProvider({ children }: { children: ReactNode }) {
  const mountedRef = useRef(false);
  const isConnectingRef = useRef(false);
  const connectSeqRef = useRef(0);
  const isConnectedRef = useRef(false);
  const [isConnected, setIsConnected] = useState(false);
  const [connectionState, setConnectionState] = useState<
    'connecting' | 'connected' | 'disconnected'
  >('disconnected');
  const [subVersion, setSubVersion] = useState(0);
  const connectTimeoutRef = useRef<number | null>(null);

  const connect = useCallback(() => {
    if (connectTimeoutRef.current || isConnectingRef.current) return;

    isConnectingRef.current = true;
    setConnectionState('connecting');
    const seq = ++connectSeqRef.current;

    connectTimeoutRef.current = window.setTimeout(() => {
      if (seq !== connectSeqRef.current) return;

      const { isAuthenticated, accessToken } = useAuthStore.getState();
      if (!isAuthenticated || !accessToken) {
        setIsConnected(false);
        isConnectedRef.current = false;
        setConnectionState('disconnected');
        isConnectingRef.current = false;
        if (connectTimeoutRef.current) {
          clearTimeout(connectTimeoutRef.current);
          connectTimeoutRef.current = null;
        }
        return;
      }

      setIsConnected(true);
      isConnectedRef.current = true;
      setConnectionState('connected');
      isConnectingRef.current = false;

      if (connectTimeoutRef.current) {
        clearTimeout(connectTimeoutRef.current);
        connectTimeoutRef.current = null;
      }
    }, 100);
  }, []);

  useEffect(() => {
    mountedRef.current = true;

    const unsubscribeAuth = useAuthStore.subscribe(
      (state, prevState) => {
        if (
          state.isAuthenticated !== prevState.isAuthenticated ||
          state.accessToken !== prevState.accessToken
        ) {
          setIsConnected(false);
          isConnectedRef.current = false;
          setConnectionState('disconnected');
          connect();
        }
      },
    );

    connect();

    // Initial poll: retry every 1s for first 10s until authenticated
    let initialPollCount = 0;
    const initialPollInterval = window.setInterval(() => {
      if (!mountedRef.current) {
        clearInterval(initialPollInterval);
        return;
      }
      initialPollCount++;
      const { isAuthenticated } = useAuthStore.getState();
      if (isAuthenticated) {
        clearInterval(initialPollInterval);
        connect();
      } else if (initialPollCount >= 10 || isConnectedRef.current) {
        clearInterval(initialPollInterval);
      }
    }, 1000);

    // Keep-alive: retry every 30s if disconnected
    const keepAliveInterval = window.setInterval(() => {
      if (
        mountedRef.current &&
        !isConnectedRef.current &&
        !isConnectingRef.current
      ) {
        const { isAuthenticated } = useAuthStore.getState();
        if (isAuthenticated) connect();
      }
    }, 30_000);

    return () => {
      mountedRef.current = false;
      unsubscribeAuth();
      clearInterval(initialPollInterval);
      clearInterval(keepAliveInterval);
      if (connectTimeoutRef.current) {
        clearTimeout(connectTimeoutRef.current);
        connectTimeoutRef.current = null;
      }
    };
  }, [connect, subVersion]);

  const reconnect = useCallback(() => {
    setIsConnected(false);
    isConnectedRef.current = false;
    setConnectionState('disconnected');
    setSubVersion((v) => v + 1);
  }, []);

  return (
    <ConnectContext.Provider value={{ isConnected, connectionState, reconnect }}>
      {children}
    </ConnectContext.Provider>
  );
}
