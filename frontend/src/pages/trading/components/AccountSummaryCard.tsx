import { Card, Select, Statistic, Row, Col, Tag, Typography } from 'antd';
import { WalletOutlined, DollarOutlined, PieChartOutlined, SafetyCertificateOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next'
import { TRADING_ACCOUNT_KEY, TRADING_BALANCE_KEY, TRADING_EQUITY_KEY, TRADING_FREE_MARGIN_KEY, TRADING_MARGIN_KEY, TRADING_MARGIN_LEVEL_KEY, TRADING_NO_ACCOUNT_KEY } from '@/gen/ant/v1/i18n/trading_keys';
;
import { useAccount } from '@/hooks/useAccount';
import { useTradingStore } from '@/stores/tradingStore';
import type { Account } from '@/types/account';

const { Option } = Select;
const { Text } = Typography;

export default function AccountSummaryCard() {
  const { t } = useTranslation();
  const { accounts, setCurrentAccount } = useAccount();
  const accountInfo = useTradingStore((s) => s.accountInfo);
  const currentAccountId = useTradingStore((s) => s.currentAccountId);
  const setCurrentAccountId = useTradingStore((s) => s.setCurrentAccountId);
  const setPositions = useTradingStore((s) => s.setPositions);

  const handleAccountChange = (accountId: string) => {
    setCurrentAccountId(accountId);
    setCurrentAccount(accounts.find((a: Account) => a.id === accountId) || null);
    if (accountId) {
      setPositions(accountId, []);
    }
  };

  const fmt = (v: number | undefined) =>
    v != null ? v.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 }) : '-';

  const statusTag = (() => {
    const acct = accounts.find((a: Account) => a.id === currentAccountId);
    if (!acct) return null;
    const s = acct.status;
    if (s === 'connected' || s === 'active') return <Tag color="green">{s}</Tag>;
    if (s === 'connecting') return <Tag color="blue">{s}</Tag>;
    if (s === 'disconnected' || s === 'frozen') return <Tag color="red">{s}</Tag>;
    return <Tag>{s}</Tag>;
  })();

  return (
    <Card
      title={
        <span>
          <WalletOutlined style={{ marginRight: 8 }} />
          {t(TRADING_ACCOUNT_KEY, 'Account')}
        </span>
      }
      extra={
        <Select
          value={currentAccountId ?? undefined}
          onChange={handleAccountChange}
          placeholder={t(TRADING_NO_ACCOUNT_KEY, 'No account selected')}
          style={{ width: 260 }}
          allowClear
        >
          {accounts.map((a: Account) => (
            <Option key={a.id} value={a.id}>
              {a.brokerCompany} — {a.login} ({a.mtType})
            </Option>
          ))}
        </Select>
      }
    >
      {!currentAccountId ? (
        <Text type="secondary">{t(TRADING_NO_ACCOUNT_KEY, 'No account selected')}</Text>
      ) : (
        <>
          {statusTag}
          <Row gutter={16} style={{ marginTop: 12 }}>
            <Col span={4}>
              <Statistic
                title={<span><DollarOutlined /> {t(TRADING_BALANCE_KEY, 'Balance')}</span>}
                value={fmt(accountInfo.balance)}
                precision={2}
              />
            </Col>
            <Col span={4}>
              <Statistic
                title={<span><PieChartOutlined /> {t(TRADING_EQUITY_KEY, 'Equity')}</span>}
                value={fmt(accountInfo.equity)}
                precision={2}
              />
            </Col>
            <Col span={4}>
              <Statistic
                title={<span><SafetyCertificateOutlined /> {t(TRADING_MARGIN_KEY, 'Margin')}</span>}
                value={fmt(accountInfo.margin)}
                precision={2}
              />
            </Col>
            <Col span={6}>
              <Statistic
                title={t(TRADING_FREE_MARGIN_KEY, 'Free Margin')}
                value={fmt(accountInfo.freeMargin)}
                precision={2}
              />
            </Col>
            <Col span={6}>
              <Statistic
                title={t(TRADING_MARGIN_LEVEL_KEY, 'Margin Level')}
                value={fmt(accountInfo.marginLevel)}
                suffix="%"
                precision={2}
              />
            </Col>
          </Row>
        </>
      )}
    </Card>
  );
}
