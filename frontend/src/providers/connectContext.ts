import { createContext } from 'react';

export interface ConnectContextType {
  isConnected: boolean;
  connectionState: 'connecting' | 'connected' | 'disconnected';
  reconnect: () => void;
}

export const ConnectContext = createContext<ConnectContextType | undefined>(undefined);
