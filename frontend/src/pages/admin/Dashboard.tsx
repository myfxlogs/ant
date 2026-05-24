import { useEffect, useState } from 'react';
import { Card, Row, Col, Statistic, Table, Tag, Spin, Segmented, Empty } from 'antd';
import {
  IconUsers,
  IconUserCheck,
  IconBuildingBank,
  IconChartLine,
  IconTrendingUp,
  IconTrendingDown,
} from '@tabler/icons-react';
import { adminApi, type DashboardStats, type AdminLog } from '@/client/admin';
import { formatDateTime } from '@/utils/date';
import { getErrorMessage } from '@/utils/error';
import { showError } from '@/utils/message';

export default function AdminDashboard() {
  const [stats, setStats] = useState<DashboardStats | null>(null);
  const [logs, setLogs] = useState<AdminLog[]>([]);
  const [metrics, setMetrics] = useState<any>(null);
  const [selectedWindow, setSelectedWindow] = useState<string>('24h');
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchData = async () => {
      try {
        const [statsData, logsData, metricsData] = await Promise.all([
          adminApi.getDashboard(),
          adminApi.listLogs({ page: 1, pageSize: 10 }),
          adminApi.getMetrics(),
        ]);
        setStats(statsData as DashboardStats);
        setLogs(logsData.logs as AdminLog[]);
        setMetrics(metricsData || null);
      } catch (error) {
        showError(getErrorMessage(error, '加载仪表盘数据失败'));
      } finally {
        setLoading(false);
      }
    };
    fetchData();
  }, []);

  const toNumber = (value: unknown): number => {
    if (typeof value === 'bigint') {
      return Number(value);
    }
    if (typeof value === 'number') {
      return value;
    }
    return Number(value || 0);
  };

  const riskWindows = ((metrics?.app?.riskWindows as any[]) || []).map((item) => ({
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
      title: '时间',
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 180,
      render: (_text: any, record: AdminLog) => formatDateTime(record.createdAt),
    },
    {
      title: '模块',
      dataIndex: 'module',
      key: 'module',
      width: 120,
      render: (text: string) => {
        const moduleMap: Record<string, string> = {
          user_management: '用户管理',
          account_management: '账户管理',
          trading: '交易',
          system_config: '系统配置',
        };
        return moduleMap[text] || text;
      },
    },
    {
      title: '操作类型',
      dataIndex: 'actionType',
      key: 'actionType',
      width: 100,
    },
    {
      title: '目标',
      dataIndex: 'targetId',
      key: 'targetId',
      width: 200,
      ellipsis: true,
    },
    {
      title: '状态',
      dataIndex: 'success',
      key: 'success',
      width: 80,
      render: (success: boolean) => (
        <Tag color={success ? 'success' : 'error'}>
          {success ? '成功' : '失败'}
        </Tag>
      ),
    },
  ];

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <Spin size="large" />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold" style={{ color: '#141D22' }}>管理仪表盘</h1>

      <Row gutter={[16, 16]}>
        <Col xs={12} sm={8} lg={4}>
          <Card>
            <Statistic
              title="总用户"
              value={(stats as any)?.totalUsers || 0}
              prefix={<IconUsers size={20} stroke={1.5} style={{ color: '#D4AF37' }} />}
            />
          </Card>
        </Col>
        <Col xs={12} sm={8} lg={4}>
          <Card>
            <Statistic
              title="活跃用户"
              value={(stats as any)?.activeUsers || 0}
              prefix={<IconUserCheck size={20} stroke={1.5} style={{ color: '#52c41a' }} />}
            />
          </Card>
        </Col>
        <Col xs={12} sm={8} lg={4}>
          <Card>
            <Statistic
              title="MT账户"
              value={(stats as any)?.totalAccounts || 0}
              prefix={<IconBuildingBank size={20} stroke={1.5} style={{ color: '#1890ff' }} />}
            />
          </Card>
        </Col>
        <Col xs={12} sm={8} lg={4}>
          <Card>
            <Statistic
              title="在线账户"
              value={(stats as any)?.onlineAccounts || 0}
              prefix={<IconChartLine size={20} stroke={1.5} style={{ color: '#722ed1' }} />}
            />
          </Card>
        </Col>
        <Col xs={12} sm={8} lg={4}>
          <Card>
            <Statistic
              title="今日交易"
              value={(stats as any)?.todayTrades || 0}
              prefix={<IconTrendingUp size={20} stroke={1.5} style={{ color: '#13c2c2' }} />}
            />
          </Card>
        </Col>
        <Col xs={12} sm={8} lg={4}>
          <Card>
            <Statistic
              title="今日盈亏"
              value={(stats as any)?.todayProfit || 0}
              precision={2}
              prefix={(stats as any)?.todayProfit >= 0 ? <IconTrendingUp size={20} stroke={1.5} style={{ color: '#52c41a' }} /> : <IconTrendingDown size={20} stroke={1.5} style={{ color: '#ff4d4f' }} />}
              valueStyle={{ color: (stats as any)?.todayProfit >= 0 ? '#52c41a' : '#ff4d4f' }}
            />
          </Card>
        </Col>
      </Row>

      <Card title="最近操作日志">
        <Table
          scroll={{ x: "max-content" }}
          columns={logColumns}
          dataSource={logs}
          rowKey="id"
          pagination={false}
          size="small"
        />
      </Card>

      <Card title="风控执行指标（实时）">
        <Row gutter={[16, 16]}>
          <Col xs={12} sm={8} lg={6}>
            <Statistic title="风控校验总量" value={toNumber(metrics?.app?.riskValidateTotal)} />
          </Col>
          <Col xs={12} sm={8} lg={6}>
            <Statistic title="风控通过" value={toNumber(metrics?.app?.riskValidatePass)} valueStyle={{ color: '#52c41a' }} />
          </Col>
          <Col xs={12} sm={8} lg={6}>
            <Statistic title="风控拒绝" value={toNumber(metrics?.app?.riskValidateReject)} valueStyle={{ color: '#fa8c16' }} />
          </Col>
          <Col xs={12} sm={8} lg={6}>
            <Statistic title="风控异常" value={toNumber(metrics?.app?.riskValidateError)} valueStyle={{ color: '#ff4d4f' }} />
          </Col>
          <Col xs={12} sm={8} lg={6}>
            <Statistic title="下单成功" value={toNumber(metrics?.app?.orderSendSuccess)} />
          </Col>
          <Col xs={12} sm={8} lg={6}>
            <Statistic title="下单失败" value={toNumber(metrics?.app?.orderSendFailed)} valueStyle={{ color: '#ff4d4f' }} />
          </Col>
          <Col xs={12} sm={8} lg={6}>
            <Statistic title="平仓成功" value={toNumber(metrics?.app?.orderCloseSuccess)} />
          </Col>
          <Col xs={12} sm={8} lg={6}>
            <Statistic title="平仓失败" value={toNumber(metrics?.app?.orderCloseFailed)} valueStyle={{ color: '#ff4d4f' }} />
          </Col>
        </Row>
      </Card>

      <Card
        title="风控时间窗口指标（1h / 24h / 72h）"
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
              <Statistic title={`${activeWindowMetrics.window} 校验总量`} value={toNumber(activeWindowMetrics.riskValidateTotal)} />
            </Col>
            <Col xs={12} sm={8} lg={6}>
              <Statistic title={`${activeWindowMetrics.window} 通过`} value={toNumber(activeWindowMetrics.riskValidatePass)} valueStyle={{ color: '#52c41a' }} />
            </Col>
            <Col xs={12} sm={8} lg={6}>
              <Statistic title={`${activeWindowMetrics.window} 拒绝`} value={toNumber(activeWindowMetrics.riskValidateReject)} valueStyle={{ color: '#fa8c16' }} />
            </Col>
            <Col xs={12} sm={8} lg={6}>
              <Statistic title={`${activeWindowMetrics.window} 异常`} value={toNumber(activeWindowMetrics.riskValidateError)} valueStyle={{ color: '#ff4d4f' }} />
            </Col>
            <Col xs={12} sm={8} lg={6}>
              <Statistic title={`${activeWindowMetrics.window} 下单成功`} value={toNumber(activeWindowMetrics.orderSendSuccess)} />
            </Col>
            <Col xs={12} sm={8} lg={6}>
              <Statistic title={`${activeWindowMetrics.window} 下单失败`} value={toNumber(activeWindowMetrics.orderSendFailed)} valueStyle={{ color: '#ff4d4f' }} />
            </Col>
            <Col xs={12} sm={8} lg={6}>
              <Statistic title={`${activeWindowMetrics.window} 平仓成功`} value={toNumber(activeWindowMetrics.orderCloseSuccess)} />
            </Col>
            <Col xs={12} sm={8} lg={6}>
              <Statistic title={`${activeWindowMetrics.window} 平仓失败`} value={toNumber(activeWindowMetrics.orderCloseFailed)} valueStyle={{ color: '#ff4d4f' }} />
            </Col>
            <Col span={24}>
              <Table
                scroll={{ x: "max-content" }}
                size="small"
                pagination={false}
                rowKey={(row) => row.riskCode}
                dataSource={topRejectRiskCodes}
                locale={{ emptyText: <Empty description="当前窗口无拒单数据" image={Empty.PRESENTED_IMAGE_SIMPLE} /> }}
                columns={[
                  { title: `拒单 Top N 风险码（${activeWindowMetrics.window}）`, dataIndex: 'riskCode', key: 'riskCode' },
                  { title: '拒单次数', dataIndex: 'count', key: 'count', width: 160, render: (value: unknown) => toNumber(value) },
                ]}
              />
            </Col>
          </Row>
        ) : (
          <Empty description="暂无窗口指标数据" image={Empty.PRESENTED_IMAGE_SIMPLE} />
        )}
      </Card>
    </div>
  );
}
