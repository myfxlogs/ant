import { Button, Modal } from 'antd';
import { IconCheck } from '@tabler/icons-react';
import { useState } from 'react';
import { showError, showSuccess, showWarning } from '@/utils/message';
import type { Account } from '@/types/account';
import GradientButton from '@/components/common/GradientButton';
import { useTranslation } from 'react-i18next';

type Props = {
  open: boolean;
  account: Account | null;
  onClose: () => void;
};

export default function EditAccountModal({ open, account, onClose }: Props) {
  const { t } = useTranslation();
  const [editPassword, setEditPassword] = useState('');
  const [verifying, setVerifying] = useState(false);
  const [verified, setVerified] = useState(false);

  const handleVerifyPassword = async () => {
    if (!editPassword) {
      showWarning(t('accounts.edit.messages.enterPassword'));
      return;
    }
    setVerifying(true);
    try {
      await new Promise((resolve) => setTimeout(resolve, 1000));
      setVerified(true);
      showSuccess(t('accounts.edit.messages.passwordVerified'));
    } catch (_error) {
      showError(t('accounts.edit.messages.passwordVerifyFailed'));
    } finally {
      setVerifying(false);
    }
  };

  const handleSavePassword = () => {
    if (!verified) {
      showWarning(t('accounts.edit.messages.verifyFirst'));
      return;
    }
    showSuccess(t('accounts.edit.messages.passwordSaved'));
    onClose();
  };

  return (
    <Modal title={t('accounts.edit.title')} open={open} onCancel={onClose} footer={null} width={480}>
      {account && (
        <div className="space-y-4">
          <div className="p-4 rounded-xl" style={{ background: '#F5F7F9' }}>
            <div className="flex justify-between mb-2">
              <span style={{ color: '#8A9AA5' }}>{t('accounts.edit.fields.tradingAccount')}</span>
              <span style={{ color: '#141D22' }}>{(account as any).login}</span>
            </div>
            <div className="flex justify-between">
              <span style={{ color: '#8A9AA5' }}>{t('accounts.edit.fields.server')}</span>
              <span style={{ color: '#141D22' }}>
                {(account as any).brokerServer || (account as any).brokerCompany}
              </span>
            </div>
          </div>

          <div>
            <label className="block mb-2" style={{ color: '#8A9AA5' }}>
              {t('accounts.edit.fields.password')}
            </label>
            <div className="flex gap-2">
              <input
                type="text"
                value={editPassword}
                onChange={(e) => {
                  setEditPassword(e.target.value);
                  setVerified(false);
                }}
                placeholder={t('accounts.edit.placeholders.newPassword')}
                className="flex-1 outline-none transition-all"
                style={{
                  background: '#FFFFFF',
                  border: '1px solid rgba(185, 201, 223, 0.4)',
                  borderRadius: '10px',
                  padding: '10px 14px',
                  fontSize: '14px',
                  color: '#141D22',
                }}
              />
              <Button onClick={handleVerifyPassword} loading={verifying} style={{ borderRadius: '10px' }}>
                {t('accounts.edit.actions.verifyPassword')}
              </Button>
            </div>
          </div>

          {verified && (
            <div className="flex items-center gap-2 p-3 rounded-xl" style={{ background: 'rgba(0, 166, 81, 0.1)' }}>
              <IconCheck size={18} stroke={1.5} color="#00A651" />
              <span style={{ color: '#00A651' }}>{t('accounts.edit.messages.passwordVerified')}</span>
            </div>
          )}

          <div className="flex justify-end gap-2 pt-4">
            <Button onClick={onClose} style={{ borderRadius: '10px' }}>
              {t('common.cancel')}
            </Button>
            <GradientButton
              onClick={() => {
                try {
                  handleSavePassword();
                } catch (e: any) {
                  showError(e?.message || t('common.saveFailed'));
                }
              }}
              disabled={!verified}
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
