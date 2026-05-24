import { useState, useEffect } from 'react';
import { Layout, Menu, Avatar, Dropdown, Drawer } from 'antd';
import {
  IconDashboard,
  IconUsers,
  IconBuildingBank,
  IconChartLine,
  IconFileText,
  IconSettings,
  IconLogout,
  IconMenu2,
  IconArrowLeft,
} from '@tabler/icons-react';
import { useNavigate, useLocation, Outlet } from 'react-router-dom';
import { useAuth } from '@/hooks/useAuth';
import ContentContainer from '@/components/layout/ContentContainer';
import { PRIMARY_GRADIENT } from '@/components/common/GradientButton';

const { Header, Content, Sider } = Layout;

const BRAND_GRADIENT = PRIMARY_GRADIENT;

const menuItems = [
  { key: '/admin', icon: <IconDashboard size={20} stroke={1.5} />, label: '仪表盘' },
  { key: '/admin/users', icon: <IconUsers size={20} stroke={1.5} />, label: '用户管理' },
  { key: '/admin/accounts', icon: <IconBuildingBank size={20} stroke={1.5} />, label: '账户管理' },
  { key: '/admin/trading', icon: <IconChartLine size={20} stroke={1.5} />, label: '交易监控' },
  { key: '/admin/logs', icon: <IconFileText size={20} stroke={1.5} />, label: '操作日志' },
  { key: '/admin/config', icon: <IconSettings size={20} stroke={1.5} />, label: '系统配置' },
];

export default function AdminLayout() {
  const [drawerVisible, setDrawerVisible] = useState(false);
  const [isMobile, setIsMobile] = useState(false);
  const navigate = useNavigate();
  const location = useLocation();
  const { user, logout } = useAuth();

  useEffect(() => {
    const checkMobile = () => {
      setIsMobile(window.innerWidth < 992);
    };
    checkMobile();
    window.addEventListener('resize', checkMobile);
    return () => window.removeEventListener('resize', checkMobile);
  }, []);

  const userMenuItems = [
    { key: 'back', icon: <IconArrowLeft size={18} stroke={1.5} />, label: '返回用户端' },
    { type: 'divider' as const },
    { key: 'logout', icon: <IconLogout size={18} stroke={1.5} />, label: '退出登录', danger: true },
  ];

  const handleUserMenuClick = ({ key }: { key: string }) => {
    if (key === 'logout') {
      logout();
    } else if (key === 'back') {
      navigate('/');
    }
  };

  const handleMenuClick = (key: string) => {
    navigate(key);
    setDrawerVisible(false);
  };

  const menuContent = (
    <Menu
      mode="inline"
      selectedKeys={[location.pathname]}
      defaultOpenKeys={['/admin']}
      items={menuItems}
      onClick={({ key }) => handleMenuClick(key)}
      style={{ background: 'transparent', border: 'none' }}
    />
  );

  return (
    <Layout className="min-h-screen" style={{ background: '#F5F7F9' }}>
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
        style={{ background: '#1a1f36' }}
      >
        <div className="h-16 flex items-center justify-center" style={{ borderBottom: '1px solid rgba(255, 255, 255, 0.1)' }}>
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-xl flex items-center justify-center" style={{ background: BRAND_GRADIENT }}>
              <IconChartLine size={22} stroke={2} color="#FFFFFF" />
            </div>
            <span className="font-bold text-lg text-white" style={{ fontFamily: 'Poppins, sans-serif' }}>管理后台</span>
          </div>
        </div>
        <div style={{ background: '#1a1f36', minHeight: 'calc(100% - 64px)' }}>
          {menuContent}
        </div>
      </Drawer>

      {!isMobile && (
        <Sider
          trigger={null}
          style={{
            background: '#1a1f36',
            position: 'fixed',
            left: 0,
            top: 0,
            bottom: 0,
            width: 240,
            minWidth: 240,
            maxWidth: 240,
          }}
          width={240}
        >
          <div className="h-16 flex items-center justify-center" style={{ borderBottom: '1px solid rgba(255, 255, 255, 0.1)' }}>
            <div className="flex items-center gap-3">
              <div className="w-10 h-10 rounded-xl flex items-center justify-center" style={{ background: BRAND_GRADIENT }}>
                <IconChartLine size={22} stroke={2} color="#FFFFFF" />
              </div>
              <span className="font-bold text-lg text-white" style={{ fontFamily: 'Poppins, sans-serif' }}>管理后台</span>
            </div>
          </div>
          <div style={{ padding: '8px' }}>
            {menuContent}
          </div>
        </Sider>
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
                <IconMenu2 size={22} stroke={1.5} />
              </button>
            )}
            {!isMobile && (
              <div className="hidden sm:flex items-center gap-2 px-3 py-1.5 rounded-lg" style={{ background: '#fef3c7', border: '1px solid #fcd34d' }}>
                <span className="text-sm font-medium" style={{ color: '#92400e' }}>管理员模式</span>
              </div>
            )}
          </div>

          <div className="flex items-center gap-3">
            <Dropdown
              menu={{ items: userMenuItems, onClick: handleUserMenuClick }}
              placement="bottomRight"
            >
              <div className="flex items-center cursor-pointer gap-2 px-2 py-1 rounded-lg transition-colors">
                <Avatar
                  icon={<IconUsers size={22} stroke={1.5} />}
                  style={{ background: BRAND_GRADIENT }}
                  size="small"
                />
                {!isMobile && (
                  <div className="hidden sm:block">
                    <div className="text-sm font-medium" style={{ color: '#141D22' }}>{user?.nickname || user?.email?.split('@')[0] || '管理员'}</div>
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
