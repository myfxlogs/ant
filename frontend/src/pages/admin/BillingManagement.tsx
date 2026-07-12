import { useState } from 'react';
import { Card, Table, Select, Row, Col, Statistic, Tag, Tabs } from 'antd';
import { CrownOutlined, DollarOutlined } from '@ant-design/icons';
import { useQuery } from '@tanstack/react-query';
import { adminBillingApi } from '@/client/adminBilling';
import { formatDateTime } from '@/utils/date';

const { TabPane } = Tabs;

export default function BillingManagement() {
  const [subPage, setSubPage] = useState(1);
  const [subPlan, setSubPlan] = useState('');
  const [subStatus, setSubStatus] = useState('');
  const [txPage, setTxPage] = useState(1);
  const [txType, setTxType] = useState('');

  const { data: subData, isLoading: subLoading } = useQuery({
    queryKey: ['admin', 'billing', 'subscriptions', subPage, subPlan, subStatus],
    queryFn: () => adminBillingApi.listSubscriptions({ page: subPage, pageSize: 20, plan: subPlan, status: subStatus }),
  });

  const { data: revenueData } = useQuery({
    queryKey: ['admin', 'billing', 'revenue'],
    queryFn: () => adminBillingApi.getRevenueSummary(),
  });

  const { data: txData, isLoading: txLoading } = useQuery({
    queryKey: ['admin', 'billing', 'transactions', txPage, txType],
    queryFn: () => adminBillingApi.listWalletTransactions({ page: txPage, pageSize: 20, txType }),
  });

  const subColumns = [
    { title: '用户', dataIndex: 'userEmail', key: 'userEmail', width: 200 },
    { title: '计划', dataIndex: 'planDisplayName', key: 'planDisplayName', width: 120,
      render: (v: string) => <Tag color="gold">{v}</Tag> },
    { title: '状态', dataIndex: 'status', key: 'status', width: 100,
      render: (v: string) => <Tag color={v === 'active' ? 'green' : 'default'}>{v}</Tag> },
    { title: '周期', dataIndex: 'billingCycle', key: 'billingCycle', width: 80 },
    { title: '价格', dataIndex: 'price', key: 'price', width: 80 },
    { title: '自动续费', dataIndex: 'autoRenew', key: 'autoRenew', width: 80,
      render: (v: boolean) => v ? <Tag color="blue">是</Tag> : <Tag>否</Tag> },
    { title: '当前周期开始', key: 'periodStart', width: 160,
      render: (_: any, r: any) => r.currentPeriodStart ? formatDateTime(r.currentPeriodStart) : '-' },
    { title: '当前周期结束', key: 'periodEnd', width: 160,
      render: (_: any, r: any) => r.currentPeriodEnd ? formatDateTime(r.currentPeriodEnd) : '-' },
    { title: '创建时间', key: 'createdAt', width: 160,
      render: (_: any, r: any) => r.createdAt ? formatDateTime(r.createdAt) : '-' },
  ];

  const txColumns = [
    { title: '用户', dataIndex: 'userEmail', key: 'userEmail', width: 200 },
    { title: '类型', dataIndex: 'txType', key: 'txType', width: 100,
      render: (v: string) => {
        const colors: Record<string, string> = { purchase: 'blue', sale: 'green', platform_fee: 'orange', deposit: 'cyan', withdrawal: 'red' };
        return <Tag color={colors[v] || 'default'}>{v}</Tag>;
      } },
    { title: '金额', dataIndex: 'amount', key: 'amount', width: 100,
      render: (v: string) => <span style={{ fontWeight: 600 }}>{v}</span> },
    { title: '交易前余额', dataIndex: 'balanceBefore', key: 'balanceBefore', width: 120 },
    { title: '交易后余额', dataIndex: 'balanceAfter', key: 'balanceAfter', width: 120 },
    { title: '描述', dataIndex: 'description', key: 'description', ellipsis: true },
    { title: '时间', key: 'createdAt', width: 160,
      render: (_: any, r: any) => r.createdAt ? formatDateTime(r.createdAt) : '-' },
  ];

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold" style={{ color: 'var(--color-text)' }}>计费管理</h1>

      <Row gutter={[16, 16]}>
        <Col xs={12} sm={8} lg={6}>
          <Card>
            <Statistic
              title="月度收入"
              value={revenueData?.totalMonthlyRevenue || '0'}
              prefix={<DollarOutlined style={{ color: '#52c41a' }} />}
            />
          </Card>
        </Col>
        <Col xs={12} sm={8} lg={6}>
          <Card>
            <Statistic
              title="总收入"
              value={revenueData?.totalRevenue || '0'}
              prefix={<DollarOutlined style={{ color: '#13c2c2' }} />}
            />
          </Card>
        </Col>
        <Col xs={12} sm={8} lg={6}>
          <Card>
            <Statistic
              title="活跃订阅"
              value={subData?.total || 0}
              prefix={<CrownOutlined style={{ color: '#D4AF37' }} />}
            />
          </Card>
        </Col>
        <Col xs={12} sm={8} lg={6}>
          <Card>
            <Statistic
              title="交易记录"
              value={txData?.total || 0}
            />
          </Card>
        </Col>
      </Row>

      {revenueData && revenueData.plans.length > 0 && (
        <Card title="各计划收入明细" size="small">
          <Table
            dataSource={revenueData.plans}
            rowKey="planName"
            pagination={false}
            size="small"
            columns={[
              { title: '计划', dataIndex: 'displayName', key: 'displayName', render: (v: string) => <Tag color="gold">{v}</Tag> },
              { title: '活跃数', dataIndex: 'activeCount', key: 'activeCount' },
              { title: '月度收入', dataIndex: 'monthlyRevenue', key: 'monthlyRevenue' },
              { title: '总收入', dataIndex: 'totalRevenue', key: 'totalRevenue' },
            ]}
          />
        </Card>
      )}

      <Tabs defaultActiveKey="subscriptions">
        <TabPane tab="订阅管理" key="subscriptions">
          <Card>
            <Row gutter={16} className="mb-4">
              <Col>
                <Select
                  placeholder="按计划筛选"
                  allowClear
                  style={{ width: 150 }}
                  value={subPlan || undefined}
                  onChange={(v) => { setSubPlan(v || ''); setSubPage(1); }}
                  options={[
                    { value: 'free', label: 'Free' },
                    { value: 'pro', label: 'Pro' },
                    { value: 'enterprise', label: 'Enterprise' },
                  ]}
                />
              </Col>
              <Col>
                <Select
                  placeholder="按状态筛选"
                  allowClear
                  style={{ width: 150 }}
                  value={subStatus || undefined}
                  onChange={(v) => { setSubStatus(v || ''); setSubPage(1); }}
                  options={[
                    { value: 'active', label: 'Active' },
                    { value: 'cancelled', label: 'Cancelled' },
                    { value: 'expired', label: 'Expired' },
                  ]}
                />
              </Col>
            </Row>
            <Table
              dataSource={subData?.subscriptions || []}
              columns={subColumns}
              rowKey="id"
              loading={subLoading}
              size="small"
              scroll={{ x: 'max-content' }}
              pagination={{
                current: subPage,
                total: subData?.total || 0,
                pageSize: 20,
                onChange: setSubPage,
                showTotal: (t) => `共 ${t} 条`,
              }}
            />
          </Card>
        </TabPane>
        <TabPane tab="钱包交易" key="transactions">
          <Card>
            <Row gutter={16} className="mb-4">
              <Col>
                <Select
                  placeholder="按类型筛选"
                  allowClear
                  style={{ width: 150 }}
                  value={txType || undefined}
                  onChange={(v) => { setTxType(v || ''); setTxPage(1); }}
                  options={[
                    { value: 'purchase', label: 'Purchase' },
                    { value: 'sale', label: 'Sale' },
                    { value: 'platform_fee', label: 'Platform Fee' },
                    { value: 'deposit', label: 'Deposit' },
                    { value: 'withdrawal', label: 'Withdrawal' },
                  ]}
                />
              </Col>
            </Row>
            <Table
              dataSource={txData?.transactions || []}
              columns={txColumns}
              rowKey="id"
              loading={txLoading}
              size="small"
              scroll={{ x: 'max-content' }}
              pagination={{
                current: txPage,
                total: txData?.total || 0,
                pageSize: 20,
                onChange: setTxPage,
                showTotal: (t) => `共 ${t} 条`,
              }}
            />
          </Card>
        </TabPane>
      </Tabs>
    </div>
  );
}
