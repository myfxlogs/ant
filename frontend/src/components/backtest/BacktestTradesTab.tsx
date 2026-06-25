import { Table, Empty } from 'antd';
import { useTranslation } from 'react-i18next';
import {
  CLOSE_PRICE_KEY, LONG_KEY, PNL_KEY, SHORT_KEY,
  TRADE_VOLUME_KEY,
} from '@/gen/ant/v1/i18n/strategy_backtest_params_keys';
import { BACKTEST_EMPTY_KEY } from '@/gen/ant/v1/i18n/strategy_workspace_keys';
import {
  BACKTEST_RECORDS_KEY, TRADE_PRICE_KEY, TRADE_SIDE_KEY,
} from '@/gen/ant/v1/i18n/strategy_backtest_keys';
import type { ChartTrade } from './useBacktestRunner';

interface Props {
  trades: ChartTrade[];
  panelHeight: number;
}

export default function BacktestTradesTab({ trades, panelHeight }: Props) {
  const { t } = useTranslation();

  if (trades.length === 0) {
    return <Empty description={t(BACKTEST_EMPTY_KEY, 'Run a backtest to see trades')} style={{ padding: 24 }} />;
  }

  const buys = trades.filter((t) => t.side === 'buy');
  const sells = trades.filter((t) => t.side === 'sell');
  const buyPnl = buys.reduce((s, t) => s + (t.pnl || 0), 0);
  const sellPnl = sells.reduce((s, t) => s + (t.pnl || 0), 0);
  const buyVol = buys.reduce((s, t) => s + (t.volume || 0), 0);
  const sellVol = sells.reduce((s, t) => s + (t.volume || 0), 0);

  return (
    <div>
      <div style={{ display: 'flex', gap: 12, marginBottom: 10, fontSize: 12 }}>
        <span>🟢 {t(LONG_KEY)}: <b>{buys.length}</b> {t(TRADE_VOLUME_KEY)} <b>{buyVol.toFixed(2)}</b> {t(PNL_KEY)} <b style={{ color: buyPnl >= 0 ? '#26a69a' : '#e57373' }}>{buyPnl >= 0 ? '+' : ''}{buyPnl.toFixed(2)}</b></span>
        <span>🔴 {t(SHORT_KEY)}: <b>{sells.length}</b> {t(TRADE_VOLUME_KEY)} <b>{sellVol.toFixed(2)}</b> {t(PNL_KEY)} <b style={{ color: sellPnl >= 0 ? '#26a69a' : '#e57373' }}>{sellPnl >= 0 ? '+' : ''}{sellPnl.toFixed(2)}</b></span>
      </div>
      <Table dataSource={trades.map((t, i) => ({ ...t, key: i }))}
        pagination={{ pageSize: 30, size: 'small' }} scroll={{ y: panelHeight - 180 }}
        columns={[
          { title: '#', dataIndex: 'key', width: 40 },
          { title: t(TRADE_SIDE_KEY, 'Side'), dataIndex: 'side', width: 60,
            render: (v: string) => <span style={{ color: v === 'buy' ? '#26a69a' : '#e57373' }}>{v?.toUpperCase()}</span> },
          { title: t(TRADE_VOLUME_KEY, 'Volume'), dataIndex: 'volume', width: 70,
            render: (v: number) => v?.toFixed(2) },
          { title: t(TRADE_PRICE_KEY, 'Price'), dataIndex: 'openPrice', width: 80,
            render: (v: number) => v?.toFixed(2) },
          { title: t(CLOSE_PRICE_KEY), dataIndex: 'closePrice', width: 80,
            render: (v: number) => v?.toFixed(2) ?? '—' },
          { title: t(PNL_KEY), dataIndex: 'pnl', width: 80,
            render: (v: number) => v != null ? (
              <span style={{ color: v >= 0 ? '#26a69a' : '#ef5350' }}>{v >= 0 ? '+' : ''}{v.toFixed(2)}</span>
            ) : '-' },
        ]} />
    </div>
  );
}
