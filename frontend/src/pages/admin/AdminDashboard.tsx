import { useState, useEffect } from 'react';
import { Card, List, Switch, Tag, message } from 'antd';
import { useTranslation } from 'react-i18next';

interface Account {
  accountId: string;
  broker: string;
  platform: string;
  state: string;
  tickRate1m: number;
  subscribedSymbols: string[];
}

const API = '/ant.v1.MtHubService';

export default function AdminPage() {
  const { t } = useTranslation();
  const [accounts, setAccounts] = useState<Account[]>([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    fetchAccounts();
  }, []);

  async function fetchAccounts() {
    setLoading(true);
    try {
      const res = await fetch(`${API}/OpenedOrders`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ accountId: '' }),
      });
      const data = await res.json();
      setAccounts(data.orders || []);
    } catch {
      message.error('Failed to load');
    } finally {
      setLoading(false);
    }
  }

  return (
    <Card title={t('admin.title', 'Admin Dashboard')}>
      <List
        loading={loading}
        dataSource={accounts}
        renderItem={(a: Account) => (
          <List.Item>
            <List.Item.Meta
              title={`${a.platform} · ${a.broker}`}
              description={
                <span>
                  <Tag color={a.state === 'connected' ? 'green' : 'red'}>{a.state}</Tag>
                  {a.tickRate1m > 0 && `${a.tickRate1m} ticks/min`}
                </span>
              }
            />
            <Switch checked={a.state === 'connected'} disabled />
          </List.Item>
        )}
      />
    </Card>
  );
}
