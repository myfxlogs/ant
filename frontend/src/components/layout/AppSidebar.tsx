import { useMemo, useState, useEffect } from 'react';
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
  type?: 'group';
  children?: MenuItem[];
}

/** Collect all navigable leaf keys nested under an item (flattens group children). */
function leafKeys(item: MenuItem): string[] {
  if (!item.children) return [];
  return item.children.flatMap(child => {
    if (child.type === 'group' && child.children) return leafKeys(child);
    return child.key ? [child.key] : [];
  });
}

interface LanguageOption {
  key: SupportedLanguage;
  nativeName: string;
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

  // Derive which submenus should be open based on the current path.
  // Uses leafKeys to flatten group children.
  const derivedOpenKeys = useMemo(
    () => items
      .filter(item => leafKeys(item).some(leaf => location.pathname.startsWith(leaf)))
      .map(item => item.key),
    [items, location.pathname],
  );

  const [openKeys, setOpenKeys] = useState<string[]>(derivedOpenKeys);

  // Auto-expand submenus when navigating to a child route (e.g. browser back/forward).
  // Preserves submenus the user manually opened.
  useEffect(() => {
    setOpenKeys(prev => {
      const merged = new Set([...prev, ...derivedOpenKeys]);
      const next = [...merged];
      if (next.length === prev.length && next.every((k, i) => k === prev[i])) {
        return prev; // no change → avoid re-render
      }
      return next;
    });
  }, [derivedOpenKeys]);

  const handleClick = ({ key }: { key: string }) => {
    // Navigate only when the clicked item itself has no children.
    // (The old check looked for "is this key someone else's child?" which
    // incorrectly blocked all submenu items from navigating.)
    const hasChildren = items.some(item =>
      item.key === key && item.children && item.children.length > 0,
    );
    if (!hasChildren) {
      navigate(key);
    }
  };

  return (
    <Menu
      mode="inline"
      selectedKeys={[location.pathname]}
      openKeys={openKeys}
      onOpenChange={setOpenKeys}
      items={items}
      onClick={handleClick}
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
      <div className="flex items-center gap-2 p-3 rounded-lg cursor-pointer" style={{ background: 'var(--color-bg-secondary)' }}>
        <GlobalOutlined size={18} stroke={1.5} />
        <span style={{ color: 'var(--color-text)' }}>{languages.find(l => l.key === language)?.nativeName || 'English'}</span>
      </div>
    </Dropdown>
  );

  // Mobile drawer
  if (isMobile) {
    return (
      <Drawer placement="left" open={drawerVisible} onClose={onDrawerClose} closable={false}
        styles={{ body: { padding: 0 }, header: { display: 'none' }, wrapper: { width: '280px' } }}
        style={{ background: 'var(--color-bg-main)' }}>
        <div className="h-16 flex items-center justify-center" style={{ borderBottom: '1px solid var(--color-border)' }}><BrandLogo /></div>
        <SidebarMenu items={menuItems} />
        <div className="absolute bottom-0 left-0 right-0 p-4" style={{ borderTop: '1px solid var(--color-border)' }}>{langDropdown}</div>
      </Drawer>
    );
  }

  // Desktop sidebar
  return (
    <Layout.Sider trigger={null}
      style={{ background: 'var(--color-bg-main)', borderRight: '1px solid var(--color-border)', width: 240, minWidth: 240, maxWidth: 240, position: 'fixed', left: 0, top: 0, bottom: 0 }}
      width={240}>
      <div className="h-16 flex items-center justify-center" style={{ borderBottom: '1px solid var(--color-border)' }}><BrandLogo /></div>
      <SidebarMenu items={menuItems} />
    </Layout.Sider>
  );
}

// Re-export for use in MainLayout
export { SidebarMenu };
