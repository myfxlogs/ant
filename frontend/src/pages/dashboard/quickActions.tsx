
import { QUICK_ACTIONS_ANALYTICS_KEY, QUICK_ACTIONS_BIND_ACCOUNT_KEY, QUICK_ACTIONS_LIBRARY_KEY, QUICK_ACTIONS_LOGS_KEY } from '@/gen/ant/v1/i18n/dashboard_keys';

import {
  BarChartOutlined, PieChartOutlined, PlusOutlined, RobotOutlined,
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
    { key: 'bind', label: t(QUICK_ACTIONS_BIND_ACCOUNT_KEY), path: '/accounts/bind', icon: <PlusOutlined size={22} />, color: 'rgba(212,175,55,0.1)' },
    { key: 'ai', label: t('dashboard.quickActions.aiStrategy', { defaultValue: 'AI Strategy' }), path: '/strategy/workspace', icon: <RobotOutlined size={22} />, color: 'rgba(33,150,243,0.1)' },
    { key: 'library', label: t(QUICK_ACTIONS_LIBRARY_KEY), path: '/strategy', icon: <PieChartOutlined size={22} />, color: 'rgba(0,166,81,0.1)' },
    { key: 'analytics', label: t(QUICK_ACTIONS_ANALYTICS_KEY), path: '/analytics', icon: <BarChartOutlined size={22} />, color: 'rgba(33,150,243,0.1)' },
    { key: 'logs', label: t(QUICK_ACTIONS_LOGS_KEY), path: '/logs', icon: <BarChartOutlined size={22} />, color: 'rgba(156,39,176,0.1)' },
  ];
}
