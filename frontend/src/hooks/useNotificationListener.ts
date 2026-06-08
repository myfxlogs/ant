import { useEffect, useRef } from 'react';
import { useNotificationStore } from '@/stores/notificationStore';
import { notificationStreamClient } from '@/client/connect';
import type { Notification as ProtoNotification } from '@/gen/ant/v1/notification_service_pb';
import type { Notification } from '@/types/notification';

function toNotification(pb: ProtoNotification): Notification {
  let data: Record<string, unknown> | undefined;
  try {
    data = pb.dataJson ? JSON.parse(pb.dataJson) : undefined;
  } catch {
    data = undefined;
  }
  return {
    id: pb.id,
    type: (pb.type as Notification['type']) || 'system',
    title: pb.title,
    message: pb.message,
    data,
    read: pb.isRead,
    created_at: pb.createdAt || pb.created_at || new Date().toISOString(),
  };
}

export function useNotificationListener() {
  const addNotification = useNotificationStore((state) => state.addNotification);
  const abortRef = useRef<AbortController | null>(null);

  useEffect(() => {
    const ctrl = new AbortController();
    abortRef.current = ctrl;

    (async () => {
      try {
        const stream = notificationStreamClient.streamNotifications(
          { unreadOnly: false },
          { signal: ctrl.signal },
        );
        for await (const pb of stream) {
          if (ctrl.signal.aborted) break;
          const notif = toNotification(pb);
          addNotification({
            type: notif.type,
            title: notif.title,
            message: notif.message,
            data: notif.data,
          });
        }
      } catch (e: unknown) {
        if ((e as { name?: string })?.name !== 'AbortError') {
          console.warn('Notification SSE stream ended:', e);
        }
      }
    })();

    return () => {
      ctrl.abort();
    };
  }, [addNotification]);
}
