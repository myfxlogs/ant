import { createContext } from 'react';

export interface ConnectContextType {
  isConnected: boolean;
  connectionState: 'connecting' | 'connected' | 'disconnected';
}

export const ConnectContext = createContext<ConnectContextType | undefined>(undefined);
