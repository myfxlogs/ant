import { useEffect, useState } from 'react';
import { Card, Row, Col, Statistic, Table, Tag, Segmented, Empty } from 'antd';
import { StatusResult } from '@/components/common/StatusResult';
import {
  IconUsers,
  AuditOutlined,
  IconBuildingBank,
  LineChartOutlined,
  RiseOutlined,
  FallOutlined,
} from '@ant-design/icons';
import { adminApi, type DashboardStats, type AdminLog } from '@/client/admin';
import { formatDateTime } from '@/utils/date';
import { getErrorMessage } from '@/utils/error';
import { showError } from '@/utils/message';
import { useTranslation } from 'react-i18next';

interface RiskWindow {
  window: string;
  hours?: number;
  riskValidateTotal?: number;
  riskValidatePass?: number;
  riskValidateReject?: number;
  riskValidateError?: number;
  orderSendSuccess?: number;
  orderSendFailed?: number;
  orderCloseSuccess?: number;
  orderCloseFailed?: number;
  topRejectRiskCodes?: Array<{ riskCode: string; count: number }>;
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
    const fetchData = async () => {
      setLoading(true);
      setError(null);
      try {
        const [statsData, logsData, metricsData] = await Promise.all([
          adminApi.getDashboard(),
          adminApi.listLogs({ page: 1, pageSize: 10 }),
          adminApi.getMetrics(),
        ]);
        setStats(statsData as DashboardStats);
        setLogs(logsData.logs as AdminLog[]);
        setMetrics(metricsData || null);
      } catch (err) {
        const msg = getErrorMessage(err, t('admin.dashboard.loadFailed'));
        setError(msg);
        showError(msg);
      } finally {
        setLoading(false);
      }
    };
    fetchData();
  }, [t]);

  const toNumber = (value: unknown): number => {
    if (typeof value === 'bigint') {
      return Number(value);
    }
    if (typeof value === 'number') {
      return value;
    }
    return Number(value || 0);
  };

  const riskWindows: RiskWindow[] = ((metrics?.app?.riskWindows as RiskWindow[]) || []).map((item) => ({
    ...item,
    window: item?.window || `${item?.hours || 0}h`,
  }));
  const activeWindowMetrics =
    riskWindows.find((item) => item.window === selectedWindow) ||
    riskWindows.find((item) => item.window === '24h') ||
    riskWindows[0] ||
    null;
  const topRejectRiskCodes = activeWindowMetrics?.topRejectRiskCodes || [];

  const logColumns = [
    {
      title: t('admin.dashboard.logs.time'),
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 180,
      render: (_text: any, record: AdminLog) => formatDateTime(record.createdAt),
    },
    {
      title: t('admin.dashboard.logs.module'),
      dataIndex: 'module',
      key: 'module',
      width: 120,
      render: (text: string) => {
        const moduleMap: Record<string, string> = {
          user_management: t('admin.dashboard.logs.moduleMap.userManagement'),
          account_management: t('admin.dashboard.logs.moduleMap.accountManagement'),
          trading: t('admin.dashboard.logs.moduleMap.trading'),
          system_config: t('admin.dashboard.logs.moduleMap.systemConfig'),
        };
        return moduleMap[text] || text;
      },
    },
    {
      title: t('admin.dashboard.logs.actionType'),
      dataIndex: 'actionType',
      key: 'actionType',
      width: 100,
    },
    {
      title: t('admin.dashboard.logs.target'),
      dataIndex: 'targetId',
      key: 'targetId',
      width: 200,
      ellipsis: true,
    },
    {
      title: t('admin.dashboard.logs.status'),
      dataIndex: 'success',
      key: 'success',
      width: 80,
      render: (success: boolean) => (
        <Tag color={success ? 'success' : 'error'}>
          {success ? t('admin.dashboard.logs.success') : t('admin.dashboard.logs.failed')}
        </Tag>
      ),
    },
  ];

  return (
    <StatusResult loading={loading} error={error} onRetry={() => window.location.reload()}>
    <div className="space-y-6">
      <h1 className="text-2xl font-bold" style={{ color: '#141D22' }}>{t('admin.dashboard.title')}</h1>

      <Row gutter={[16, 16]}>
        <Col xs={12} sm={8} lg={4}>
          <Card>
            <Statistic
              title={t('admin.dashboard.totalUsers')}
              value={stats?.totalUsers || 0}
              prefix={<IconUsers size={20} stroke={1.5} style={{ color: '#D4AF37' }} />}
            />
          </Card>
        </Col>
        <Col xs={12} sm={8} lg={4}>
          <Card>
            <Statistic
              title={t('admin.dashboard.activeUsers')}
              value={stats?.activeUsers || 0}
              prefix={<AuditOutlined size={20} stroke={1.5} style={{ color: '#52c41a' }} />}
            />
          </Card>
        </Col>
        <Col xs={12} sm={8} lg={4}>
          <Card>
            <Statistic
              title={t('admin.dashboard.mtAccounts')}
              value={stats?.totalAccounts || 0}
              prefix={<IconBuildingBank size={20} stroke={1.5} style={{ color: '#1890ff' }} />}
            />
          </Card>
        </Col>
        <Col xs={12} sm={8} lg={4}>
          <Card>
            <Statistic
              title={t('admin.dashboard.onlineAccounts')}
              value={stats?.onlineAccounts || 0}
              prefix={<LineChartOutlined size={20} stroke={1.5} style={{ color: '#722ed1' }} />}
            />
          </Card>
        </Col>
        <Col xs={12} sm={8} lg={4}>
          <Card>
            <Statistic
              title={t('admin.dashboard.todayTrades')}
              value={stats?.todayTrades || 0}
              prefix={<RiseOutlined size={20} stroke={1.5} style={{ color: '#13c2c2' }} />}
            />
          </Card>
        </Col>
        <Col xs={12} sm={8} lg={4}>
          <Card>
            <Statistic
              title={t('admin.dashboard.todayProfit')}
              value={stats?.todayProfit || 0}
              precision={2}
              prefix={stats?.todayProfit >= 0 ? <RiseOutlined size={20} stroke={1.5} style={{ color: '#52c41a' }} /> : <FallOutlined size={20} stroke={1.5} style={{ color: '#ff4d4f' }} />}
              valueStyle={{ color: stats?.todayProfit >= 0 ? '#52c41a' : '#ff4d4f' }}
            />
          </Card>
        </Col>
      </Row>

      <Card title={t('admin.dashboard.recentLogs')}>
        <Table
          scroll={{ x: "max-content" }}
          columns={logColumns}
          dataSource={logs}
          rowKey="id"
          pagination={false}
          size="small"
        />
      </Card>

      <Card title={t('admin.dashboard.riskMetrics.title')}>
        <Row gutter={[16, 16]}>
          <Col xs={12} sm={8} lg={6}>
            <Statistic title={t('admin.dashboard.riskMetrics.riskValidateTotal')} value={toNumber(metrics?.app?.riskValidateTotal)} />
          </Col>
          <Col xs={12} sm={8} lg={6}>
            <Statistic title={t('admin.dashboard.riskMetrics.riskValidatePass')} value={toNumber(metrics?.app?.riskValidatePass)} valueStyle={{ color: '#52c41a' }} />
          </Col>
          <Col xs={12} sm={8} lg={6}>
            <Statistic title={t('admin.dashboard.riskMetrics.riskValidateReject')} value={toNumber(metrics?.app?.riskValidateReject)} valueStyle={{ color: '#fa8c16' }} />
          </Col>
          <Col xs={12} sm={8} lg={6}>
            <Statistic title={t('admin.dashboard.riskMetrics.riskValidateError')} value={toNumber(metrics?.app?.riskValidateError)} valueStyle={{ color: '#ff4d4f' }} />
          </Col>
          <Col xs={12} sm={8} lg={6}>
            <Statistic title={t('admin.dashboard.riskMetrics.orderSendSuccess')} value={toNumber(metrics?.app?.orderSendSuccess)} />
          </Col>
          <Col xs={12} sm={8} lg={6}>
            <Statistic title={t('admin.dashboard.riskMetrics.orderSendFailed')} value={toNumber(metrics?.app?.orderSendFailed)} valueStyle={{ color: '#ff4d4f' }} />
          </Col>
          <Col xs={12} sm={8} lg={6}>
            <Statistic title={t('admin.dashboard.riskMetrics.orderCloseSuccess')} value={toNumber(metrics?.app?.orderCloseSuccess)} />
          </Col>
          <Col xs={12} sm={8} lg={6}>
            <Statistic title={t('admin.dashboard.riskMetrics.orderCloseFailed')} value={toNumber(metrics?.app?.orderCloseFailed)} valueStyle={{ color: '#ff4d4f' }} />
          </Col>
        </Row>
      </Card>

      <Card
        title={t('admin.dashboard.riskWindow.title')}
        extra={(
          <Segmented
            value={selectedWindow}
            onChange={(value) => setSelectedWindow(String(value))}
            options={['1h', '24h', '72h']}
          />
        )}
      >
        {activeWindowMetrics ? (
          <Row gutter={[16, 16]}>
            <Col xs={12} sm={8} lg={6}>
              <Statistic title={t('admin.dashboard.riskWindow.validateTotal', { window: activeWindowMetrics.window })} value={toNumber(activeWindowMetrics.riskValidateTotal)} />
            </Col>
            <Col xs={12} sm={8} lg={6}>
              <Statistic title={t('admin.dashboard.riskWindow.validatePass', { window: activeWindowMetrics.window })} value={toNumber(activeWindowMetrics.riskValidatePass)} valueStyle={{ color: '#52c41a' }} />
            </Col>
            <Col xs={12} sm={8} lg={6}>
              <Statistic title={t('admin.dashboard.riskWindow.validateReject', { window: activeWindowMetrics.window })} value={toNumber(activeWindowMetrics.riskValidateReject)} valueStyle={{ color: '#fa8c16' }} />
            </Col>
            <Col xs={12} sm={8} lg={6}>
              <Statistic title={t('admin.dashboard.riskWindow.validateError', { window: activeWindowMetrics.window })} value={toNumber(activeWindowMetrics.riskValidateError)} valueStyle={{ color: '#ff4d4f' }} />
            </Col>
            <Col xs={12} sm={8} lg={6}>
              <Statistic title={t('admin.dashboard.riskWindow.orderSendSuccess', { window: activeWindowMetrics.window })} value={toNumber(activeWindowMetrics.orderSendSuccess)} />
            </Col>
            <Col xs={12} sm={8} lg={6}>
              <Statistic title={t('admin.dashboard.riskWindow.orderSendFailed', { window: activeWindowMetrics.window })} value={toNumber(activeWindowMetrics.orderSendFailed)} valueStyle={{ color: '#ff4d4f' }} />
            </Col>
            <Col xs={12} sm={8} lg={6}>
              <Statistic title={t('admin.dashboard.riskWindow.orderCloseSuccess', { window: activeWindowMetrics.window })} value={toNumber(activeWindowMetrics.orderCloseSuccess)} />
            </Col>
            <Col xs={12} sm={8} lg={6}>
              <Statistic title={t('admin.dashboard.riskWindow.orderCloseFailed', { window: activeWindowMetrics.window })} value={toNumber(activeWindowMetrics.orderCloseFailed)} valueStyle={{ color: '#ff4d4f' }} />
            </Col>
            <Col span={24}>
              <Table
                scroll={{ x: "max-content" }}
                size="small"
                pagination={false}
                rowKey={(row) => row.riskCode}
                dataSource={topRejectRiskCodes}
                locale={{ emptyText: <Empty description={t('admin.dashboard.riskWindow.noRejectData')} image={Empty.PRESENTED_IMAGE_SIMPLE} /> }}
                columns={[
                  { title: t('admin.dashboard.riskWindow.rejectRiskCodesHeader', { window: activeWindowMetrics.window }), dataIndex: 'riskCode', key: 'riskCode' },
                  { title: t('admin.dashboard.riskWindow.rejectCount'), dataIndex: 'count', key: 'count', width: 160, render: (value: unknown) => toNumber(value) },
                ]}
              />
            </Col>
          </Row>
        ) : (
          <Empty description={t('admin.dashboard.riskWindow.noData')} image={Empty.PRESENTED_IMAGE_SIMPLE} />
        )}
      </Card>
    </div>
    </StatusResult>
  );
}
