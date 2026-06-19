import React from 'react';
import { Typography, Space, Alert } from 'antd';
import { useTranslation } from 'react-i18next'
import { STRATEGY_EXECUTE_CONFIRM_ACTION_KEY, STRATEGY_EXECUTE_CONFIRM_BUY_KEY, STRATEGY_EXECUTE_CONFIRM_CONFIRM_TEXT_KEY, STRATEGY_EXECUTE_CONFIRM_SELL_KEY, STRATEGY_EXECUTE_CONFIRM_STRATEGY_NAME_KEY, STRATEGY_EXECUTE_CONFIRM_SYMBOL_KEY, STRATEGY_EXECUTE_CONFIRM_TITLE_KEY, STRATEGY_EXECUTE_CONFIRM_VOLUME_KEY, STRATEGY_EXECUTE_CONFIRM_WARNING_DESCRIPTION_KEY, STRATEGY_EXECUTE_CONFIRM_WARNING_TITLE_KEY } from '@/gen/ant/v1/i18n/trading_keys';

;
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
      title={t(STRATEGY_EXECUTE_CONFIRM_TITLE_KEY)}
      danger
      confirmText={t(STRATEGY_EXECUTE_CONFIRM_CONFIRM_TEXT_KEY)}
      loading={loading}
      onConfirm={onConfirm}
      onCancel={onCancel}
      content={
        <Space orientation="vertical" size="middle" style={{ width: '100%' }}>
          <Alert type="warning" showIcon
            message={t(STRATEGY_EXECUTE_CONFIRM_WARNING_TITLE_KEY)}
            description={t(STRATEGY_EXECUTE_CONFIRM_WARNING_DESCRIPTION_KEY)} />
          <div style={{ background: '#fff1f0', padding: 12, borderRadius: 4, border: '1px solid #ffa39e' }}>
            <Space orientation="vertical" size="small" style={{ width: '100%' }}>
              <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                <Text type="secondary">{t(STRATEGY_EXECUTE_CONFIRM_STRATEGY_NAME_KEY)}:</Text>
                <Text strong>{strategyName}</Text>
              </div>
              <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                <Text type="secondary">{t(STRATEGY_EXECUTE_CONFIRM_SYMBOL_KEY)}:</Text>
                <Text strong>{symbol}</Text>
              </div>
              <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                <Text type="secondary">{t(STRATEGY_EXECUTE_CONFIRM_ACTION_KEY)}:</Text>
                <Text strong style={{ color: action === 'buy' ? '#52c41a' : '#ff4d4f' }}>
                  {action === 'buy' ? t(STRATEGY_EXECUTE_CONFIRM_BUY_KEY) : t(STRATEGY_EXECUTE_CONFIRM_SELL_KEY)}
                </Text>
              </div>
              <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                <Text type="secondary">{t(STRATEGY_EXECUTE_CONFIRM_VOLUME_KEY)}:</Text>
                <Text strong>{volume}</Text>
              </div>
            </Space>
          </div>
        </Space>
      }
    />
  );
};
