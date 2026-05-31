import { Layout, Menu, Drawer, Dropdown } from 'antd';
import { GlobalOutlined, LineChartOutlined } from '@ant-design/icons';
import { useNavigate, useLocation } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { PRIMARY_GRADIENT } from '@/components/common/GradientButton';
import type { SupportedLanguage } from '@/i18n';

interface MenuItem {
  key: string;
  icon: React.ReactNode;
  label: string;
}

interface LanguageOption {
  key: SupportedLanguage;
  labelKey: string;
}

interface Props {
  isMobile: boolean;
  drawerVisible: boolean;
  menuItems: MenuItem[];
  language: SupportedLanguage;
  languages: LanguageOption[];
  languageMenu: { items: { key: string; label: string; icon: React.ReactNode | null }[]; onClick: (info: { key: string }) => void };
  onDrawerClose: () => void;
  onMenuClick: (key: string) => void;
}

const BRAND_GRADIENT = PRIMARY_GRADIENT;

function BrandLogo() {
  const { t } = useTranslation();
  return (
    <div className="flex items-center gap-3">
      <div className="w-10 h-10 rounded-xl flex items-center justify-center" style={{ background: BRAND_GRADIENT }}>
        <LineChartOutlined size={22} stroke={2} color="#FFFFFF" />
      </div>
      <span className="font-bold text-lg text-gradient" style={{ fontFamily: 'Poppins, sans-serif' }}>{t('app.name')}</span>
    </div>
  );
}

function SidebarMenu({ items }: { items: MenuItem[] }) {
  const location = useLocation();
  const navigate = useNavigate();
  return (
    <Menu
      mode="inline"
      selectedKeys={[location.pathname]}
      items={items}
      onClick={({ key }) => navigate(key)}
      style={{ background: 'transparent', border: 'none' }}
    />
  );
}

export default function AppSidebar({
  isMobile, drawerVisible, menuItems, language, languages, languageMenu,
  onDrawerClose, onMenuClick,
}: Props) {
  const { t } = useTranslation();

  const langDropdown = (
    <Dropdown menu={languageMenu} placement="topLeft" trigger={['click']}>
      <div className="flex items-center gap-2 p-3 rounded-lg cursor-pointer" style={{ background: '#F5F7F9' }}>
        <GlobalOutlined size={18} stroke={1.5} />
        <span style={{ color: '#141D22' }}>{t(languages.find(l => l.key === language)?.labelKey || 'language.english')}</span>
      </div>
    </Dropdown>
  );

  // Mobile drawer
  if (isMobile) {
    return (
      <Drawer placement="left" open={drawerVisible} onClose={onDrawerClose} closable={false}
        styles={{ body: { padding: 0 }, header: { display: 'none' }, wrapper: { width: '280px' } }}
        style={{ background: '#FFFFFF' }}>
        <div className="h-16 flex items-center justify-center" style={{ borderBottom: '1px solid rgba(0,0,0,0.08)' }}><BrandLogo /></div>
        <SidebarMenu items={menuItems} />
        <div className="absolute bottom-0 left-0 right-0 p-4" style={{ borderTop: '1px solid rgba(0,0,0,0.08)' }}>{langDropdown}</div>
      </Drawer>
    );
  }

  // Desktop sidebar
  return (
    <Layout.Sider trigger={null}
      style={{ background: '#FFFFFF', borderRight: '1px solid rgba(0,0,0,0.08)', width: 240, minWidth: 240, maxWidth: 240, position: 'fixed', left: 0, top: 0, bottom: 0 }}
      width={240}>
      <div className="h-16 flex items-center justify-center" style={{ borderBottom: '1px solid rgba(0,0,0,0.08)' }}><BrandLogo /></div>
      <SidebarMenu items={menuItems} />
    </Layout.Sider>
  );
}

// Re-export for use in MainLayout
export { SidebarMenu };
