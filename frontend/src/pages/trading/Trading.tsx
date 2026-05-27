import { useEffect } from 'react';
import { Row, Col } from 'antd';
import { useTranslation } from 'react-i18next';
import { useAccount } from '@/hooks/useAccount';
import { useTrading } from '@/hooks/useTrading';
import { useTradingStore } from '@/stores/tradingStore';
import AccountSummaryCard from './components/AccountSummaryCard';
import PositionsTable from './components/PositionsTable';
import PlaceOrderForm from './components/PlaceOrderForm';
import OrderHistoryTable from './components/OrderHistoryTable';

export default function Trading() {
  const { t } = useTranslation();
  const { fetchAccounts } = useAccount();
  const { fetchPositions } = useTrading();
  const currentAccountId = useTradingStore((s) => s.currentAccountId);
  const setCurrentAccountId = useTradingStore((s) => s.setCurrentAccountId);

  // Load accounts on mount.
  useEffect(() => {
    fetchAccounts().then((list) => {
      if (list && list.length > 0 && !currentAccountId) {
        const first = list[0];
        setCurrentAccountId(first.id);
      }
    });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Load positions when account changes.
  useEffect(() => {
    if (currentAccountId) {
      fetchPositions(currentAccountId);
    }
  }, [currentAccountId, fetchPositions]);

  return (
    <div style={{ padding: '0 0 24px 0' }}>
      <h2 style={{ marginBottom: 16 }}>{t('trading.title', 'Trading')}</h2>
      <AccountSummaryCard />
      <Row gutter={16}>
        <Col xs={24} lg={14}>
          <PositionsTable />
          <PlaceOrderForm />
        </Col>
        <Col xs={24} lg={10}>
          <OrderHistoryTable />
        </Col>
      </Row>
    </div>
  );
}
