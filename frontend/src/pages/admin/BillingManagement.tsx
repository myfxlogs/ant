import { useState } from 'react';
import { Card, Table, Select, Row, Col, Statistic, Tag, Tabs } from 'antd';
import { CrownOutlined, DollarOutlined } from '@ant-design/icons';
import { useQuery } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { adminBillingApi } from '@/client/adminBilling';
import { formatDateTime } from '@/utils/date';

const { TabPane } = Tabs;

export default function BillingManagement() {
  const { t } = useTranslation();
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
    { title: t('admin.billing.columns.user', { defaultValue: 'User' }), dataIndex: 'userEmail', key: 'userEmail', width: 200 },
    { title: t('admin.billing.columns.plan', { defaultValue: 'Plan' }), dataIndex: 'planDisplayName', key: 'planDisplayName', width: 120,
      render: (v: string) => <Tag color="gold">{v}</Tag> },
    { title: t('admin.billing.columns.status', { defaultValue: 'Status' }), dataIndex: 'status', key: 'status', width: 100,
      render: (v: string) => <Tag color={v === 'active' ? 'green' : 'default'}>{v}</Tag> },
    { title: t('admin.billing.columns.cycle', { defaultValue: 'Cycle' }), dataIndex: 'billingCycle', key: 'billingCycle', width: 80 },
    { title: t('admin.billing.columns.price', { defaultValue: 'Price' }), dataIndex: 'price', key: 'price', width: 80 },
    { title: t('admin.billing.columns.autoRenew', { defaultValue: 'Auto Renew' }), dataIndex: 'autoRenew', key: 'autoRenew', width: 80,
      render: (v: boolean) => v ? <Tag color="blue">{t('common.yes', { defaultValue: 'Yes' })}</Tag> : <Tag>{t('common.no', { defaultValue: 'No' })}</Tag> },
    { title: t('admin.billing.columns.periodStart', { defaultValue: 'Period Start' }), key: 'periodStart', width: 160,
      render: (_: unknown, r: Record<string, unknown>) => r.currentPeriodStart ? formatDateTime(r.currentPeriodStart as string | number | bigint) : '-' },
    { title: t('admin.billing.columns.periodEnd', { defaultValue: 'Period End' }), key: 'periodEnd', width: 160,
      render: (_: unknown, r: Record<string, unknown>) => r.currentPeriodEnd ? formatDateTime(r.currentPeriodEnd as string | number | bigint) : '-' },
    { title: t('admin.billing.columns.createdAt', { defaultValue: 'Created At' }), key: 'createdAt', width: 160,
      render: (_: unknown, r: Record<string, unknown>) => r.createdAt ? formatDateTime(r.createdAt as string | number | bigint) : '-' },
  ];

  const txColumns = [
    { title: t('admin.billing.columns.user', { defaultValue: 'User' }), dataIndex: 'userEmail', key: 'userEmail', width: 200 },
    { title: t('admin.billing.columns.type', { defaultValue: 'Type' }), dataIndex: 'txType', key: 'txType', width: 100,
      render: (v: string) => {
        const colors: Record<string, string> = { purchase: 'blue', sale: 'green', platform_fee: 'orange', deposit: 'cyan', withdrawal: 'red' };
        return <Tag color={colors[v] || 'default'}>{v}</Tag>;
      } },
    { title: t('admin.billing.columns.amount', { defaultValue: 'Amount' }), dataIndex: 'amount', key: 'amount', width: 100,
      render: (v: string) => <span style={{ fontWeight: 600 }}>{v}</span> },
    { title: t('admin.billing.columns.balanceBefore', { defaultValue: 'Balance Before' }), dataIndex: 'balanceBefore', key: 'balanceBefore', width: 120 },
    { title: t('admin.billing.columns.balanceAfter', { defaultValue: 'Balance After' }), dataIndex: 'balanceAfter', key: 'balanceAfter', width: 120 },
    { title: t('admin.billing.columns.description', { defaultValue: 'Description' }), dataIndex: 'description', key: 'description', ellipsis: true },
    { title: t('admin.billing.columns.time', { defaultValue: 'Time' }), key: 'createdAt', width: 160,
      render: (_: unknown, r: Record<string, unknown>) => r.createdAt ? formatDateTime(r.createdAt as string | number | bigint) : '-' },
  ];

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold" style={{ color: 'var(--color-text)' }}>{t('admin.billing.title', { defaultValue: 'Billing Management' })}</h1>

      <Row gutter={[16, 16]}>
        <Col xs={12} sm={8} lg={6}>
          <Card>
            <Statistic
              title={t('admin.billing.monthlyRevenue', { defaultValue: 'Monthly Revenue' })}
              value={revenueData?.totalMonthlyRevenue || '0'}
              prefix={<DollarOutlined style={{ color: '#52c41a' }} />}
              formatter={(v) => `$${v}`}
            />
          </Card>
        </Col>
        <Col xs={12} sm={8} lg={6}>
          <Card>
            <Statistic
              title={t('admin.billing.totalRevenue', { defaultValue: 'Total Revenue' })}
              value={revenueData?.totalRevenue || '0'}
              prefix={<DollarOutlined style={{ color: '#13c2c2' }} />}
              formatter={(v) => `$${v}`}
            />
          </Card>
        </Col>
        <Col xs={12} sm={8} lg={6}>
          <Card>
            <Statistic
              title={t('admin.billing.activeSubs', { defaultValue: 'Active Subscriptions' })}
              value={subData?.total || 0}
              prefix={<CrownOutlined style={{ color: '#D4AF37' }} />}
            />
          </Card>
        </Col>
        <Col xs={12} sm={8} lg={6}>
          <Card>
            <Statistic
              title={t('admin.billing.txRecords', { defaultValue: 'Transactions' })}
              value={txData?.total || 0}
            />
          </Card>
        </Col>
      </Row>

      {revenueData && revenueData.plans.length > 0 && (
        <Card title={t('admin.billing.planRevenue', { defaultValue: 'Plan Revenue Details' })} size="small">
          <Table
            dataSource={revenueData.plans}
            rowKey="planName"
            pagination={false}
            size="small"
            columns={[
              { title: t('admin.billing.columns.plan', { defaultValue: 'Plan' }), dataIndex: 'displayName', key: 'displayName', render: (v: string) => <Tag color="gold">{v}</Tag> },
              { title: t('admin.billing.activeCount', { defaultValue: 'Active' }), dataIndex: 'activeCount', key: 'activeCount' },
              { title: t('admin.billing.monthlyRevenue', { defaultValue: 'Monthly Revenue' }), dataIndex: 'monthlyRevenue', key: 'monthlyRevenue' },
              { title: t('admin.billing.totalRevenue', { defaultValue: 'Total Revenue' }), dataIndex: 'totalRevenue', key: 'totalRevenue' },
            ]}
          />
        </Card>
      )}

      <Tabs defaultActiveKey="subscriptions">
        <TabPane tab={t('admin.billing.subscriptions', { defaultValue: 'Subscriptions' })} key="subscriptions">
          <Card>
            <Row gutter={16} className="mb-4">
              <Col>
                <Select
                  placeholder={t('admin.billing.filterByPlan', { defaultValue: 'Filter by plan' })}
                  allowClear
                  style={{ width: 150 }}
                  value={subPlan || undefined}
                  onChange={(v) => { setSubPlan(v || ''); setSubPage(1); }}
                  options={[
                    { value: 'free', label: t('admin.billing.planFree', { defaultValue: 'Free' }) },
                    { value: 'pro', label: t('admin.billing.planPro', { defaultValue: 'Pro' }) },
                    { value: 'enterprise', label: t('admin.billing.planEnterprise', { defaultValue: 'Enterprise' }) },
                  ]}
                />
              </Col>
              <Col>
                <Select
                  placeholder={t('admin.billing.filterByStatus', { defaultValue: 'Filter by status' })}
                  allowClear
                  style={{ width: 150 }}
                  value={subStatus || undefined}
                  onChange={(v) => { setSubStatus(v || ''); setSubPage(1); }}
                  options={[
                    { value: 'active', label: t('admin.billing.statusActive', { defaultValue: 'Active' }) },
                    { value: 'cancelled', label: t('admin.billing.statusCancelled', { defaultValue: 'Cancelled' }) },
                    { value: 'expired', label: t('admin.billing.statusExpired', { defaultValue: 'Expired' }) },
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
                showTotal: (total) => t('common.total', { total, defaultValue: `${total} total` }),
              }}
            />
          </Card>
        </TabPane>
        <TabPane tab={t('admin.billing.walletTransactions', { defaultValue: 'Wallet Transactions' })} key="transactions">
          <Card>
            <Row gutter={16} className="mb-4">
              <Col>
                <Select
                  placeholder={t('admin.billing.filterByType', { defaultValue: 'Filter by type' })}
                  allowClear
                  style={{ width: 150 }}
                  value={txType || undefined}
                  onChange={(v) => { setTxType(v || ''); setTxPage(1); }}
                  options={[
                    { value: 'purchase', label: t('admin.billing.txPurchase', { defaultValue: 'Purchase' }) },
                    { value: 'sale', label: t('admin.billing.txSale', { defaultValue: 'Sale' }) },
                    { value: 'platform_fee', label: t('admin.billing.txPlatformFee', { defaultValue: 'Platform Fee' }) },
                    { value: 'deposit', label: t('admin.billing.txDeposit', { defaultValue: 'Deposit' }) },
                    { value: 'withdrawal', label: t('admin.billing.txWithdrawal', { defaultValue: 'Withdrawal' }) },
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
                showTotal: (total) => t('common.total', { total, defaultValue: `${total} total` }),
              }}
            />
          </Card>
        </TabPane>
      </Tabs>
    </div>
  );
}
