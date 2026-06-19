import React from 'react';
import { Typography, Space, Alert } from 'antd';
import { useTranslation } from 'react-i18next'
import { AUTO_TRADE_CONFIRM_DISABLE_CONFIRM_KEY, AUTO_TRADE_CONFIRM_DISABLE_INFO_DESCRIPTION_KEY, AUTO_TRADE_CONFIRM_DISABLE_INFO_TITLE_KEY, AUTO_TRADE_CONFIRM_DISABLE_QUESTION_KEY, AUTO_TRADE_CONFIRM_DISABLE_TITLE_KEY, AUTO_TRADE_CONFIRM_ENABLE_BULLET1_KEY, AUTO_TRADE_CONFIRM_ENABLE_BULLET2_KEY, AUTO_TRADE_CONFIRM_ENABLE_BULLET3_KEY, AUTO_TRADE_CONFIRM_ENABLE_CONFIRM_KEY, AUTO_TRADE_CONFIRM_ENABLE_QUESTION_KEY, AUTO_TRADE_CONFIRM_ENABLE_RISK_DESCRIPTION_KEY, AUTO_TRADE_CONFIRM_ENABLE_RISK_TITLE_KEY, AUTO_TRADE_CONFIRM_ENABLE_TITLE_KEY } from '@/gen/ant/v1/i18n/trading_keys';

;
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
      title={enabling ? t(AUTO_TRADE_CONFIRM_ENABLE_TITLE_KEY) : t(AUTO_TRADE_CONFIRM_DISABLE_TITLE_KEY)}
      danger={enabling}
      confirmText={enabling ? t(AUTO_TRADE_CONFIRM_ENABLE_CONFIRM_KEY) : t(AUTO_TRADE_CONFIRM_DISABLE_CONFIRM_KEY)}
      loading={loading}
      onConfirm={onConfirm}
      onCancel={onCancel}
      content={
        <Space orientation="vertical" size="middle" style={{ width: '100%' }}>
          {enabling ? (
            <>
              <Alert type="warning" showIcon
                message={t(AUTO_TRADE_CONFIRM_ENABLE_RISK_TITLE_KEY)}
                description={t(AUTO_TRADE_CONFIRM_ENABLE_RISK_DESCRIPTION_KEY)} />
              <div>
                <Text>{t(AUTO_TRADE_CONFIRM_ENABLE_QUESTION_KEY)}</Text>
                <ul style={{ marginTop: 8, marginBottom: 0 }}>
                  <li>{t(AUTO_TRADE_CONFIRM_ENABLE_BULLET1_KEY)}</li>
                  <li>{t(AUTO_TRADE_CONFIRM_ENABLE_BULLET2_KEY)}</li>
                  <li>{t(AUTO_TRADE_CONFIRM_ENABLE_BULLET3_KEY)}</li>
                </ul>
              </div>
            </>
          ) : (
            <>
              <Alert type="info" showIcon
                message={t(AUTO_TRADE_CONFIRM_DISABLE_INFO_TITLE_KEY)}
                description={t(AUTO_TRADE_CONFIRM_DISABLE_INFO_DESCRIPTION_KEY)} />
              <Text>{t(AUTO_TRADE_CONFIRM_DISABLE_QUESTION_KEY)}</Text>
            </>
          )}
        </Space>
      }
    />
  );
};
