import { useEffect, useState } from 'react';
import { Card, Row, Col, Statistic, Table, Empty } from 'antd';
import { StatusResult } from '@/components/common/StatusResult';
import { TeamOutlined, AuditOutlined, BankOutlined, LineChartOutlined, RiseOutlined, FallOutlined } from '@ant-design/icons';
import { adminApi, type DashboardStats, type AdminLog } from '@/client/admin';
import { getErrorMessage } from '@/utils/error';
import { showError } from '@/utils/message';
import { getLogColumns } from './DashboardLogColumns';
import { DashboardRiskMetrics } from './DashboardRiskMetrics';

function toNumber(value: unknown): number {
  if (typeof value === 'bigint') return Number(value);
  if (typeof value === 'number') return value;
  return Number(value || 0);
}

export default function AdminDashboard() {
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
        const msg = getErrorMessage(err, '加载仪表盘数据失败');
        setError(msg); showError(msg);
      } finally { setLoading(false); }
    })();
  }, []);

  const logColumns = getLogColumns();

  return (
    <StatusResult loading={loading} error={error} onRetry={() => window.location.reload()}>
    <div className="space-y-6">
      <h1 className="text-2xl font-bold" style={{ color: 'var(--color-text)' }}>管理仪表盘</h1>

      <Row gutter={[16, 16]}>
        {[
          ['总用户', stats?.totalUsers || 0, TeamOutlined, '#D4AF37'],
          ['活跃用户', stats?.activeUsers || 0, AuditOutlined, '#52c41a'],
          ['MT 账户', stats?.totalAccounts || 0, BankOutlined, '#1890ff'],
          ['在线账户', stats?.onlineAccounts || 0, LineChartOutlined, '#722ed1'],
          ['今日交易', stats?.todayTrades || 0, RiseOutlined, '#13c2c2'],
        ].map(([title, value, Icon, color], i) => (
          <Col xs={12} sm={8} lg={4} key={i}>
            <Card><Statistic title={title} value={value as number} prefix={<Icon size={20} stroke={1.5} style={{ color }} />} /></Card>
          </Col>
        ))}
        <Col xs={12} sm={8} lg={4}>
          <Card>
            <Statistic title="今日盈亏" value={stats?.todayProfit || 0} precision={2}
              prefix={stats?.todayProfit >= 0 ? <RiseOutlined size={20} stroke={1.5} style={{ color: '#52c41a' }} /> : <FallOutlined size={20} stroke={1.5} style={{ color: '#ff4d4f' }} />}
              valueStyle={{ color: stats?.todayProfit >= 0 ? '#52c41a' : '#ff4d4f' }} />
          </Card>
        </Col>
      </Row>

      <Card title="最近日志">
        <Table scroll={{ x: "max-content" }} columns={logColumns} dataSource={logs} rowKey="id" pagination={false} size="small" />
      </Card>

      <DashboardRiskMetrics metrics={metrics} selectedWindow={selectedWindow} onWindowChange={setSelectedWindow} />
    </div>
    </StatusResult>
  );
}
