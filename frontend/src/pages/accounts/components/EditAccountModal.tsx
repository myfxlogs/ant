import { Button, Input, Modal } from 'antd';
import { useState } from 'react';
import { showError, showSuccess, showWarning } from '@/utils/message';
import type { Account } from '@/types/account';
import GradientButton from '@/components/common/GradientButton';
import { useTranslation } from 'react-i18next'
import { EDIT_FIELDS_OLD_PASSWORD_KEY, EDIT_FIELDS_PASSWORD_KEY, EDIT_FIELDS_SERVER_KEY, EDIT_FIELDS_TRADING_ACCOUNT_KEY, EDIT_MESSAGES_ENTER_OLD_PASSWORD_KEY, EDIT_MESSAGES_ENTER_PASSWORD_KEY, EDIT_MESSAGES_PASSWORD_SAVED_KEY, EDIT_MESSAGES_PASSWORD_VERIFY_FAILED_KEY, EDIT_PLACEHOLDERS_NEW_PASSWORD_KEY, EDIT_PLACEHOLDERS_OLD_PASSWORD_KEY, EDIT_TITLE_KEY } from '@/gen/ant/v1/i18n/accounts_keys';

;
import { accountApi } from '@/client/account';

type Props = {
  open: boolean;
  account: Account | null;
  onClose: () => void;
};

export default function EditAccountModal({ open, account, onClose }: Props) {
  const { t } = useTranslation();
  const [oldPassword, setOldPassword] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [saving, setSaving] = useState(false);

  const handleSavePassword = async () => {
    if (!oldPassword) {
      showWarning(t(EDIT_MESSAGES_ENTER_OLD_PASSWORD_KEY));
      return;
    }
    if (!newPassword) {
      showWarning(t(EDIT_MESSAGES_ENTER_PASSWORD_KEY));
      return;
    }
    if (!account) return;
    setSaving(true);
    try {
      const result = await accountApi.updateTradingPassword(account.id, newPassword, oldPassword);
      if (result.success) {
        showSuccess(t(EDIT_MESSAGES_PASSWORD_SAVED_KEY));
        setOldPassword('');
        setNewPassword('');
        onClose();
      } else {
        showError(result.message || t(EDIT_MESSAGES_PASSWORD_VERIFY_FAILED_KEY));
      }
    } catch (e: unknown) {
      const msg = (e as { message?: string })?.message || '';
      showError(msg || t(EDIT_MESSAGES_PASSWORD_VERIFY_FAILED_KEY));
    } finally {
      setSaving(false);
    }
  };

  return (
    <Modal title={t(EDIT_TRADING_TITLE_KEY)} open={open} onCancel={onClose} footer={null} width={480}>
      {account && (
        <div className="space-y-4">
          <div className="p-4 rounded-xl" style={{ background: 'var(--color-bg-secondary)' }}>
            <div className="flex justify-between mb-2">
              <span style={{ color: 'var(--color-text-muted)' }}>{t(EDIT_FIELDS_TRADING_TRADING_ACCOUNT_KEY)}</span>
              <span style={{ color: 'var(--color-text)' }}>{account.login}</span>
            </div>
            <div className="flex justify-between">
              <span style={{ color: 'var(--color-text-muted)' }}>{t(EDIT_FIELDS_SERVER_KEY)}</span>
              <span style={{ color: 'var(--color-text)' }}>
                {account.brokerServer || account.brokerCompany}
              </span>
            </div>
          </div>

          <div>
            <label className="block mb-2" style={{ color: 'var(--color-text-muted)' }}>
              {t(EDIT_FIELDS_OLD_PASSWORD_KEY)}
            </label>
            <Input
              autoComplete="current-password"
              value={oldPassword}
              onChange={(e) => setOldPassword(e.target.value)}
              placeholder={t(EDIT_PLACEHOLDERS_OLD_PASSWORD_KEY)}
              className="flex-1 outline-none transition-all w-full"
              style={{
                background: 'var(--color-bg-card)',
                border: '1px solid rgba(185, 201, 223, 0.4)',
                borderRadius: '10px',
                padding: '10px 14px',
                fontSize: '14px',
                color: 'var(--color-text)',
              }}
            />
          </div>

          <div>
            <label className="block mb-2" style={{ color: 'var(--color-text-muted)' }}>
              {t(EDIT_FIELDS_PASSWORD_KEY)}
            </label>
            <Input
              autoComplete="new-password"
              value={newPassword}
              onChange={(e) => setNewPassword(e.target.value)}
              placeholder={t(EDIT_PLACEHOLDERS_NEW_PASSWORD_KEY)}
              className="flex-1 outline-none transition-all w-full"
              style={{
                background: 'var(--color-bg-card)',
                border: '1px solid rgba(185, 201, 223, 0.4)',
                borderRadius: '10px',
                padding: '10px 14px',
                fontSize: '14px',
                color: 'var(--color-text)',
              }}
            />
          </div>

          <div className="flex justify-end gap-2 pt-4">
            <Button onClick={onClose} style={{ borderRadius: '10px' }}>
              {t('common.cancel')}
            </Button>
            <GradientButton
              onClick={handleSavePassword}
              disabled={!oldPassword || !newPassword || saving}
              loading={saving}
              style={{ borderRadius: '10px' }}
            >
              {t('common.save')}
            </GradientButton>
          </div>
        </div>
      )}
    </Modal>
  );
}
