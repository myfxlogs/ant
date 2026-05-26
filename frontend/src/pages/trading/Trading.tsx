import PlaceholderPage from '@/components/common/PlaceholderPage';
import { DollarOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';

export default function Trading() {
  const { t } = useTranslation();
  return (
    <PlaceholderPage
      icon={<DollarOutlined style={{ fontSize: 64 }} />}
      title={t('trading.title', 'Trading')}
    />
  );
}
