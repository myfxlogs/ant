export interface Notification {
  id: string;
  type: 'trade' | 'signal' | 'risk_alert' | 'strategy_execution' | 'strategy_version_update' | 'auto_fix_started' | 'auto_fix_stopped' | 'system';
  title: string;
  message: string;
  data?: Record<string, unknown>;
  read: boolean;
  created_at: string;
}

export interface NotificationState {
  notifications: Notification[];
  unreadCount: number;
  addNotification: (_notification: Omit<Notification, 'id' | 'read' | 'created_at'>) => void;
  markAsRead: (_id: string) => void;
  markAllAsRead: () => void;
  removeNotification: (_id: string) => void;
  clearAll: () => void;
  reset: () => void;
}
