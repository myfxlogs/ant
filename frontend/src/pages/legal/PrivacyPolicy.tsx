import PlaceholderPage from '@/components/common/PlaceholderPage';
import { SafetyOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';

export default function PrivacyPolicy() {
  const { t } = useTranslation();
  return (
    <PlaceholderPage
      icon={<SafetyOutlined style={{ fontSize: 64 }} />}
      title={t('legal.privacy', 'Privacy Policy')}
    />
  );
}
