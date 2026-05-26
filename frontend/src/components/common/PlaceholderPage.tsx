import { Result } from 'antd';
import { useTranslation } from 'react-i18next';

interface Props {
  title?: string;
  icon?: React.ReactNode;
}

export default function PlaceholderPage({ title, icon }: Props) {
  const { t } = useTranslation();
  return (
    <Result
      icon={icon}
      title={title || t('common.comingSoon', 'Coming Soon')}
      subTitle={t('common.pageUnderDevelopment', 'This page is under development.')}
    />
  );
}
