import { useState, useEffect } from 'react';
import { Layout, Menu, Avatar, Dropdown, Drawer, Modal, Typography } from 'antd';
import {
  GlobalOutlined,
  QuestionCircleOutlined,
  UserOutlined,
  LogoutOutlined,
  SettingOutlined,
  LineChartOutlined,
  BarChartOutlined,
  PieChartOutlined,
  ShopOutlined,
  HomeOutlined,
  MenuOutlined,
  RobotOutlined,
  ThunderboltOutlined,
  HistoryOutlined,
  UnorderedListOutlined,
  FundOutlined,
  FunctionOutlined,
  ExperimentOutlined,
  FolderOutlined,
} from '@ant-design/icons';
import { useNavigate, useLocation, Outlet } from 'react-router-dom';
import { useAuth } from '@/hooks/useAuth';
import NotificationCenter from '@/components/notification/NotificationCenter';
import ContentContainer from '@/components/layout/ContentContainer';
import { PRIMARY_GRADIENT } from '@/components/common/GradientButton';
import { useTranslation } from 'react-i18next';
import i18n, { normalizeLanguage, setLanguage, type SupportedLanguage } from '@/i18n';

const { Header, Content } = Layout;

const BRAND_GRADIENT = PRIMARY_GRADIENT;

const menuKeys = {
  dashboard: '/',
  trading: '/trading',
  market: '/market',
  analytics: '/analytics',
  marketplace: '/marketplace',
  ai: '/ai',
  strategies: '/strategy/templates',
  experiments: '/strategy/experiments',
  marketRegime: '/strategy/market-regime',
  assets: '/strategy/assets',
  schedules: '/strategy/schedules',
  indicatorCatalog: '/strategy/indicator-catalog',
  logs: '/logs',
} as const;

const VITE_MENU_MODE = import.meta.env.VITE_MENU_MODE || 'full';

const languages: { key: SupportedLanguage; labelKey: string }[] = [
  { key: 'zh-cn', labelKey: 'language.simplifiedChinese' },
  { key: 'zh-tw', labelKey: 'language.traditionalChinese' },
  { key: 'en', labelKey: 'language.english' },
  { key: 'ja', labelKey: 'language.japanese' },
  { key: 'vi', labelKey: 'language.vietnamese' },
];

