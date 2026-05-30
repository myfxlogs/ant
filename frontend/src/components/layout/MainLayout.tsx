import { useState, useEffect } from 'react';
import { Layout } from 'antd';
import { HomeOutlined, RobotOutlined, ThunderboltOutlined, HistoryOutlined, UnorderedListOutlined, FolderOutlined } from '@ant-design/icons';
import { Outlet } from 'react-router-dom';
import ContentContainer from '@/components/layout/ContentContainer';
import { useTranslation } from 'react-i18next';
import i18n, { normalizeLanguage, setLanguage, type SupportedLanguage } from '@/i18n';
import AppSidebar from '@/components/layout/AppSidebar';
import TopBar from '@/components/layout/TopBar';

const { Content } = Layout;

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

  useEffect(() => {
    const handler = (lng: string) => setLanguageState(normalizeLanguage(lng));
    i18n.on('languageChanged', handler);
    return () => { i18n.off('languageChanged', handler); };
  }, []);

  useEffect(() => {
    const checkMobile = () => setIsMobile(window.innerWidth < 992);
    checkMobile();
    window.addEventListener('resize', checkMobile);
    return () => window.removeEventListener('resize', checkMobile);
  }, []);

  const menuItems = [
    { key: '/', icon: <HomeOutlined size={20} stroke={1.5} />, label: t('menu.dashboard') },
    { key: '/ai', icon: <RobotOutlined size={20} stroke={1.5} />, label: t('menu.aiAssistant') },
    { key: '/strategy/templates', icon: <UnorderedListOutlined size={20} stroke={1.5} />, label: t('menu.strategies') },
    { key: '/strategy/assets', icon: <FolderOutlined size={20} stroke={1.5} />, label: t('menu.assets') },
    { key: '/strategy/schedules', icon: <ThunderboltOutlined size={20} stroke={1.5} />, label: t('menu.schedules') },
    { key: '/logs', icon: <HistoryOutlined size={20} stroke={1.5} />, label: t('menu.logs') },
  ];

  const handleLanguageChange = ({ key }: { key: string }) => {
    setLanguageState(normalizeLanguage(key));
    setLanguage(normalizeLanguage(key));
  };

  const languageMenu = {
    items: languages.map(lang => ({ key: lang.key, label: t(lang.labelKey), icon: language === lang.key ? <span style={{ color: '#D4AF37' }}>✓</span> : null })),
    onClick: handleLanguageChange,
  };

  return (
    <Layout className="min-h-screen" style={{ background: '#F5F7F9' }}>
      <AppSidebar
        isMobile={isMobile} drawerVisible={drawerVisible}
        menuItems={menuItems} language={language} languages={languages} languageMenu={languageMenu}
        onDrawerClose={() => setDrawerVisible(false)}
        onMenuClick={(key) => { /* handled by SidebarMenu internally via navigate */ }}
      />
      <Layout style={{ background: 'transparent', marginLeft: isMobile ? 0 : 240 }}>
        <TopBar isMobile={isMobile} onMenuToggle={() => setDrawerVisible(true)} />
        <Content className="pt-14 sm:pt-16 px-0">
          <ContentContainer>
            <Outlet />
          </ContentContainer>
        </Content>
      </Layout>
    </Layout>
  );
}
