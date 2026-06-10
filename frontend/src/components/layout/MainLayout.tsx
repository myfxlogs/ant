import { useState, useEffect, useMemo } from 'react';
import { Layout } from 'antd';
import { HomeOutlined, ThunderboltOutlined, HistoryOutlined, UnorderedListOutlined, FolderOutlined, CodeOutlined, ShopOutlined, BulbOutlined, SettingOutlined, ExperimentOutlined, RadarChartOutlined, DashboardOutlined, PieChartOutlined, WalletOutlined } from '@ant-design/icons';
import { Outlet } from 'react-router-dom';
import ContentContainer from '@/components/layout/ContentContainer';
import { useTranslation } from 'react-i18next';
import i18n, { normalizeLanguage, setLanguage, type SupportedLanguage } from '@/i18n';
import AppSidebar from '@/components/layout/AppSidebar';
import TopBar from '@/components/layout/TopBar';
import { useQuery } from '@tanstack/react-query';
import { queryKeys } from '@/queries/queryKeys';
import { accountApi } from '@/client/account';
import type { Account } from '@/types/account';

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

  // Share React Query cache with Dashboard — no duplicate fetches.
  // If Dashboard already fetched, TQ returns cached data immediately.
  const { data: accounts = [] } = useQuery<Account[]>({
    queryKey: queryKeys.accounts.list(),
    queryFn: () => accountApi.list(),
    staleTime: Infinity,
  });

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

  const accountChildren = useMemo(() =>
    accounts
      .filter(a => !a.isDisabled)
      .map(a => ({
        key: `/accounts/${a.id}`,
        icon: <WalletOutlined size={20} stroke={1.5} />,
        label: `${a.login} · ${a.brokerCompany ?? ''}`,
      })),
    [accounts],
  );

  const menuItems = useMemo(() => [
    { key: '/', icon: <HomeOutlined size={20} stroke={1.5} />, label: t('menu.dashboard') },
    ...(accountChildren.length > 0 ? [{
      key: '/accounts', icon: <WalletOutlined size={20} stroke={1.5} />, label: t('menu.accounts'),
      children: accountChildren,
    }] : []),
    {
      key: '/strategy', icon: <CodeOutlined size={20} stroke={1.5} />, label: t('menu.strategy'),
      children: [
        { type: 'group' as const, label: t('menu.devGroup'),
          children: [
            { key: '/strategy/templates', icon: <UnorderedListOutlined size={20} stroke={1.5} />, label: t('menu.strategies') },
            { key: '/strategy/workspace', icon: <CodeOutlined size={20} stroke={1.5} />, label: t('menu.strategyWorkspace') },
            { key: '/strategy/experiments', icon: <ExperimentOutlined size={20} stroke={1.5} />, label: t('menu.experiments') },
          ]},
        { type: 'group' as const, label: t('menu.opsGroup'),
          children: [
            { key: '/strategy/schedules', icon: <ThunderboltOutlined size={20} stroke={1.5} />, label: t('menu.schedules') },
            { key: '/strategy/market-tools', icon: <RadarChartOutlined size={20} stroke={1.5} />, label: t('menu.marketTools') },
          ]},
      ],
    },
    { key: '/wallet', icon: <WalletOutlined size={20} stroke={1.5} />, label: t('menu.wallet') },
    { key: '/trading/algos', icon: <DashboardOutlined size={20} stroke={1.5} />, label: t('menu.algoDashboard') },
    { key: '/auto-trading', icon: <SettingOutlined size={20} stroke={1.5} />, label: t('menu.autoTrading') },
    { key: '/analytics', icon: <PieChartOutlined size={20} stroke={1.5} />, label: t('menu.analytics') },
    { key: '/marketplace', icon: <ShopOutlined size={20} stroke={1.5} />, label: t('menu.marketplace') },
    { key: '/logs', icon: <HistoryOutlined size={20} stroke={1.5} />, label: t('menu.logs') },
  ], [t, accountChildren]);

  const handleLanguageChange = ({ key }: { key: string }) => {
    setLanguageState(normalizeLanguage(key));
    setLanguage(normalizeLanguage(key));
  };

  const languageMenu = {
    items: languages.map(lang => ({ key: lang.key, label: t(lang.labelKey), icon: language === lang.key ? <span style={{ color: '#D4AF37' }}>✓</span> : null })),
    onClick: handleLanguageChange,
  };

  return (
    <Layout className="min-h-screen" style={{ background: 'var(--color-bg-secondary)' }}>
      <AppSidebar
        isMobile={isMobile} drawerVisible={drawerVisible}
        menuItems={menuItems} language={language} languages={languages} languageMenu={languageMenu}
        onDrawerClose={() => setDrawerVisible(false)}
        onMenuClick={(key) => { /* handled by SidebarMenu internally via navigate */ }}
      />
      <Layout style={{ background: 'transparent', marginLeft: isMobile ? 0 : 240 }}>
        <TopBar isMobile={isMobile} onMenuToggle={() => setDrawerVisible(true)} language={language} languages={languages} languageMenu={languageMenu} />
        <Content className="pt-14 sm:pt-16 px-0">
          <ContentContainer>
            <Outlet />
          </ContentContainer>
        </Content>
      </Layout>
    </Layout>
  );
}
