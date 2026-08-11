import { Tag, Alert } from 'antd';
import { useTranslation } from 'react-i18next';

interface DecayBadgeProps {
  decayStatus: string;
  showDescription?: boolean;
}

export function DecayBadge({ decayStatus, showDescription = false }: DecayBadgeProps) {
  const { t } = useTranslation();

  if (!decayStatus || decayStatus === 'none') return null;

  const isDecayed = decayStatus === 'decayed';
  const color = isDecayed ? 'red' : 'orange';
  const label = isDecayed
    ? t('marketplace.decay.badgeDecayed')
    : t('marketplace.decay.badgeDecaying');

  if (showDescription) {
    const desc = isDecayed
      ? t('marketplace.decay.descDecayed')
      : t('marketplace.decay.descDecaying');
    return (
      <Alert
        type={isDecayed ? 'error' : 'warning'}
        message={label}
        description={desc}
        showIcon
        style={{ marginBottom: 12 }}
      />
    );
  }

  return <Tag color={color} style={{ marginLeft: 8 }}>{label}</Tag>;
}
