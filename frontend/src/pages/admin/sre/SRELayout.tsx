import { Tabs } from 'antd';
import { StopOutlined, ThunderboltOutlined, ExperimentOutlined } from '@ant-design/icons';
import { Outlet, useLocation, useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';

export default function SRELayout() {
  const location = useLocation();
  const navigate = useNavigate();
  const { t } = useTranslation();

  const items = [
    { key: '/admin/sre/killswitch', label: <span><StopOutlined /> {t('sre.killSwitch.title', { defaultValue: 'Kill Switch' })}</span> },
    { key: '/admin/sre/breakers', label: <span><ThunderboltOutlined /> {t('sre.breakers.title', { defaultValue: 'Breakers' })}</span> },
    { key: '/admin/sre/canary', label: <span><ExperimentOutlined /> {t('sre.canary.title', { defaultValue: 'Canary' })}</span> },
  ];

  const activeKey = (() => {
    const p = location.pathname || '';
    if (p.startsWith('/admin/sre/breakers')) return '/admin/sre/breakers';
    if (p.startsWith('/admin/sre/canary')) return '/admin/sre/canary';
    return '/admin/sre/killswitch';
  })();

  return (
    <div>
      <Tabs activeKey={activeKey} items={items} onChange={key => navigate(key)} />
      <Outlet />
    </div>
  );
}
