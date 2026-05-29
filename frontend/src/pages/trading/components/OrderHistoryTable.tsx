import { useCallback, useEffect, useState, useMemo } from 'react';
import { Table, Tag, Card, Pagination } from 'antd';
import { HistoryOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { useTradingStore } from '@/stores/tradingStore';
import { useTrading } from '@/hooks/useTrading';

export default function OrderHistoryTable() {
  const { t } = useTranslation();
  const currentAccountId = useTradingStore((s) => s.currentAccountId);
  const { getOrderHistory } = useTrading();
  const [orders, setOrders] = useState<unknown[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(false);
  const pageSize = 20;

  const loadHistory = useCallback(async (accountId: string, p: number) => {
    setLoading(true);
    try {
      const r = await getOrderHistory({ accountId, page: p, pageSize });
      setOrders(r.orders);
      setTotal(r.total);
    } catch {
      setOrders([]);
      setTotal(0);
    } finally {
      setLoading(false);
    }
  }, [getOrderHistory, pageSize]);

  useEffect(() => {
    if (!currentAccountId) {
      setOrders([]);
      setTotal(0);
      return;
    }
    loadHistory(currentAccountId, page);
  }, [currentAccountId, page, loadHistory]);

  const fmtTime = (v: unknown) => {
    if (!v) return '-';
    const ts = Number(v);
    if (!ts || ts <= 0) return '-';
    return new Date(ts * 1000).toLocaleString();
  };

  const columns = useMemo(() => [
    { title: '#', dataIndex: 'ticket', key: 'ticket', width: 80, render: (v: unknown) => String(v ?? '-') },
    { title: t('trading.symbol', 'Symbol'), dataIndex: 'symbol', key: 'symbol', width: 100 },
    {
      title: t('trading.type', 'Type'),
      dataIndex: 'type',
      key: 'type',
      width: 70,
      render: (v: unknown) => {
        const s = String(v ?? '');
        const color = s.includes('buy') ? 'green' : 'red';
        return <Tag color={color}>{s.toUpperCase()}</Tag>;
      },
    },
    { title: t('trading.volume', 'Volume'), dataIndex: 'volume', key: 'volume', width: 80, render: (v: unknown) => Number(v ?? 0).toFixed(2) },
    { title: t('trading.price', 'Price'), dataIndex: 'openPrice', key: 'openPrice', width: 90, render: (v: unknown) => Number(v ?? 0).toFixed(5) },
    {
      title: 'P&L',
      dataIndex: 'profit',
      key: 'profit',
      width: 90,
      render: (v: unknown) => {
        const n = Number(v ?? 0);
        const color = n >= 0 ? '#52c41a' : '#ff4d4f';
        return <span style={{ color, fontWeight: 600 }}>{n.toFixed(2)}</span>;
      },
    },
    { title: 'Time', dataIndex: 'openTime', key: 'openTime', width: 160, render: (v: unknown) => fmtTime(v) },
  ], [t]);

  return (
    <Card
      title={<span><HistoryOutlined style={{ marginRight: 8 }} />{t('trading.orderHistory', 'Order History')}</span>}
      style={{ marginTop: 16 }}
    >
      <Table
        dataSource={orders as Record<string, unknown>[]}
        columns={columns}
        rowKey="ticket"
        loading={loading}
        size="small"
        pagination={false}
        locale={{ emptyText: t('trading.noOrders', 'No orders yet') }}
        tableLayout="fixed"
      />
      {total > pageSize && (
        <Pagination
          current={page}
          total={total}
          pageSize={pageSize}
          onChange={(p) => setPage(p)}
          style={{ marginTop: 12, textAlign: 'right' }}
          showTotal={(t2) => `${t2} orders`}
        />
      )}
    </Card>
  );
}
