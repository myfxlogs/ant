import PlaceholderPage from '@/components/common/PlaceholderPage';
import { IdcardOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';

export default function AccountsList() {
  const { t } = useTranslation();
  return (
    <PlaceholderPage
      icon={<IdcardOutlined style={{ fontSize: 64 }} />}
      title={t('accounts.list', 'Accounts')}
    />
  );
}
