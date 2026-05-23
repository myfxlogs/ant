import { Card, Tag, Button, Space, Progress, Popconfirm } from 'antd';
import {
  IconTrendingUp,
  IconTrendingDown,
  IconCheck,
  IconX,
  IconPlayerPlay,
  IconTarget,
  IconShield,
} from '@tabler/icons-react';
import type { Signal } from '@/types/ai';
import { getDeviceLocale, getDeviceTimeZone } from '@/utils/date';
import { useTranslation } from 'react-i18next';

interface SignalCardProps {
  signal: Signal;
  onConfirm: (id: string) => void;
  onCancel: (id: string) => void;
  onExecute: (id: string) => void;
}

export default function SignalCard({ signal, onConfirm, onCancel, onExecute }: SignalCardProps) {
  const { t } = useTranslation();
  const getStatusTag = () => {
    switch (signal.status) {
      case 'pending':
        return <Tag color="orange">{t('ai.signalCard.status.pending')}</Tag>;
      case 'confirmed':
        return <Tag color="blue">{t('ai.signalCard.status.confirmed')}</Tag>;
      case 'executed':
        return <Tag color="green">{t('ai.signalCard.status.executed')}</Tag>;
      case 'cancelled':
        return <Tag color="default">{t('ai.signalCard.status.cancelled')}</Tag>;
      default:
        return <Tag>{t('common.unknown')}</Tag>;
    }
  };

  const getConfidenceColor = (confidence: number) => {
    if (confidence >= 80) return '#00A651';
    if (confidence >= 60) return '#D4AF37';
    return '#E53935';
  };

  const formatDate = (dateStr: string) => {
    const date = new Date(dateStr);
    const locale = getDeviceLocale();
    const timeZone = getDeviceTimeZone();
    return date.toLocaleString(locale, {
      timeZone,
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
    });
  };

  const isPending = signal.status === 'pending';
  const isConfirmed = signal.status === 'confirmed';

  return (
    <Card
      className="signal-card"
      style={{
        background: '#FFFFFF',
        border: '1px solid rgba(0, 0, 0, 0.08)',
        borderRadius: '12px',
      }}
      styles={{
        body: { padding: '16px' },
      }}
    >
      <div className="space-y-3">
        {/* 头部 */}
        <div className="flex items-start justify-between">
          <div className="flex items-center gap-2">
            {signal.type === 'buy' ? (
              <div
                className="w-10 h-10 rounded-xl flex items-center justify-center"
                style={{ background: 'rgba(0, 166, 81, 0.1)' }}
              >
                <IconTrendingUp size={20} stroke={1.5} color="#00A651" />
              </div>
            ) : (
              <div
                className="w-10 h-10 rounded-xl flex items-center justify-center"
                style={{ background: 'rgba(229, 57, 53, 0.1)' }}
              >
                <IconTrendingDown size={20} stroke={1.5} color="#E53935" />
              </div>
            )}
            <div>
              <div className="flex items-center gap-2">
                <Tag color={signal.type === 'buy' ? 'green' : 'red'}>
                  {signal.type === 'buy'
                    ? t('trading.strategyExecute.confirm.buy')
                    : t('trading.strategyExecute.confirm.sell')}
                </Tag>
                <Tag color="gold">{signal.symbol}</Tag>
                {getStatusTag()}
              </div>
            </div>
          </div>
        </div>

        {/* 价格信息 */}
        <div className="grid grid-cols-3 gap-2">
          <div
            className="p-2 rounded-lg"
            style={{ background: '#F5F7F9' }}
          >
            <div className="text-xs mb-1" style={{ color: '#8A9AA5' }}>
              {t('ai.signalCard.labels.price')}
            </div>
            <div className="font-semibold" style={{ color: '#141D22' }}>
              {signal.price.toFixed(2)}
            </div>
          </div>
          <div
            className="p-2 rounded-lg"
            style={{ background: '#F5F7F9' }}
          >
            <div className="text-xs mb-1" style={{ color: '#8A9AA5' }}>
              {t('ai.signalCard.labels.volume')}
            </div>
            <div className="font-semibold" style={{ color: '#141D22' }}>
              {signal.volume}
            </div>
          </div>
          <div
            className="p-2 rounded-lg"
            style={{ background: '#F5F7F9' }}
          >
            <div className="text-xs mb-1" style={{ color: '#8A9AA5' }}>
              {t('ai.signalCard.labels.confidence')}
            </div>
            <div className="flex items-center gap-1">
              <Progress
                percent={signal.confidence}
                size="small"
                showInfo={false}
                strokeColor={getConfidenceColor(signal.confidence)}
              />
              <span
                className="font-semibold text-sm"
                style={{ color: getConfidenceColor(signal.confidence) }}
              >
                {signal.confidence}%
              </span>
            </div>
          </div>
        </div>

        {/* 止损止盈 */}
        {(signal.stop_loss || signal.take_profit) && (
          <div className="flex gap-4">
            {signal.stop_loss && (
              <div className="flex items-center gap-2 text-sm">
                <IconShield size={14} stroke={1.5} color="#E53935" />
                <span style={{ color: '#8A9AA5' }}>{t('ai.signalCard.labels.stopLoss')}:</span>
                <span style={{ color: '#E53935' }}>{signal.stop_loss.toFixed(2)}</span>
              </div>
            )}
            {signal.take_profit && (
              <div className="flex items-center gap-2 text-sm">
                <IconTarget size={14} stroke={1.5} color="#00A651" />
                <span style={{ color: '#8A9AA5' }}>{t('ai.signalCard.labels.takeProfit')}:</span>
                <span style={{ color: '#00A651' }}>{signal.take_profit.toFixed(2)}</span>
              </div>
            )}
          </div>
        )}

        {/* 原因 */}
        <div
          className="text-sm p-2 rounded-lg"
          style={{ background: '#F5F7F9', color: '#5A6B75' }}
        >
          <span className="font-medium" style={{ color: '#141D22' }}>
            {t('ai.signalCard.labels.analysisReason')}:
          </span>{' '}
          {signal.reason}
        </div>

        {/* 底部 */}
        <div className="flex items-center justify-between pt-2" style={{ borderTop: '1px solid rgba(0, 0, 0, 0.08)' }}>
          <div className="text-xs" style={{ color: '#8A9AA5' }}>
            {formatDate(signal.created_at)}
          </div>

          {/* 操作按钮 */}
          <Space size="small">
            {isPending && (
              <>
                <Button
                  size="small"
                  icon={<IconCheck size={14} stroke={1.5} />}
                  onClick={() => onConfirm(signal.id)}
                >
                  {t('ai.signalCard.actions.confirm')}
                </Button>
                <Popconfirm
                  title={t('ai.signalCard.confirmCancel.title')}
                  onConfirm={() => onCancel(signal.id)}
                  okText={t('common.confirm')}
                  cancelText={t('common.cancel')}
                >
                  <Button
                    size="small"
                    danger
                    icon={<IconX size={14} stroke={1.5} />}
                  >
                    {t('ai.signalCard.actions.cancel')}
                  </Button>
                </Popconfirm>
              </>
            )}
            {isConfirmed && (
              <Popconfirm
                title={t('ai.signalCard.confirmExecute.title')}
                description={t('ai.signalCard.confirmExecute.description')}
                onConfirm={() => onExecute(signal.id)}
                okText={t('common.confirm')}
                cancelText={t('common.cancel')}
              >
                <Button
                  size="small"
                  type="primary"
                  icon={<IconPlayerPlay size={14} stroke={1.5} />}
                >
                  {t('ai.signalCard.actions.executeTrade')}
                </Button>
              </Popconfirm>
            )}
          </Space>
        </div>
      </div>
    </Card>
  );
}
