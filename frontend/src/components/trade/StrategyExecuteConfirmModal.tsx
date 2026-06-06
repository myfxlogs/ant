import React from 'react';
import { Typography, Space, Alert } from 'antd';
import { useTranslation } from 'react-i18next';
import { TradeConfirmModal } from './TradeConfirmModal';

const { Text } = Typography;

interface StrategyExecuteConfirmProps {
  open: boolean;
  strategyName: string;
  symbol: string;
  action: 'buy' | 'sell';
  volume: number;
  loading?: boolean;
  onConfirm: () => void;
  onCancel: () => void;
}

export const StrategyExecuteConfirmModal: React.FC<StrategyExecuteConfirmProps> = ({
  open, strategyName, symbol, action, volume, loading, onConfirm, onCancel,
}) => {
  const { t } = useTranslation();

  return (
    <TradeConfirmModal
      open={open}
      title={t('trading.strategyExecute.confirm.title')}
      danger
      confirmText={t('trading.strategyExecute.confirm.confirmText')}
      loading={loading}
      onConfirm={onConfirm}
      onCancel={onCancel}
      content={
        <Space orientation="vertical" size="middle" style={{ width: '100%' }}>
          <Alert type="warning" showIcon
            message={t('trading.strategyExecute.confirm.warningTitle')}
            description={t('trading.strategyExecute.confirm.warningDescription')} />
          <div style={{ background: '#fff1f0', padding: 12, borderRadius: 4, border: '1px solid #ffa39e' }}>
            <Space orientation="vertical" size="small" style={{ width: '100%' }}>
              <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                <Text type="secondary">{t('trading.strategyExecute.confirm.strategyName')}:</Text>
                <Text strong>{strategyName}</Text>
              </div>
              <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                <Text type="secondary">{t('trading.strategyExecute.confirm.symbol')}:</Text>
                <Text strong>{symbol}</Text>
              </div>
              <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                <Text type="secondary">{t('trading.strategyExecute.confirm.action')}:</Text>
                <Text strong style={{ color: action === 'buy' ? '#52c41a' : '#ff4d4f' }}>
                  {action === 'buy' ? t('trading.strategyExecute.confirm.buy') : t('trading.strategyExecute.confirm.sell')}
                </Text>
              </div>
              <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                <Text type="secondary">{t('trading.strategyExecute.confirm.volume')}:</Text>
                <Text strong>{volume}</Text>
              </div>
            </Space>
          </div>
        </Space>
      }
    />
  );
};
