import React from 'react';
import { Typography, Space, Alert } from 'antd';
import { useTranslation } from 'react-i18next'
import { RISK_CONFIG_CONFIRM_CONFIRM_TEXT_KEY, RISK_CONFIG_CONFIRM_DESCRIPTION_KEY, RISK_CONFIG_CONFIRM_INFO_KEY, RISK_CONFIG_CONFIRM_TITLE_KEY, RISK_CONFIG_FIELDS_MAX_DAILY_LOSS_KEY, RISK_CONFIG_FIELDS_MAX_DRAWDOWN_PERCENT_KEY, RISK_CONFIG_FIELDS_MAX_LOT_SIZE_KEY, RISK_CONFIG_FIELDS_MAX_POSITIONS_KEY, RISK_CONFIG_FIELDS_MAX_RISK_PERCENT_KEY, RISK_CONFIG_FIELDS_TRAILING_STOP_ENABLED_KEY, RISK_CONFIG_FIELDS_TRAILING_STOP_PIPS_KEY } from '@/gen/ant/v1/i18n/trading_keys';

;
import { TradeConfirmModal } from './TradeConfirmModal';

const { Text } = Typography;

interface RiskConfigConfirmProps {
  open: boolean;
  values: Record<string, unknown>;
  loading?: boolean;
  onConfirm: () => void;
  onCancel: () => void;
}

export const RiskConfigConfirmModal: React.FC<RiskConfigConfirmProps> = ({
  open, values, loading, onConfirm, onCancel,
}) => {
  const { t } = useTranslation();

  const formatValue = (key: string, value: unknown): string => {
    if (typeof value === 'number') {
      if (key.includes('percent') || key.includes('Percent')) return `${value}%`;
      if (key.includes('loss') || key.includes('Loss')) return `$${value}`;
      return String(value);
    }
    return String(value);
  };

  const fieldLabels: Record<string, string> = {
    max_risk_percent: t(RISK_CONFIG_FIELDS_MAX_RISK_PERCENT_KEY),
    max_daily_loss: t(RISK_CONFIG_FIELDS_MAX_DAILY_LOSS_KEY),
    max_drawdown_percent: t(RISK_CONFIG_FIELDS_MAX_DRAWDOWN_PERCENT_KEY),
    max_positions: t(RISK_CONFIG_FIELDS_MAX_POSITIONS_KEY),
    max_lot_size: t(RISK_CONFIG_FIELDS_MAX_LOT_SIZE_KEY),
    trailing_stop_enabled: t(RISK_CONFIG_FIELDS_TRAILING_STOP_ENABLED_KEY),
    trailing_stop_pips: t(RISK_CONFIG_FIELDS_TRAILING_STOP_PIPS_KEY),
  };

  return (
    <TradeConfirmModal
      open={open}
      title={t(RISK_CONFIG_CONFIRM_TITLE_KEY)}
      confirmText={t(RISK_CONFIG_CONFIRM_CONFIRM_TEXT_KEY)}
      loading={loading}
      onConfirm={onConfirm}
      onCancel={onCancel}
      content={
        <Space orientation="vertical" size="middle" style={{ width: '100%' }}>
          <Text>{t(RISK_CONFIG_CONFIRM_DESCRIPTION_KEY)}</Text>
          <div style={{ background: '#f5f5f5', padding: 12, borderRadius: 4 }}>
            {Object.entries(values).map(([key, value]) => (
              <div key={key} style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 4 }}>
                <Text type="secondary">{fieldLabels[key] || key}:</Text>
                <Text strong>{formatValue(key, value)}</Text>
              </div>
            ))}
          </div>
          <Alert type="info" showIcon message={t(RISK_CONFIG_CONFIRM_INFO_KEY)} />
        </Space>
      }
    />
  );
};
