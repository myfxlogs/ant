import React from 'react';
import { Modal, Typography, Space, Alert } from 'antd';
import { ExclamationCircleOutlined, WarningOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';

const { Text } = Typography;

interface TradeConfirmModalProps {
  open: boolean;
  title: string;
  content: React.ReactNode;
  confirmText?: string;
  cancelText?: string;
  danger?: boolean;
  loading?: boolean;
  onConfirm: () => void;
  onCancel: () => void;
}

export const TradeConfirmModal: React.FC<TradeConfirmModalProps> = ({
  open,
  title,
  content,
  confirmText,
  cancelText,
  danger = false,
  loading = false,
  onConfirm,
  onCancel,
}) => {
  const { t } = useTranslation();
  const resolvedConfirmText = confirmText ?? t('common.confirm');
  const resolvedCancelText = cancelText ?? t('common.cancel');

  return (
    <Modal
      open={open}
      title={
        <Space>
          {danger ? (
            <WarningOutlined style={{ color: '#ff4d4f' }} />
          ) : (
            <ExclamationCircleOutlined style={{ color: '#faad14' }} />
          )}
          <span>{title}</span>
        </Space>
      }
      onOk={onConfirm}
      onCancel={onCancel}
      okText={resolvedConfirmText}
      cancelText={resolvedCancelText}
      okButtonProps={{ danger, loading }}
      focusTriggerAfterClose
    >
      {content}
    </Modal>
  );
};

interface AutoTradeConfirmProps {
  open: boolean;
  enabling: boolean;
  loading?: boolean;
  onConfirm: () => void;
  onCancel: () => void;
}

export const AutoTradeConfirmModal: React.FC<AutoTradeConfirmProps> = ({
  open,
  enabling,
  loading,
  onConfirm,
  onCancel,
}) => {
  const { t } = useTranslation();

  return (
    <TradeConfirmModal
      open={open}
      title={enabling ? t('trading.autoTrade.confirm.enableTitle') : t('trading.autoTrade.confirm.disableTitle')}
      danger={enabling}
      confirmText={enabling ? t('trading.autoTrade.confirm.enableConfirm') : t('trading.autoTrade.confirm.disableConfirm')}
      loading={loading}
      onConfirm={onConfirm}
      onCancel={onCancel}
      content={
        <Space orientation="vertical" size="middle" style={{ width: '100%' }}>
          {enabling ? (
            <>
              <Alert
                type="warning"
                showIcon
                message={t('trading.autoTrade.confirm.enableRiskTitle')}
                description={t('trading.autoTrade.confirm.enableRiskDescription')}
              />
              <div>
                <Text>{t('trading.autoTrade.confirm.enableQuestion')}</Text>
                <ul style={{ marginTop: 8, marginBottom: 0 }}>
                  <li>{t('trading.autoTrade.confirm.enableBullet1')}</li>
                  <li>{t('trading.autoTrade.confirm.enableBullet2')}</li>
                  <li>{t('trading.autoTrade.confirm.enableBullet3')}</li>
                </ul>
              </div>
            </>
          ) : (
            <>
              <Alert
                type="info"
                showIcon
                message={t('trading.autoTrade.confirm.disableInfoTitle')}
                description={t('trading.autoTrade.confirm.disableInfoDescription')}
              />
              <Text>{t('trading.autoTrade.confirm.disableQuestion')}</Text>
            </>
          )}
        </Space>
      }
    />
  );
};

interface RiskConfigConfirmProps {
  open: boolean;
  values: Record<string, unknown>;
  loading?: boolean;
  onConfirm: () => void;
  onCancel: () => void;
}

export const RiskConfigConfirmModal: React.FC<RiskConfigConfirmProps> = ({
  open,
  values,
  loading,
  onConfirm,
  onCancel,
}) => {
  const { t } = useTranslation();

  const formatValue = (key: string, value: unknown): string => {
    if (typeof value === 'number') {
      if (key.includes('percent') || key.includes('Percent')) {
        return `${value}%`;
      }
      if (key.includes('loss') || key.includes('Loss')) {
        return `$${value}`;
      }
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
          <Alert
            type="info"
            showIcon
            message={t('trading.riskConfig.confirm.info')}
          />
        </Space>
      }
    />
  );
};

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
  open,
  strategyName,
  symbol,
  action,
  volume,
  loading,
  onConfirm,
  onCancel,
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
          <Alert
            type="warning"
            showIcon
            message={t('trading.strategyExecute.confirm.warningTitle')}
            description={t('trading.strategyExecute.confirm.warningDescription')}
          />
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
