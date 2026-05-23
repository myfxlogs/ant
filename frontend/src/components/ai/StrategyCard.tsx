import { Card, Tag, Button, Popconfirm, Space, Tooltip } from 'antd';
import {
  IconPlayerPlay,
  IconPlayerPause,
  IconTrash,
  IconTarget,
  IconBolt,
  IconClock,
} from '@tabler/icons-react';
import type { Strategy } from '@/types/ai';
import { getDeviceLocale, getDeviceTimeZone } from '@/utils/date';
import { useTranslation } from 'react-i18next';

interface StrategyCardProps {
  strategy: Strategy;
  onEnable: (id: string) => void;
  onDisable: (id: string) => void;
  onDelete: (id: string) => void;
}

export default function StrategyCard({ strategy, onEnable, onDisable, onDelete }: StrategyCardProps) {
  const { t } = useTranslation();
  const getStatusTag = () => {
    switch (strategy.status) {
      case 'active':
        return <Tag color="green">{t('ai.strategyCard.status.active')}</Tag>;
      case 'inactive':
        return <Tag color="default">{t('ai.strategyCard.status.inactive')}</Tag>;
      case 'paused':
        return <Tag color="orange">{t('ai.strategyCard.status.paused')}</Tag>;
      default:
        return <Tag color="default">{t('common.unknown')}</Tag>;
    }
  };

  const getActionTypeTag = (type: string) => {
    switch (type) {
      case 'buy':
        return <Tag color="green">{t('ai.strategyCard.actionType.buy')}</Tag>;
      case 'sell':
        return <Tag color="red">{t('ai.strategyCard.actionType.sell')}</Tag>;
      case 'close_long':
        return <Tag color="blue">{t('ai.strategyCard.actionType.closeLong')}</Tag>;
      case 'close_short':
        return <Tag color="orange">{t('ai.strategyCard.actionType.closeShort')}</Tag>;
      case 'alert':
        return <Tag color="purple">{t('ai.strategyCard.actionType.alert')}</Tag>;
      default:
        return <Tag>{type}</Tag>;
    }
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

  return (
    <Card
      className="strategy-card"
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
          <div className="flex-1">
            <div className="flex items-center gap-2 mb-1">
              <h3 className="font-semibold text-base" style={{ color: '#141D22' }}>
                {strategy.name}
              </h3>
              {getStatusTag()}
            </div>
            <div className="flex items-center gap-2 text-sm" style={{ color: '#8A9AA5' }}>
              <Tag color="gold">{strategy.symbol}</Tag>
              {strategy.triggered_count !== undefined && (
                <span className="flex items-center gap-1">
                  <IconTarget size={14} stroke={1.5} />
                  {t('ai.strategyCard.labels.triggeredCount', { count: strategy.triggered_count })}
                </span>
              )}
            </div>
          </div>
        </div>

        {/* 描述 */}
        {strategy.description && (
          <p className="text-sm" style={{ color: '#5A6B75' }}>
            {strategy.description}
          </p>
        )}

        {/* 条件 */}
        <div>
          <div className="text-xs font-medium mb-2" style={{ color: '#8A9AA5' }}>
            {t('ai.strategyCard.sections.conditions')}
          </div>
          <div className="space-y-1">
            {strategy.conditions.map((condition, index) => (
              <div
                key={index}
                className="flex items-center gap-2 text-sm p-2 rounded-lg"
                style={{ background: '#F5F7F9' }}
              >
                <IconBolt size={14} stroke={1.5} color="#D4AF37" />
                <span style={{ color: '#141D22' }}>{condition.description}</span>
              </div>
            ))}
          </div>
        </div>

        {/* 动作 */}
        <div>
          <div className="text-xs font-medium mb-2" style={{ color: '#8A9AA5' }}>
            {t('ai.strategyCard.sections.actions')}
          </div>
          <div className="flex flex-wrap gap-2">
            {strategy.actions.map((action, index) => (
              <div
                key={index}
                className="flex items-center gap-2 text-sm p-2 rounded-lg"
                style={{ background: '#F5F7F9' }}
              >
                {getActionTypeTag(action.type)}
                <span style={{ color: '#141D22' }}>{action.description}</span>
              </div>
            ))}
          </div>
        </div>

        {/* 底部信息 */}
        <div className="flex items-center justify-between pt-2" style={{ borderTop: '1px solid rgba(0, 0, 0, 0.08)' }}>
          <div className="flex items-center gap-4 text-xs" style={{ color: '#8A9AA5' }}>
            <Tooltip title={t('ai.strategyCard.tooltips.createdAt')}>
              <span className="flex items-center gap-1">
                <IconClock size={12} stroke={1.5} />
                {formatDate(strategy.created_at)}
              </span>
            </Tooltip>
            {strategy.last_triggered_at && (
              <Tooltip title={t('ai.strategyCard.tooltips.lastTriggeredAt')}>
                <span>{t('ai.strategyCard.labels.lastTriggeredAt', { time: formatDate(strategy.last_triggered_at) })}</span>
              </Tooltip>
            )}
          </div>

          {/* 操作按钮 */}
          <Space size="small">
            {strategy.status === 'active' ? (
              <Button
                size="small"
                icon={<IconPlayerPause size={14} stroke={1.5} />}
                onClick={() => onDisable(strategy.id)}
              >
                {t('ai.strategyCard.actions.stop')}
              </Button>
            ) : (
              <Button
                size="small"
                type="primary"
                icon={<IconPlayerPlay size={14} stroke={1.5} />}
                onClick={() => onEnable(strategy.id)}
              >
                {t('ai.strategyCard.actions.start')}
              </Button>
            )}
            <Popconfirm
              title={t('ai.strategyCard.confirmDelete.title')}
              description={t('ai.strategyCard.confirmDelete.description')}
              onConfirm={() => onDelete(strategy.id)}
              okText={t('common.confirm')}
              cancelText={t('common.cancel')}
            >
              <Button
                size="small"
                danger
                icon={<IconTrash size={14} stroke={1.5} />}
              />
            </Popconfirm>
          </Space>
        </div>
      </div>
    </Card>
  );
}
