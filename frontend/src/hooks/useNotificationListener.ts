import { useEffect } from 'react';
import { useNotificationStore } from '@/stores/notificationStore';
import i18n from '@/i18n';

const STREAM_NOTIFICATION_EVENTS = [
  'strategy_execution',
  'risk_alert',
  'strategy_signal',
  'auto_trading',
];

export function useNotificationListener() {
  const addNotification = useNotificationStore((state) => state.addNotification);

  useEffect(() => {
    const handleStreamEvent = (event: CustomEvent) => {
      const data = event.detail;
      
      if (!data || !data.type) return;

      if (STREAM_NOTIFICATION_EVENTS.includes(data.type)) {
        switch (data.type) {
          case 'strategy_execution':
            addNotification({
              type: 'strategy_execution',
              title: i18n.t('notifications.stream.strategyExecution.title'),
              message:
                data.status === 'completed'
                  ? i18n.t('notifications.stream.strategyExecution.completed', { symbol: data.symbol, action: data.action })
                  : i18n.t('notifications.stream.strategyExecution.failed', {
                      error: data.error_message || i18n.t('common.unknown'),
                    }),
              data,
            });
            break;

          case 'risk_alert':
            addNotification({
              type: 'risk_alert',
              title: i18n.t('notifications.stream.riskAlert.title'),
              message:
                data.message || i18n.t('notifications.stream.riskAlert.fallback', { alertType: data.alert_type }),
              data,
            });
            break;

          case 'strategy_signal':
            addNotification({
              type: 'signal',
              title: i18n.t('notifications.stream.strategySignal.title'),
              message: i18n.t('notifications.stream.strategySignal.message', { symbol: data.symbol, signalType: data.signal_type }),
              data,
            });
            break;

          case 'auto_trading':
            addNotification({
              type: 'system',
              title: i18n.t('notifications.stream.autoTrading.title'),
              message: data.message || i18n.t('notifications.stream.autoTrading.fallback'),
              data,
            });
            break;
        }
      }
    };

    window.addEventListener('stream-notification', handleStreamEvent as EventListener);

    return () => {
      window.removeEventListener('stream-notification', handleStreamEvent as EventListener);
    };
  }, [addNotification]);
}
