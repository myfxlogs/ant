import React, { useEffect, useState } from 'react';
import { Input, Modal, message } from 'antd';
import { useTranslation } from 'react-i18next'
import { SCHEDULE_LAUNCH_NEW_PASSWORD_PLACEHOLDER_KEY, SCHEDULE_LAUNCH_UPDATE_PASSWORD_FAILED_KEY, SCHEDULE_LAUNCH_UPDATE_PASSWORD_HINT_KEY, SCHEDULE_LAUNCH_UPDATE_PASSWORD_OK_KEY, SCHEDULE_LAUNCH_UPDATE_PASSWORD_STILL_INVESTOR_KEY, SCHEDULE_LAUNCH_UPDATE_PASSWORD_TITLE_KEY } from '@/gen/ant/v1/i18n/strategy_templates_keys';

;
import { accountApi } from '@/client/account';

interface TradePasswordModalProps {
  open: boolean;
  accountId: string;
  onCancel: () => void;
  onSuccess: (res: { hasTradePermission: boolean; isInvestor: boolean; message: string }) => void;
}

const TradePasswordModal: React.FC<TradePasswordModalProps> = ({ open, accountId, onCancel, onSuccess }) => {
  const { t } = useTranslation();
  const [password, setPassword] = useState('');
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    if (!open) { setPassword(''); setSubmitting(false); }
  }, [open]);

  const handleSubmit = async () => {
    if (!password || !accountId) return;
    setSubmitting(true);
    try {
      const res = await accountApi.updateTradingPassword(accountId, password);
      if (!res.success) {
        message.error(res.message || t(SCHEDULE_LAUNCH_UPDATE_PASSWORD_FAILED_KEY));
        return;
      }
      if (res.isInvestor) {
        message.warning(t(SCHEDULE_LAUNCH_UPDATE_PASSWORD_STILL_INVESTOR_KEY));
      } else {
        message.success(t(SCHEDULE_LAUNCH_UPDATE_PASSWORD_OK_KEY));
      }
      onSuccess(res);
    } catch (e: unknown) {
      message.error(e instanceof Error ? e.message : t('common.unknownError', { defaultValue: 'Unknown error' }));
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Modal
      title={t(SCHEDULE_LAUNCH_UPDATE_PASSWORD_TITLE_KEY)}
      open={open}
      onCancel={onCancel}
      onOk={() => void handleSubmit()}
      confirmLoading={submitting}
      okText={t('common.confirm')}
      cancelText={t('common.cancel')}
      destroyOnClose
    >
      <div className="text-sm text-gray-600 mb-3">
        {t(SCHEDULE_LAUNCH_UPDATE_PASSWORD_HINT_KEY)}
      </div>
      <Input.Password
        autoFocus
        placeholder={t(SCHEDULE_LAUNCH_NEW_PASSWORD_PLACEHOLDER_KEY)}
        value={password}
        onChange={(e) => setPassword(e.target.value)}
        onPressEnter={() => void handleSubmit()}
      />
    </Modal>
  );
};

export default TradePasswordModal;
