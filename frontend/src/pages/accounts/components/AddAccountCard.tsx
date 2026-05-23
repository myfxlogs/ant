import { IconPlus } from '@tabler/icons-react';
import { useTranslation } from 'react-i18next';

type Props = {
  onClick: () => void;
};

export default function AddAccountCard({ onClick }: Props) {
  const { t } = useTranslation();
  return (
    <div
      onClick={onClick}
      className="rounded-2xl overflow-hidden cursor-pointer transition-all flex flex-col items-center justify-center min-h-[280px]"
      style={{
        background: '#F5F7F9',
        border: '2px dashed rgba(185, 201, 223, 0.4)',
      }}
      onMouseEnter={(e) => {
        e.currentTarget.style.background = '#E8ECF0';
        e.currentTarget.style.borderColor = 'rgba(212, 175, 55, 0.4)';
      }}
      onMouseLeave={(e) => {
        e.currentTarget.style.background = '#F5F7F9';
        e.currentTarget.style.borderColor = 'rgba(185, 201, 223, 0.4)';
      }}
    >
      <div
        className="w-16 h-16 rounded-full flex items-center justify-center mb-4"
        style={{ background: 'rgba(212, 175, 55, 0.1)' }}
      >
        <IconPlus size={32} stroke={1.5} color="#D4AF37" />
      </div>
      <span className="font-medium" style={{ color: '#8A9AA5' }}>
        {t('accounts.bindNew')}
      </span>
    </div>
  );
}
