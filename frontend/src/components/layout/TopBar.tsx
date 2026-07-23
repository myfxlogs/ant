import { Layout, Avatar, Dropdown } from 'antd';
import { MenuOutlined, UserOutlined, SettingOutlined, LogoutOutlined, LineChartOutlined, GlobalOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '@/hooks/useAuth';
import NotificationCenter from '@/components/notification/NotificationCenter';
import WalletDropdown from '@/components/wallet/WalletDropdown';
import ThemeToggle from '@/components/theme/ThemeToggle';
import { isAdmin } from '@/constants/permissions';
import { useTranslation } from 'react-i18next';
import type { SupportedLanguage } from '@/i18n';

const { Header } = Layout;

interface LanguageOption {
  key: SupportedLanguage;
  nativeName: string;
}

interface Props {
  isMobile: boolean;
  onMenuToggle: () => void;
  language: SupportedLanguage;
  languages: LanguageOption[];
  languageMenu: { items: { key: string; label: string; icon: React.ReactNode | null }[]; onClick: (info: { key: string }) => void };
}

export default function TopBar({ isMobile, onMenuToggle, language, languages, languageMenu }: Props) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { user, logout } = useAuth();
  const admin = isAdmin(user?.permissions);

  const userMenuItems = [
    { key: 'profile', icon: <UserOutlined size={18} />, label: t('topbar.profile') },
    { key: 'settings', icon: <SettingOutlined size={18} />, label: t('topbar.settings') },
    ...(admin ? [{ type: 'divider' as const }, { key: 'admin', icon: <LineChartOutlined size={18} />, label: t('topbar.switchToAdmin') }] : []),
    { type: 'divider' as const },
    { key: 'logout', icon: <LogoutOutlined size={18} />, label: t('topbar.logout'), danger: true },
  ];

  const handleUserMenu = ({ key }: { key: string }) => {
    if (key === 'logout') logout();
    else if (key === 'admin') navigate('/admin');
    else if (key === 'profile' || key === 'settings') navigate('/profile');
  };

  return (
    <Header className="px-4 sm:px-6 flex items-center justify-between h-14 sm:h-16"
      style={{ background: 'var(--color-bg-header)', backdropFilter: 'blur(12px)', borderBottom: '1px solid var(--color-border)', position: 'fixed', top: 0, left: isMobile ? 0 : 240, right: 0, zIndex: 100 }}>
      <div className="flex items-center gap-2">
        {isMobile && (
          <button onClick={onMenuToggle} className="p-2 rounded-lg transition-colors" style={{ color: 'var(--color-text-secondary)' }}>
            <MenuOutlined size={22} stroke={1.5} />
          </button>
        )}
        {!isMobile && (
          <div className="hidden sm:flex items-center gap-2 px-3 py-1.5 rounded-lg" style={{ background: 'var(--color-bg-secondary)', border: '1px solid var(--color-border)' }}>
            <div className="w-2 h-2 rounded-full" style={{ background: '#00A651', animation: 'pulse 2s infinite' }} />
            <span className="text-sm" style={{ color: 'var(--color-text-secondary)' }}>{t('topbar.systemOk')}</span>
          </div>
        )}
      </div>
      <div className="flex items-center gap-1 sm:gap-3">
        <WalletDropdown />
        <ThemeToggle />
        <Dropdown menu={languageMenu} placement="bottomRight" trigger={['click']}>
          <div className="flex items-center gap-1.5 p-2 rounded-lg cursor-pointer transition-colors" style={{ background: 'var(--color-bg-secondary)' }}>
            <GlobalOutlined size={18} />
          </div>
        </Dropdown>
        <NotificationCenter />
        <Dropdown menu={{ items: userMenuItems, onClick: handleUserMenu }} placement="bottomRight" trigger={['click']}>
          <div className="flex items-center gap-2 p-1.5 rounded-lg cursor-pointer transition-colors" style={{ background: 'var(--color-bg-secondary)' }}>
            <Avatar size={32} icon={<UserOutlined />} style={{ background: '#D4AF37' }} />
            <span className="hidden sm:inline text-sm font-medium" style={{ color: 'var(--color-text)' }}>{user?.email?.split('@')[0] || user?.username || ''}</span>
          </div>
        </Dropdown>
      </div>
    </Header>
  );
}
