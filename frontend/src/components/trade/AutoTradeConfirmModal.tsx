import React from 'react';
import { Typography, Space, Alert } from 'antd';
import { useTranslation } from 'react-i18next';
import { TradeConfirmModal } from './TradeConfirmModal';

const { Text } = Typography;

interface AutoTradeConfirmProps {
  open: boolean;
  enabling: boolean;
  loading?: boolean;
  onConfirm: () => void;
  onCancel: () => void;
}

export const AutoTradeConfirmModal: React.FC<AutoTradeConfirmProps> = ({
  open, enabling, loading, onConfirm, onCancel,
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
              <Alert type="warning" showIcon
                message={t('trading.autoTrade.confirm.enableRiskTitle')}
                description={t('trading.autoTrade.confirm.enableRiskDescription')} />
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
              <Alert type="info" showIcon
                message={t('trading.autoTrade.confirm.disableInfoTitle')}
                description={t('trading.autoTrade.confirm.disableInfoDescription')} />
              <Text>{t('trading.autoTrade.confirm.disableQuestion')}</Text>
            </>
          )}
        </Space>
      }
    />
  );
};
