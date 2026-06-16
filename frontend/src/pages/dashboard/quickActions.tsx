import {
  BarChartOutlined, PieChartOutlined, PlusOutlined,
} from '@ant-design/icons';
import type { TFunction } from 'i18next';

export interface QuickAction {
  key: string;
  label: string;
  path: string;
  icon: React.ReactNode;
  color: string;
}

export function createQuickActions(t: TFunction): QuickAction[] {
  return [
    { key: 'bind', label: t('dashboard.quickActions.bindAccount'), path: '/accounts/bind', icon: <PlusOutlined size={22} />, color: 'rgba(212,175,55,0.1)' },
    { key: 'library', label: t('dashboard.quickActions.library'), path: '/strategy/library', icon: <PieChartOutlined size={22} />, color: 'rgba(0,166,81,0.1)' },
    { key: 'analytics', label: t('dashboard.quickActions.analytics'), path: '/analytics', icon: <BarChartOutlined size={22} />, color: 'rgba(33,150,243,0.1)' },
    { key: 'logs', label: t('dashboard.quickActions.logs'), path: '/logs', icon: <BarChartOutlined size={22} />, color: 'rgba(156,39,176,0.1)' },
  ];
}
