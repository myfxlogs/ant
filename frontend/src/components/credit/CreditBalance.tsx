import { useState, useEffect, useCallback } from 'react';
import { Card, Table, Tag, Statistic } from 'antd';
import { useTranslation } from 'react-i18next';
import { creditApi } from '@/client/credit';
import type { CreditTransaction } from '@/gen/ant/v1/credit_pb';

const TX_TYPE_COLORS: Record<string, string> = {
  deposit: 'green',
  subscription_grant: 'blue',
  free_grant: 'blue',
  ai_usage: 'orange',
  ai_hold: 'gold',
  ai_release: 'cyan',
  refund: 'purple',
  adjustment: 'default',
};

export default function CreditBalance() {
  const { t } = useTranslation();
  const [balance, setBalance] = useState('0');
  const [frozen, setFrozen] = useState('0');
  const [transactions, setTransactions] = useState<CreditTransaction[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(false);

  const loadBalance = useCallback(async () => {
    try {
      const resp = await creditApi.getBalance();
      setBalance(resp.balance);
      setFrozen(resp.frozenBalance);
    } catch {
      // fail silently
    }
  }, []);

  const loadTransactions = useCallback(async () => {
    setLoading(true);
    try {
      const resp = await creditApi.listTransactions(page, 20);
      setTransactions(resp.transactions);
      setTotal(Number(resp.total));
    } catch {
      // fail silently
    } finally {
      setLoading(false);
    }
  }, [page]);

  useEffect(() => {
    loadBalance();
  }, [loadBalance]);

  useEffect(() => {
    loadTransactions();
  }, [loadTransactions]);

  const formatCredits = (s: string) => {
    const n = parseFloat(s);
    if (isNaN(n)) return s;
    return n.toFixed(0);
  };

  const columns = [
    {
      title: t('common.credit.type'),
      dataIndex: 'txType',
      render: (type: string) => <Tag color={TX_TYPE_COLORS[type] || 'default'}>{type}</Tag>,
    },
    {
      title: t('common.credit.amount'),
      dataIndex: 'amount',
      render: (amt: string) => {
        const n = parseFloat(amt);
        const sign = n >= 0 ? '+' : '';
        return `${sign}${formatCredits(amt)}`;
      },
    },
    {
      title: t('common.credit.balanceAfter'),
      dataIndex: 'balanceAfter',
      render: (s: string) => formatCredits(s),
    },
    {
      title: t('common.credit.description'),
      dataIndex: 'description',
    },
    {
      title: t('common.credit.time'),
      dataIndex: 'createdAtTsMs',
      render: (ms: bigint) => new Date(Number(ms)).toLocaleString(),
    },
  ];

  return (
    <div className="space-y-4">
      <div className="flex gap-4">
        <Card size="small">
          <Statistic
            title={t('common.credit.balance')}
            value={formatCredits(balance)}
            suffix={t('common.credit.credits')}
          />
        </Card>
        <Card size="small">
          <Statistic
            title={t('common.credit.frozen')}
            value={formatCredits(frozen)}
            suffix={t('common.credit.credits')}
          />
        </Card>
      </div>

      <Card title={t('common.credit.history')} size="small">
        <Table
          columns={columns}
          dataSource={transactions}
          rowKey={(r) => r.id}
          loading={loading}
          size="small"
          pagination={{
            current: page,
            total,
            pageSize: 20,
            onChange: setPage,
            showSizeChanger: false,
          }}
        />
      </Card>
    </div>
  );
}
