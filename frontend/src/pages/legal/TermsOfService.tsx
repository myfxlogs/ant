import PlaceholderPage from '@/components/common/PlaceholderPage';
import { FileTextOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';

export default function TermsOfService() {
  const { t } = useTranslation();
  return (
    <PlaceholderPage
      icon={<FileTextOutlined style={{ fontSize: 64 }} />}
      title={t('legal.terms', 'Terms of Service')}
    />
  );
}