export default function MainLayout() {
  const [drawerVisible, setDrawerVisible] = useState(false);
  const { t } = useTranslation();
  const [language, setLanguageState] = useState<SupportedLanguage>(normalizeLanguage(i18n.language));
  const [isMobile, setIsMobile] = useState(false);
  const navigate = useNavigate();
  const location = useLocation();
  const { user, logout } = useAuth();

  const isAdmin = user?.permissions?.includes('admin:view') ?? false;

  useEffect(() => {
    const handler = (lng: string) => setLanguageState(normalizeLanguage(lng));
    i18n.on('languageChanged', handler);
    return () => {
      i18n.off('languageChanged', handler);
    };
  }, []);

  // 核心路径：Dashboard + AI + 策略 + 日志 + Marketplace
  const allMenuItems = [
    { key: menuKeys.dashboard, icon: <HomeOutlined size={20} stroke={1.5} />, label: t('menu.dashboard') },
    {
      key: menuKeys.ai,
      icon: <RobotOutlined size={20} stroke={1.5} />,
      label: t('menu.aiAssistant'),
    },
    { key: menuKeys.strategies, icon: <UnorderedListOutlined size={20} stroke={1.5} />, label: t('menu.strategies') },
    { key: menuKeys.assets, icon: <FolderOutlined size={20} stroke={1.5} />, label: t('menu.assets') },
    { key: menuKeys.schedules, icon: <ThunderboltOutlined size={20} stroke={1.5} />, label: t('menu.schedules') },
    { key: menuKeys.logs, icon: <HistoryOutlined size={20} stroke={1.5} />, label: t('menu.logs') },
  ];

  const menuItems =
    VITE_MENU_MODE === 'simple'
      ? allMenuItems.filter(item => item.key === '/')
      : allMenuItems;

  useEffect(() => {
    const checkMobile = () => {
      setIsMobile(window.innerWidth < 992);
    };
    checkMobile();
    window.addEventListener('resize', checkMobile);
    return () => window.removeEventListener('resize', checkMobile);
  }, []);

  const userMenuItems = [
    { key: 'profile', icon: <UserOutlined size={18} stroke={1.5} />, label: t('topbar.profile') },
    { key: 'settings', icon: <SettingOutlined size={18} stroke={1.5} />, label: t('topbar.settings') },
    ...(isAdmin
      ? [{ type: 'divider' as const }, { key: 'admin', icon: <LineChartOutlined size={18} stroke={1.5} />, label: t('topbar.switchToAdmin') }]
      : []),
    { type: 'divider' as const },
    { key: 'logout', icon: <LogoutOutlined size={18} stroke={1.5} />, label: t('topbar.logout'), danger: true },
  ];

  const handleUserMenuClick = ({ key }: { key: string }) => {
    if (key === 'logout') {
      logout();
    } else if (key === 'admin') {
      navigate('/admin');
    } else if (key === 'profile' || key === 'settings') {
      navigate('/profile');
    }
  };

  const handleLanguageChange = ({ key }: { key: string }) => {
    const lng = normalizeLanguage(key);
    setLanguageState(lng);
    setLanguage(lng);
  };

  const handleMenuClick = (key: string) => {
    navigate(key);
    setDrawerVisible(false);
  };

  const languageMenu = {
    items: languages.map(lang => ({
      key: lang.key,
      label: t(lang.labelKey),
      icon: language === lang.key ? <span style={{ color: '#D4AF37' }}>✓</span> : null,
    })),
    onClick: handleLanguageChange,
    selectedKeys: [language],
  };

  const menuContent = (
    <Menu
      mode="inline"
      selectedKeys={[location.pathname]}
      items={menuItems}
      onClick={({ key }) => handleMenuClick(key)}
      style={{ background: 'transparent', border: 'none' }}
    />
  );

  return (
    <Layout className="min-h-screen" style={{ background: '#F5F7F9' }}>
      {/* 移动端抽屉菜单 */}
      <Drawer
        placement="left"
        open={drawerVisible}
        onClose={() => setDrawerVisible(false)}
        closable={false}
        styles={{
          body: { padding: 0 },
          header: { display: 'none' },
          wrapper: { width: '280px' },
        }}
        style={{ background: '#FFFFFF' }}
      >
        <div className="h-16 flex items-center justify-center" style={{ borderBottom: '1px solid rgba(0, 0, 0, 0.08)' }}>
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-xl flex items-center justify-center" style={{ background: BRAND_GRADIENT }}>
              <LineChartOutlined size={22} stroke={2} color="#FFFFFF" />
            </div>
            <span className="font-bold text-lg text-gradient" style={{ fontFamily: 'Poppins, sans-serif' }}>{t('app.name')}</span>
          </div>
        </div>
        {menuContent}
        <div className="absolute bottom-0 left-0 right-0 p-4" style={{ borderTop: '1px solid rgba(0, 0, 0, 0.08)' }}>
          <Dropdown
            menu={languageMenu}
            placement="topLeft"
            trigger={['click']}
          >
            <div className="flex items-center gap-2 p-3 rounded-lg cursor-pointer" style={{ background: '#F5F7F9' }}>
              <GlobalOutlined size={18} stroke={1.5} />
              <span style={{ color: '#141D22' }}>{t(languages.find(l => l.key === language)?.labelKey || 'language.english')}</span>
            </div>
          </Dropdown>
        </div>
      </Drawer>

      {/* 桌面端侧边栏 */}
      {!isMobile && (
        <Layout.Sider 
          trigger={null} 
          style={{ 
            background: '#FFFFFF', 
            borderRight: '1px solid rgba(0, 0, 0, 0.08)',
            width: 240,
            minWidth: 240,
            maxWidth: 240,
            position: 'fixed',
            left: 0,
            top: 0,
            bottom: 0,
          }}
          width={240}
        >
          <div className="h-16 flex items-center justify-center" style={{ borderBottom: '1px solid rgba(0, 0, 0, 0.08)' }}>
            <div className="flex items-center gap-3">
              <div className="w-10 h-10 rounded-xl flex items-center justify-center" style={{ background: BRAND_GRADIENT }}>
                <LineChartOutlined size={22} stroke={2} color="#FFFFFF" />
              </div>
              <span className="font-bold text-lg text-gradient" style={{ fontFamily: 'Poppins, sans-serif' }}>{t('app.name')}</span>
            </div>
          </div>
          {menuContent}
        </Layout.Sider>
      )}

      <Layout style={{ background: 'transparent', marginLeft: isMobile ? 0 : 240 }}>
        <Header 
          className="px-4 sm:px-6 flex items-center justify-between h-14 sm:h-16" 
          style={{ 
            background: 'rgba(255, 255, 255, 0.95)', 
            backdropFilter: 'blur(12px)',
            borderBottom: '1px solid rgba(0, 0, 0, 0.08)',
            position: 'fixed',
            top: 0,
            left: isMobile ? 0 : 240,
            right: 0,
            zIndex: 100,
          }}
        >
          <div className="flex items-center gap-2">
            {isMobile && (
              <button
                onClick={() => setDrawerVisible(true)}
                className="p-2 rounded-lg transition-colors"
                style={{ color: '#5A6B75' }}
              >
                <MenuOutlined size={22} stroke={1.5} />
              </button>
            )}
            {!isMobile && (
              <div className="hidden sm:flex items-center gap-2 px-3 py-1.5 rounded-lg" style={{ background: '#F5F7F9', border: '1px solid rgba(0, 0, 0, 0.08)' }}>
                <div className="w-2 h-2 rounded-full" style={{ background: '#00A651', animation: 'pulse 2s infinite' }} />
                <span className="text-sm" style={{ color: '#5A6B75' }}>{t('topbar.systemOk')}</span>
              </div>
            )}
          </div>
          
          <div className="flex items-center gap-1 sm:gap-3">
            {isMobile && (
              <div className="flex items-center gap-2 px-2 py-1 rounded-lg" style={{ background: '#F5F7F9' }}>
                <div className="w-2 h-2 rounded-full" style={{ background: '#00A651' }} />
              </div>
            )}
            
            <Dropdown
              menu={languageMenu}
              placement="bottomRight"
              trigger={['click']}
            >
              <button
                className="p-2 rounded-lg transition-colors"
                style={{ color: '#5A6B75' }}
              >
                <GlobalOutlined size={20} stroke={1.5} />
              </button>
            </Dropdown>
            
            <NotificationCenter />
            
            {!isMobile && (
              <button
                className="p-2 rounded-lg transition-colors"
                style={{ color: '#5A6B75' }}
                onClick={() => {
                  Modal.info({
                    title: t('app.name'),
                    content: (
                      <div>
                        <Typography.Paragraph>
                          {t('common.helpText', 'AntTrader — algorithmic trading platform with AI-assisted strategy generation, multi-broker support, and risk management.')}
                        </Typography.Paragraph>
                        <Typography.Paragraph type="secondary">
                          {t('common.helpContact', 'For support, please contact your system administrator.')}
                        </Typography.Paragraph>
                      </div>
                    ),
                    okText: t('common.gotIt'),
                  });
                }}
              >
                <QuestionCircleOutlined size={20} stroke={1.5} />
              </button>
            )}
            
            <Dropdown
              menu={{ items: userMenuItems, onClick: handleUserMenuClick }}
              placement="bottomRight"
            >
              <div className="flex items-center cursor-pointer gap-2 px-2 py-1 rounded-lg transition-colors">
                <Avatar 
                  icon={<UserOutlined size={22} stroke={1.5} />} 
                  style={{ background: BRAND_GRADIENT }}
                  size="small"
                />
                {!isMobile && (
                  <div className="hidden sm:block">
                    <div className="text-sm font-medium" style={{ color: '#141D22' }}>{user?.nickname || user?.email?.split('@')[0] || t('topbar.user')}</div>
                  </div>
                )}
              </div>
            </Dropdown>
          </div>
        </Header>
        
        <Content 
          className="p-4 sm:p-6" 
          style={{ 
            marginTop: isMobile ? 56 : 64,
            minHeight: 'calc(100vh - 64px)',
            overflow: 'auto',
          }}
        >
          <ContentContainer>
            <Outlet />
          </ContentContainer>
        </Content>
      </Layout>
    </Layout>
  );
}
