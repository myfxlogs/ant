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
        message.error(res.message || t(SCHEDULE_LAUNCH_UPDATE_PASSWORD_FAILED_KEY, '密码验证失败，请检查是否正确'));
        return;
      }
      if (res.isInvestor) {
        message.warning(t(SCHEDULE_LAUNCH_UPDATE_PASSWORD_STILL_INVESTOR_KEY, '登录成功，但该账户仍是投资者只读模式，无法下单。'));
      } else {
        message.success(t(SCHEDULE_LAUNCH_UPDATE_PASSWORD_OK_KEY, '交易密码已更新，账户具备交易权限'));
      }
      onSuccess(res);
    } catch (e: unknown) {
      message.error(String((e as any)?.message || t('common.unknownError', { defaultValue: 'Unknown error' })));
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Modal
      title={t(SCHEDULE_LAUNCH_UPDATE_PASSWORD_TITLE_KEY, '填写交易密码')}
      open={open}
      onCancel={onCancel}
      onOk={() => void handleSubmit()}
      confirmLoading={submitting}
      okText={t('common.confirm', '确认')}
      cancelText={t('common.cancel', '取消')}
      destroyOnClose
    >
      <div className="text-sm text-gray-600 mb-3">
        {t(SCHEDULE_LAUNCH_UPDATE_PASSWORD_HINT_KEY, '后端会用新密码做一次 Connect 测试，成功后覆盖当前存储的密码。MT5 账户会同时识别出是否为投资者模式。')}
      </div>
      <Input.Password
        autoFocus
        placeholder={t(SCHEDULE_LAUNCH_NEW_PASSWORD_PLACEHOLDER_KEY, '新的交易密码')}
        value={password}
        onChange={(e) => setPassword(e.target.value)}
        onPressEnter={() => void handleSubmit()}
      />
    </Modal>
  );
};

export default TradePasswordModal;
