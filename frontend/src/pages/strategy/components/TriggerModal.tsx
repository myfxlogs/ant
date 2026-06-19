import { Button, Card, Descriptions, Modal, Popconfirm, Space, Typography } from 'antd';
import { useTranslation } from 'react-i18next'
import { TRIGGER_MODAL_ACTIONS_CONFIRM_ORDER_KEY, TRIGGER_MODAL_ACTIONS_RERUN_KEY, TRIGGER_MODAL_CARDS_LOGS_KEY, TRIGGER_MODAL_CARDS_SIGNAL_KEY, TRIGGER_MODAL_CONFIRM_ORDER_OK_KEY, TRIGGER_MODAL_CONFIRM_ORDER_TITLE_KEY, TRIGGER_MODAL_EMPTY_LOGS_KEY, TRIGGER_MODAL_EMPTY_SIGNAL_KEY, TRIGGER_MODAL_MESSAGES_SIGNAL_NOT_ORDERABLE_KEY, TRIGGER_MODAL_SUMMARY_ACCOUNT_KEY, TRIGGER_MODAL_SUMMARY_SCHEDULE_NAME_KEY, TRIGGER_MODAL_SUMMARY_SYMBOL_KEY, TRIGGER_MODAL_SUMMARY_TIMEFRAME_KEY, TRIGGER_MODAL_TITLE_KEY } from '@/gen/ant/v1/i18n/strategy_schedules_keys';

;

const { Text } = Typography;

type Props = {
  open: boolean;
  triggering: boolean;
  triggerContext: { schedule: any; accountId: string } | null;
  triggerResult: { logs: string[]; signal: any; meta: any } | null;
  onClose: () => void;
  onRerun: () => void;
  onConfirmOrder: () => void;
};

export default function TriggerModal({
  open,
  triggering,
  triggerContext,
  triggerResult,
  onClose,
  onRerun,
  onConfirmOrder,
}: Props) {
  const { t } = useTranslation();
  const safeStringify = (obj: any) => {
    try {
      return JSON.stringify(
        obj,
        (_k, v) => (typeof v === 'bigint' ? v.toString() : v),
        2,
      );
    } catch (_e) {
      return String(obj);
    }
  };

  const canOrder = (() => {
    const sig: any = triggerResult?.signal;
    if (!sig) return false;
    const raw = String(sig?.type ?? sig?.signalType ?? sig?.signal ?? '').trim().toLowerCase();
    const actionOk = raw === 'buy' || raw === 'sell';
    const volNum = typeof sig?.volume === 'number' ? sig.volume : Number(sig?.volume);
    const volOk = Number.isFinite(volNum) && volNum > 0;
    return actionOk && volOk;
  })();

  return (
    <Modal
      title={t(TRIGGER_MODAL_TRADING_TITLE_KEY)}
      open={open}
      onCancel={() => {
        if (triggering) return;
        onClose();
      }}
      footer={
        <Space>
          <Button
            onClick={() => {
              if (triggering) return;
              onClose();
            }}
          >
            {t('common.close')}
          </Button>
          <Popconfirm
            title={t(TRIGGER_MODAL_CONFIRM_ORDER_TRADING_TITLE_KEY)}
            okText={t(TRIGGER_MODAL_CONFIRM_ORDER_OK_KEY)}
            cancelText={t('common.cancel')}
            onConfirm={onConfirmOrder}
            disabled={!canOrder || triggering}
          >
            <Button type="primary" disabled={!canOrder} loading={triggering}>
              {t(TRIGGER_MODAL_ACTIONS_CONFIRM_ORDER_KEY)}
            </Button>
          </Popconfirm>
        </Space>
      }
      width={860}
    >
      <Space orientation="vertical" style={{ width: '100%' }}>
        <Descriptions size="small" bordered column={2}>
          <Descriptions.Item label={t(TRIGGER_MODAL_SUMMARY_SCHEDULE_NAME_KEY)}>{triggerContext?.schedule?.name || '-'}</Descriptions.Item>
          <Descriptions.Item label={t(TRIGGER_MODAL_SUMMARY_TRADING_ACCOUNT_KEY)}>{triggerContext?.accountId || '-'}</Descriptions.Item>
          <Descriptions.Item label={t(TRIGGER_MODAL_SUMMARY_TRADING_SYMBOL_KEY)}>{triggerContext?.schedule?.symbol || '-'}</Descriptions.Item>
          <Descriptions.Item label={t(TRIGGER_MODAL_SUMMARY_TIMEFRAME_KEY)}>{triggerContext?.schedule?.timeframe || '-'}</Descriptions.Item>
        </Descriptions>

        <Button onClick={onRerun} loading={triggering} disabled={!triggerContext?.schedule}>
          {t(TRIGGER_MODAL_ACTIONS_RERUN_KEY)}
        </Button>

        {triggerResult?.meta?.error ? <Text type="danger">{triggerResult.meta.error}</Text> : null}
        {!canOrder && triggerResult?.signal ? (
          <Text type="secondary">{t(TRIGGER_MODAL_MESSAGES_SIGNAL_NOT_ORDERABLE_KEY)}</Text>
        ) : null}

        <Card size="small" title={t(TRIGGER_MODAL_CARDS_LOGS_KEY)} styles={{ body: { maxHeight: 200, overflow: 'auto' } }}>
          {triggerResult?.logs?.length ? (
            <pre style={{ margin: 0, whiteSpace: 'pre-wrap' }}>{triggerResult.logs.join('\n')}</pre>
          ) : (
            <Text type="secondary">{t(TRIGGER_MODAL_EMPTY_LOGS_KEY)}</Text>
          )}
        </Card>

        <Card size="small" title={t(TRIGGER_MODAL_CARDS_SIGNAL_KEY)} styles={{ body: { maxHeight: 240, overflow: 'auto' } }}>
          {triggerResult?.signal ? (
            <pre style={{ margin: 0, whiteSpace: 'pre-wrap' }}>{safeStringify(triggerResult.signal)}</pre>
          ) : (
            <Text type="secondary">{t(TRIGGER_MODAL_EMPTY_SIGNAL_KEY)}</Text>
          )}
        </Card>
      </Space>
    </Modal>
  );
}
