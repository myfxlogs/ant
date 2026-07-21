import { useState, useEffect, useCallback } from 'react';
import { Card, Row, Col, Statistic, Table, Tag, Button, Spin } from 'antd';
import { DollarOutlined, WalletOutlined, ArrowUpOutlined, ShopOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { marketplaceClient } from '@/client/connect';
import type { ProviderEarnings, ProviderTransactionItem } from '@/gen/ant/v1/marketplace_service_pb';

export default function ProviderEarningsPanel() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [earnings, setEarnings] = useState<ProviderEarnings | null>(null);
  const [transactions, setTransactions] = useState<ProviderTransactionItem[]>([]);
  const [txTotal, setTxTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(true);

  const fetchEarnings = useCallback(async () => {
    setLoading(true);
    try {
      const [e, tx] = await Promise.all([
        marketplaceClient.getProviderEarnings({}),
        marketplaceClient.listProviderTransactions({ limit: 10, offset: (page - 1) * 10 }),
      ]);
      setEarnings(e);
      setTransactions(tx.transactions || []);
      setTxTotal(tx.total || 0);
    } catch {
      // ignore
    } finally {
      setLoading(false);
    }
  }, [page]);

  useEffect(() => {
    fetchEarnings();
  }, [fetchEarnings]);

  if (loading) return <Spin style={{ display: 'block', margin: '40px auto' }} />;
  if (!earnings) return null;

  const cols = [
    {
      title: t('marketplace.earnings.colType', { defaultValue: 'Type' }),
      dataIndex: 'txType', key: 'txType',
      render: (v: string) => {
        const color = v === 'sale' ? 'green' : v === 'refund_reversal' ? 'red' : 'default';
        return <Tag color={color}>{v}</Tag>;
      },
    },
    {
      title: t('marketplace.earnings.colAmount', { defaultValue: 'Amount' }),
      dataIndex: 'amount', key: 'amount',
      render: (v: string) => `¥${Number(v || 0).toFixed(2)}`,
    },
    {
      title: t('marketplace.earnings.colStrategy', { defaultValue: 'Strategy' }),
      dataIndex: 'strategyTitle', key: 'strategyTitle', ellipsis: true,
    },
    {
      title: t('marketplace.earnings.colBuyer', { defaultValue: 'Buyer' }),
      dataIndex: 'buyerName', key: 'buyerName', ellipsis: true,
    },
    {
      title: t('marketplace.earnings.colDate', { defaultValue: 'Date' }),
      dataIndex: 'createdAt', key: 'createdAt',
    },
  ];

  return (
    <div style={{ marginBottom: 20 }}>
      <Row gutter={[12, 12]} style={{ marginBottom: 16 }}>
        <Col xs={12} sm={6}>
          <Card size="small" style={{ borderRadius: 12, border: 'none', background: '#f6ffed' }}>
            <Statistic
              title={t('marketplace.earnings.total', { defaultValue: 'Total Earnings' })}
              value={`¥${Number(earnings.totalEarnings || 0).toFixed(2)}`}
              prefix={<DollarOutlined />}
            />
          </Card>
        </Col>
        <Col xs={12} sm={6}>
          <Card size="small" style={{ borderRadius: 12, border: 'none', background: '#e6f7ff' }}>
            <Statistic
              title={t('marketplace.earnings.available', { defaultValue: 'Available Balance' })}
              value={`¥${Number(earnings.availableBalance || 0).toFixed(2)}`}
              prefix={<WalletOutlined />}
            />
          </Card>
        </Col>
        <Col xs={12} sm={6}>
          <Card size="small" style={{ borderRadius: 12, border: 'none', background: '#fff7e6' }}>
            <Statistic
              title={t('marketplace.earnings.pending', { defaultValue: 'Pending Withdrawal' })}
              value={`¥${Number(earnings.pendingWithdrawal || 0).toFixed(2)}`}
              prefix={<ArrowUpOutlined />}
            />
          </Card>
        </Col>
        <Col xs={12} sm={6}>
          <Card size="small" style={{ borderRadius: 12, border: 'none', background: '#f0f5ff' }}>
            <Statistic
              title={t('marketplace.earnings.lifetime', { defaultValue: 'Lifetime Withdrawn' })}
              value={`¥${Number(earnings.lifetimeWithdrawn || 0).toFixed(2)}`}
              prefix={<ShopOutlined />}
            />
          </Card>
        </Col>
      </Row>

      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8 }}>
        <span style={{ fontWeight: 600 }}>{t('marketplace.earnings.history', { defaultValue: 'Transaction History' })}</span>
        <Button type="primary" icon={<WalletOutlined />} onClick={() => navigate('/wallet')}>
          {t('marketplace.earnings.withdraw', { defaultValue: 'Withdraw' })}
        </Button>
      </div>

      <Table
        dataSource={transactions}
        columns={cols}
        rowKey="id"
        size="small"
        pagination={{
          current: page,
          total: txTotal,
          pageSize: 10,
          onChange: setPage,
          showTotal: (total) => `${total} ${t('marketplace.earnings.records', { defaultValue: 'records' })}`,
        }}
      />
    </div>
  );
}
