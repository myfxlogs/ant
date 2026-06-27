import { useEffect, useRef } from 'react';
import { useNotificationStore } from '@/stores/notificationStore';
import { notificationStreamClient } from '@/client/connect';
import type { Notification as ProtoNotification } from '@/gen/ant/v1/notification_service_pb';
import type { Notification } from '@/types/notification';

/** Maximum consecutive transport-level failures before giving up. */
const TRANSPORT_FAILURE_CAP = 12;

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

function isAbortError(e: unknown): boolean {
  return (e as { name?: string })?.name === 'AbortError';
}

export function useNotificationListener() {
  const addNotification = useNotificationStore((state) => state.addNotification);
  const abortedRef = useRef(false);
  const abortRef = useRef<AbortController | null>(null);

  useEffect(() => {
    abortedRef.current = false;
    let transportFailStreak = 0;

    const runStream = async (retryCount = 0) => {
      if (abortedRef.current) return;

      const ctrl = new AbortController();
      abortRef.current = ctrl;

      try {
        const stream = notificationStreamClient.streamNotifications(
          { unreadOnly: false },
          { signal: ctrl.signal },
        );

        for await (const pb of stream) {
          if (abortedRef.current) break;
          transportFailStreak = 0;
          retryCount = 0;

          const notif = toNotification(pb);
          addNotification({
            type: notif.type,
            title: notif.title,
            message: notif.message,
            data: notif.data,
          });
        }

        // Stream ended cleanly — reconnect with backoff.
        if (!abortedRef.current) {
          const delay = Math.min(1000 * Math.pow(2, retryCount), 30000);
          setTimeout(() => runStream(retryCount + 1), delay);
        }
      } catch (e: unknown) {
        if (abortedRef.current || isAbortError(e)) return;

        transportFailStreak++;
        if (transportFailStreak >= TRANSPORT_FAILURE_CAP) {
          console.warn('[notif] transport failure cap reached, giving up');
          return;
        }

        console.warn(`[notif] stream error, retrying (${transportFailStreak}/${TRANSPORT_FAILURE_CAP}):`, e);

        const delay = Math.min(1000 * Math.pow(2, retryCount), 30000);
        setTimeout(() => runStream(retryCount + 1), delay);
      }
    };

    runStream();

    return () => {
      abortedRef.current = true;
      abortRef.current?.abort();
      abortRef.current = null;
    };
  }, [addNotification]);
}
