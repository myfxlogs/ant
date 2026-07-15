import { SunOutlined, MoonOutlined } from '@ant-design/icons';
import { useUIStore } from '@/stores/uiStore';
import { useTranslation } from 'react-i18next';

export default function ThemeToggle() {
  const theme = useUIStore((s) => s.theme);
  const setTheme = useUIStore((s) => s.setTheme);
  const { t } = useTranslation();

  const isDark = theme === 'dark';

  return (
    <div
      className="flex items-center p-2 rounded-lg cursor-pointer transition-colors"
      style={{ background: 'var(--color-bg-secondary)' }}
      onClick={() => setTheme(isDark ? 'light' : 'dark')}
      title={isDark ? t('theme.switchToLight', { defaultValue: 'Switch to light mode' }) : t('theme.switchToDark', { defaultValue: 'Switch to dark mode' })}
    >
      {isDark ? (
        <SunOutlined style={{ fontSize: 18, color: '#D4AF37' }} />
      ) : (
        <MoonOutlined style={{ fontSize: 18, color: 'var(--color-text-secondary)' }} />
      )}
    </div>
  );
}
