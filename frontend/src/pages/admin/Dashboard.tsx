import { useEffect, useState } from 'react';
import { Card, Row, Col, Statistic, Table, Empty } from 'antd';
import { StatusResult } from '@/components/common/StatusResult';
import { TeamOutlined, AuditOutlined, BankOutlined, LineChartOutlined, RiseOutlined, FallOutlined } from '@ant-design/icons';
import { adminApi, type DashboardStats, type AdminLog } from '@/client/admin';
import { getErrorMessage } from '@/utils/error';
import { showError } from '@/utils/message';
import { useTranslation } from 'react-i18next';
import { getLogColumns } from './DashboardLogColumns';
import { DashboardRiskMetrics } from './DashboardRiskMetrics';

function toNumber(value: unknown): number {
  if (typeof value === 'bigint') return Number(value);
  if (typeof value === 'number') return value;
  return Number(value || 0);
}

export default function AdminDashboard() {
  const { t } = useTranslation();
  const [stats, setStats] = useState<DashboardStats | null>(null);
  const [logs, setLogs] = useState<AdminLog[]>([]);
  const [metrics, setMetrics] = useState<Record<string, any> | null>(null);
  const [selectedWindow, setSelectedWindow] = useState<string>('24h');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    setLoading(true); setError(null);
    (async () => {
      try {
        const [statsData, logsData, metricsData] = await Promise.all([
          adminApi.getDashboard(), adminApi.listLogs({ page: 1, pageSize: 10 }), adminApi.getMetrics(),
        ]);
        setStats(statsData as DashboardStats);
        setLogs(logsData.logs as AdminLog[]);
        setMetrics(metricsData || null);
      } catch (err) {
        const msg = getErrorMessage(err, t('admin.dashboard.loadFailed'));
        setError(msg); showError(msg);
      } finally { setLoading(false); }
    })();
  }, [t]);

  const logColumns = getLogColumns(t);

  return (
    <StatusResult loading={loading} error={error} onRetry={() => window.location.reload()}>
    <div className="space-y-6">
      <h1 className="text-2xl font-bold" style={{ color: '#141D22' }}>{t('admin.dashboard.title')}</h1>

      <Row gutter={[16, 16]}>
        {[
          [t('admin.dashboard.totalUsers'), stats?.totalUsers || 0, TeamOutlined, '#D4AF37'],
          [t('admin.dashboard.activeUsers'), stats?.activeUsers || 0, AuditOutlined, '#52c41a'],
          [t('admin.dashboard.mtAccounts'), stats?.totalAccounts || 0, BankOutlined, '#1890ff'],
          [t('admin.dashboard.onlineAccounts'), stats?.onlineAccounts || 0, LineChartOutlined, '#722ed1'],
          [t('admin.dashboard.todayTrades'), stats?.todayTrades || 0, RiseOutlined, '#13c2c2'],
        ].map(([title, value, Icon, color], i) => (
          <Col xs={12} sm={8} lg={4} key={i}>
            <Card><Statistic title={title} value={value as number} prefix={<Icon size={20} stroke={1.5} style={{ color }} />} /></Card>
          </Col>
        ))}
        <Col xs={12} sm={8} lg={4}>
          <Card>
            <Statistic title={t('admin.dashboard.todayProfit')} value={stats?.todayProfit || 0} precision={2}
              prefix={stats?.todayProfit >= 0 ? <RiseOutlined size={20} stroke={1.5} style={{ color: '#52c41a' }} /> : <FallOutlined size={20} stroke={1.5} style={{ color: '#ff4d4f' }} />}
              valueStyle={{ color: stats?.todayProfit >= 0 ? '#52c41a' : '#ff4d4f' }} />
          </Card>
        </Col>
      </Row>

      <Card title={t('admin.dashboard.recentLogs')}>
        <Table scroll={{ x: "max-content" }} columns={logColumns} dataSource={logs} rowKey="id" pagination={false} size="small" />
      </Card>

      <DashboardRiskMetrics metrics={metrics} selectedWindow={selectedWindow} onWindowChange={setSelectedWindow} />
    </div>
    </StatusResult>
  );
}
