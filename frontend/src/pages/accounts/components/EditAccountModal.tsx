import { Button, Modal } from 'antd';
import { useState } from 'react';
import { showError, showSuccess, showWarning } from '@/utils/message';
import type { Account } from '@/types/account';
import GradientButton from '@/components/common/GradientButton';
import { useTranslation } from 'react-i18next';
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
      showWarning(t('accounts.edit.messages.enterOldPassword'));
      return;
    }
    if (!newPassword) {
      showWarning(t('accounts.edit.messages.enterPassword'));
      return;
    }
    if (!account) return;
    setSaving(true);
    try {
      const result = await accountApi.updateTradingPassword(account.id, newPassword, oldPassword);
      if (result.success) {
        showSuccess(t('accounts.edit.messages.passwordSaved'));
        setOldPassword('');
        setNewPassword('');
        onClose();
      } else {
        showError(result.message || t('accounts.edit.messages.passwordVerifyFailed'));
      }
    } catch (e: unknown) {
      const msg = (e as { message?: string })?.message || '';
      showError(msg || t('accounts.edit.messages.passwordVerifyFailed'));
    } finally {
      setSaving(false);
    }
  };

  return (
    <Modal title={t('accounts.edit.title')} open={open} onCancel={onClose} footer={null} width={480}>
      {account && (
        <div className="space-y-4">
          <div className="p-4 rounded-xl" style={{ background: 'var(--color-bg-secondary)' }}>
            <div className="flex justify-between mb-2">
              <span style={{ color: 'var(--color-text-muted)' }}>{t('accounts.edit.fields.tradingAccount')}</span>
              <span style={{ color: 'var(--color-text)' }}>{account.login}</span>
            </div>
            <div className="flex justify-between">
              <span style={{ color: 'var(--color-text-muted)' }}>{t('accounts.edit.fields.server')}</span>
              <span style={{ color: 'var(--color-text)' }}>
                {account.brokerServer || account.brokerCompany}
              </span>
            </div>
          </div>

          <div>
            <label className="block mb-2" style={{ color: 'var(--color-text-muted)' }}>
              {t('accounts.edit.fields.oldPassword')}
            </label>
            <input
              type="password"
              autoComplete="current-password"
              value={oldPassword}
              onChange={(e) => setOldPassword(e.target.value)}
              placeholder={t('accounts.edit.placeholders.oldPassword')}
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
              {t('accounts.edit.fields.password')}
            </label>
            <input
              type="password"
              autoComplete="new-password"
              value={newPassword}
              onChange={(e) => setNewPassword(e.target.value)}
              placeholder={t('accounts.edit.placeholders.newPassword')}
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
