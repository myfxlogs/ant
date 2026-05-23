import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import type { Notification, NotificationState } from '@/types/notification';

export const useNotificationStore = create<NotificationState>()(
  persist(
    (set) => ({
      notifications: [],
      unreadCount: 0,

      addNotification: (notification) => {
        const newNotification: Notification = {
          ...notification,
          id: `${Date.now()}-${Math.random().toString(36).substr(2, 9)}`,
          read: false,
          created_at: new Date().toISOString(),
        };
        set((state) => {
          const notifications = [newNotification, ...state.notifications].slice(0, 100);
          return {
            notifications,
            unreadCount: notifications.filter((n) => !n.read).length,
          };
        });
      },

      markAsRead: (id) => {
        set((state) => {
          const notifications = state.notifications.map((n) =>
            n.id === id ? { ...n, read: true } : n
          );
          return {
            notifications,
            unreadCount: notifications.filter((n) => !n.read).length,
          };
        });
      },

      markAllAsRead: () => {
        set((state) => ({
          notifications: state.notifications.map((n) => ({ ...n, read: true })),
          unreadCount: 0,
        }));
      },

      removeNotification: (id) => {
        set((state) => {
          const notifications = state.notifications.filter((n) => n.id !== id);
          return {
            notifications,
            unreadCount: notifications.filter((n) => !n.read).length,
          };
        });
      },

      clearAll: () => {
        set({ notifications: [], unreadCount: 0 });
      },
    }),
    {
      name: 'notification-storage',
      partialize: (state) => ({
        notifications: state.notifications,
        unreadCount: state.unreadCount,
      }),
    }
  )
);
