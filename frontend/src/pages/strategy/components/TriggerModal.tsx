import { Button, Card, Descriptions, Modal, Popconfirm, Space, Typography } from 'antd';
import { useTranslation } from 'react-i18next';

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
      title={t('strategy.schedules.triggerModal.title')}
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
            title={t('strategy.schedules.triggerModal.confirmOrder.title')}
            okText={t('strategy.schedules.triggerModal.confirmOrder.ok')}
            cancelText={t('common.cancel')}
            onConfirm={onConfirmOrder}
            disabled={!canOrder || triggering}
          >
            <Button type="primary" disabled={!canOrder} loading={triggering}>
              {t('strategy.schedules.triggerModal.actions.confirmOrder')}
            </Button>
          </Popconfirm>
        </Space>
      }
      width={860}
    >
      <Space orientation="vertical" style={{ width: '100%' }}>
        <Descriptions size="small" bordered column={2}>
          <Descriptions.Item label={t('strategy.schedules.triggerModal.summary.scheduleName')}>{triggerContext?.schedule?.name || '-'}</Descriptions.Item>
          <Descriptions.Item label={t('strategy.schedules.triggerModal.summary.account')}>{triggerContext?.accountId || '-'}</Descriptions.Item>
          <Descriptions.Item label={t('strategy.schedules.triggerModal.summary.symbol')}>{triggerContext?.schedule?.symbol || '-'}</Descriptions.Item>
          <Descriptions.Item label={t('strategy.schedules.triggerModal.summary.timeframe')}>{triggerContext?.schedule?.timeframe || '-'}</Descriptions.Item>
        </Descriptions>

        <Button onClick={onRerun} loading={triggering} disabled={!triggerContext?.schedule}>
          {t('strategy.schedules.triggerModal.actions.rerun')}
        </Button>

        {triggerResult?.meta?.error ? <Text type="danger">{triggerResult.meta.error}</Text> : null}
        {!canOrder && triggerResult?.signal ? (
          <Text type="secondary">{t('strategy.schedules.triggerModal.messages.signalNotOrderable')}</Text>
        ) : null}

        <Card size="small" title={t('strategy.schedules.triggerModal.cards.logs')} styles={{ body: { maxHeight: 200, overflow: 'auto' } }}>
          {triggerResult?.logs?.length ? (
            <pre style={{ margin: 0, whiteSpace: 'pre-wrap' }}>{triggerResult.logs.join('\n')}</pre>
          ) : (
            <Text type="secondary">{t('strategy.schedules.triggerModal.emptyLogs')}</Text>
          )}
        </Card>

        <Card size="small" title={t('strategy.schedules.triggerModal.cards.signal')} styles={{ body: { maxHeight: 240, overflow: 'auto' } }}>
          {triggerResult?.signal ? (
            <pre style={{ margin: 0, whiteSpace: 'pre-wrap' }}>{safeStringify(triggerResult.signal)}</pre>
          ) : (
            <Text type="secondary">{t('strategy.schedules.triggerModal.emptySignal')}</Text>
          )}
        </Card>
      </Space>
    </Modal>
  );
}
