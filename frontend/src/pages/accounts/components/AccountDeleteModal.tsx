import { Modal, Input } from 'antd';
import { useTranslation } from 'react-i18next';

type Props = {
  open: boolean;
  deletePassword: string;
  deleting: boolean;
  onDelete: () => void;
  onCancel: () => void;
  onPasswordChange: (value: string) => void;
};

export default function AccountDeleteModal({
  open, deletePassword, deleting,
  onDelete, onCancel, onPasswordChange,
}: Props) {
  const { t } = useTranslation();

  return (
    <Modal
      title={t('accounts.detail.actions.deleteAccount')}
      open={open}
      onOk={onDelete}
      onCancel={onCancel}
      confirmLoading={deleting}
      okText={t('accounts.detail.actions.deleteConfirm')}
      cancelText={t('common.cancel')}
      okButtonProps={{ danger: true }}
      destroyOnClose
    >
      <div style={{ marginBottom: 16, color: 'var(--color-danger)' }}>{t('accounts.detail.actions.deleteWarning')}</div>
      <div style={{ marginBottom: 8, color: 'var(--color-text-muted)' }}>{t('accounts.detail.actions.deletePasswordHint')}</div>
      <Input
        placeholder={t('accounts.detail.actions.deletePasswordPlaceholder')}
        value={deletePassword}
        onChange={(e) => onPasswordChange(e.target.value)}
        onPressEnter={onDelete}
        disabled={deleting}
      />
    </Modal>
  );
}
