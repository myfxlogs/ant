import { useContext } from 'react';
import { ConnectContext } from './connectContext';

export function useConnect() {
  const context = useContext(ConnectContext);
  if (!context) throw new Error('useConnect must be used within ConnectProvider');
  return context;
}
