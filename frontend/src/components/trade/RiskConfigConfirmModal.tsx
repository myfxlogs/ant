import React from 'react';
import { Typography, Space, Alert } from 'antd';
import { useTranslation } from 'react-i18next';
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
    max_risk_percent: t('trading.riskConfig.fields.maxRiskPercent'),
    max_daily_loss: t('trading.riskConfig.fields.maxDailyLoss'),
    max_drawdown_percent: t('trading.riskConfig.fields.maxDrawdownPercent'),
    max_positions: t('trading.riskConfig.fields.maxPositions'),
    max_lot_size: t('trading.riskConfig.fields.maxLotSize'),
    trailing_stop_enabled: t('trading.riskConfig.fields.trailingStopEnabled'),
    trailing_stop_pips: t('trading.riskConfig.fields.trailingStopPips'),
  };

  return (
    <TradeConfirmModal
      open={open}
      title={t('trading.riskConfig.confirm.title')}
      confirmText={t('trading.riskConfig.confirm.confirmText')}
      loading={loading}
      onConfirm={onConfirm}
      onCancel={onCancel}
      content={
        <Space orientation="vertical" size="middle" style={{ width: '100%' }}>
          <Text>{t('trading.riskConfig.confirm.description')}</Text>
          <div style={{ background: '#f5f5f5', padding: 12, borderRadius: 4 }}>
            {Object.entries(values).map(([key, value]) => (
              <div key={key} style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 4 }}>
                <Text type="secondary">{fieldLabels[key] || key}:</Text>
                <Text strong>{formatValue(key, value)}</Text>
              </div>
            ))}
          </div>
          <Alert type="info" showIcon message={t('trading.riskConfig.confirm.info')} />
        </Space>
      }
    />
  );
};
