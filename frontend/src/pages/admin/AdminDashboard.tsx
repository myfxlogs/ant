import { Card, Col, Row, Statistic, Tag } from 'antd';
import { useTranslation } from 'react-i18next';
import { useRpcQuery } from '@/hooks/useRpcQuery';
import { StatusResult } from '@/components/common/StatusResult';
import { adminApi } from '@/client/admin';
import {
  UserOutlined,
  TeamOutlined,
  SwapOutlined,
  DollarOutlined,
} from '@ant-design/icons';

function int(v: unknown): number {
  if (typeof v === 'bigint') return Number(v);
  if (typeof v === 'number') return v;
  return 0;
}

export default function AdminDashboard() {
  const { t } = useTranslation();

  const { data, isLoading, error, refetch } = useRpcQuery(
    ['admin', 'dashboard'],
    () => adminApi.getDashboard(),
  );

  const stats = (data as Record<string, unknown> | undefined) ?? {};

  return (
    <StatusResult
      loading={isLoading}
      error={error instanceof Error ? error.message : null}
      onRetry={refetch}
    >
      <Card title={t('admin.dashboard.title', { defaultValue: 'Admin Dashboard' })}>
        <Row gutter={[16, 16]}>
          <Col xs={24} sm={12} lg={6}>
            <Card size="small">
              <Statistic
                title={t('admin.dashboard.totalUsers', { defaultValue: 'Total Users' })}
                value={int(stats.totalUsers)}
                prefix={<UserOutlined />}
              />
            </Card>
          </Col>
          <Col xs={24} sm={12} lg={6}>
            <Card size="small">
              <Statistic
                title={t('admin.dashboard.activeUsers', { defaultValue: 'Active Users' })}
                value={int(stats.activeUsers)}
                prefix={<TeamOutlined />}
                valueStyle={{ color: '#3f8600' }}
              />
            </Card>
          </Col>
          <Col xs={24} sm={12} lg={6}>
            <Card size="small">
              <Statistic
                title={t('admin.dashboard.todayTrades', { defaultValue: 'Today Trades' })}
                value={int(stats.todayTrades)}
                prefix={<SwapOutlined />}
              />
            </Card>
          </Col>
          <Col xs={24} sm={12} lg={6}>
            <Card size="small">
              <Statistic
                title={t('admin.dashboard.todayProfit', { defaultValue: 'Today Profit' })}
                value={Number(stats.todayProfit ?? 0)}
                prefix={<DollarOutlined />}
                precision={2}
                valueStyle={{ color: Number(stats.todayProfit ?? 0) >= 0 ? '#3f8600' : '#cf1322' }}
              />
            </Card>
          </Col>
        </Row>
        <Row gutter={[16, 16]} style={{ marginTop: 16 }}>
          <Col xs={24} sm={12} lg={6}>
            <Card size="small">
              <Statistic
                title={t('admin.dashboard.totalAccounts', { defaultValue: 'Total Accounts' })}
                value={int(stats.totalAccounts)}
              />
            </Card>
          </Col>
          <Col xs={24} sm={12} lg={6}>
            <Card size="small">
              <Statistic
                title={t('admin.dashboard.onlineAccounts', { defaultValue: 'Online' })}
                value={int(stats.onlineAccounts)}
                valueStyle={{ color: '#3f8600' }}
              />
            </Card>
          </Col>
          <Col xs={24} sm={12} lg={6}>
            <Card size="small">
              <Statistic
                title={t('admin.dashboard.todayVolume', { defaultValue: 'Today Volume' })}
                value={Number(stats.todayVolume ?? 0)}
                precision={2}
              />
            </Card>
          </Col>
        </Row>
      </Card>
    </StatusResult>
  );
}
