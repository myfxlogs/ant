import { Modal, Input } from 'antd';
import { useTranslation } from 'react-i18next'
import { DETAIL_ACTIONS_DELETE_ACCOUNT_KEY, DETAIL_ACTIONS_DELETE_CONFIRM_KEY, DETAIL_ACTIONS_DELETE_PASSWORD_HINT_KEY, DETAIL_ACTIONS_DELETE_PASSWORD_PLACEHOLDER_KEY, DETAIL_ACTIONS_DELETE_WARNING_KEY } from '@/gen/ant/v1/i18n/accounts_keys';

;

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
      title={t(DETAIL_ACTIONS_DELETE_TRADING_ACCOUNT_KEY)}
      open={open}
      onOk={onDelete}
      onCancel={onCancel}
      confirmLoading={deleting}
      okText={t(DETAIL_ACTIONS_DELETE_CONFIRM_KEY)}
      cancelText={t('common.cancel')}
      okButtonProps={{ danger: true }}
      destroyOnClose
    >
      <div style={{ marginBottom: 16, color: 'var(--color-danger)' }}>{t(DETAIL_ACTIONS_DELETE_WARNING_KEY)}</div>
      <div style={{ marginBottom: 8, color: 'var(--color-text-muted)' }}>{t(DETAIL_ACTIONS_DELETE_PASSWORD_HINT_KEY)}</div>
      <Input
        placeholder={t(DETAIL_ACTIONS_DELETE_PASSWORD_PLACEHOLDER_KEY)}
        value={deletePassword}
        onChange={(e) => onPasswordChange(e.target.value)}
        onPressEnter={onDelete}
        disabled={deleting}
      />
    </Modal>
  );
}
