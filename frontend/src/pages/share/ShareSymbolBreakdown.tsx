import { Card, Empty, Table } from 'antd';
import { useTranslation } from 'react-i18next';
import { PieChart, Pie, Cell, ResponsiveContainer, Tooltip } from 'recharts';

const PIE_COLORS = ['#1677ff', '#52c41a', '#fa8c16', '#722ed1', '#eb2f96', '#13c2c2', '#a0d911', '#f5222d', '#2f54eb', '#faad14'];

interface SymbolStat {
  symbol: string;
  count: number;
  net: number;
}

interface TradeRecord {
  symbol?: string;
  side?: string;
  volume?: unknown;
  profit?: unknown;
  closeTimeMs?: unknown;
}

export function ShareSymbolBreakdown({ bySymbol, cardBg, pageColor, green, red, signed }: {
  bySymbol: SymbolStat[];
  cardBg: string;
  pageColor: string;
  green: string;
  red: string;
  signed: (n: number) => string;
}) {
  const { t } = useTranslation();
  if (bySymbol.length === 0) return null;
  return (
    <Card size="small" title={<span style={{ fontSize: 'clamp(12px, 2.5vw, 14px)' }}>{t('sharePage.bySymbol')}</span>} style={{ marginBottom: 16, borderRadius: 10, background: cardBg }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 16, flexWrap: 'wrap' }}>
        <ResponsiveContainer width={110} height={110}>
          <PieChart>
            <Pie data={bySymbol} dataKey="net" nameKey="symbol" cx="50%" cy="50%" innerRadius={30} outerRadius={48} paddingAngle={2} isAnimationActive={false}>
              {bySymbol.map((_, i) => <Cell key={i} fill={PIE_COLORS[i % PIE_COLORS.length]} />)}
            </Pie>
            <Tooltip formatter={((v: number | undefined, n: string) => [signed(v ?? 0), n]) as never} />
          </PieChart>
        </ResponsiveContainer>
        <div style={{ flex: 1, minWidth: 160, fontSize: 'clamp(11px, 2vw, 13px)' }}>
          {bySymbol.slice(0, 8).map((s, i) => (
            <div key={s.symbol} style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 2 }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                <div style={{ width: 8, height: 8, borderRadius: '50%', flexShrink: 0, background: PIE_COLORS[i % PIE_COLORS.length] }} />
                <span style={{ color: pageColor }}>{s.symbol}</span>
                <span style={{ color: '#8c8c8c', fontSize: '0.9em' }}>{s.count}{t('sharePage.countUnit', { defaultValue: '笔' })}</span>
              </div>
              <span style={{ fontWeight: 500, color: s.net >= 0 ? green : red }}>{signed(s.net)}</span>
            </div>
          ))}
          {bySymbol.length > 8 && <div style={{ color: '#8c8c8c', fontSize: 11 }}>+{bySymbol.length - 8} more</div>}
        </div>
      </div>
    </Card>
  );
}

export function ShareTradeTable({ trades, cardBg, columns }: {
  trades: TradeRecord[];
  cardBg: string;
  columns: { title: string; dataIndex: string; key: string; ellipsis?: boolean; render?: (v: unknown) => React.ReactNode }[];
}) {
  const { t } = useTranslation();
  return (
    <Card size="small" title={<span style={{ fontSize: 'clamp(12px, 2.5vw, 14px)' }}>{t('sharePage.tradeRecords')} ({trades.length})</span>} style={{ borderRadius: 10, background: cardBg }}>
      {trades.length === 0 ? (
        <Empty description={t('sharePage.noTrades')} image={Empty.PRESENTED_IMAGE_SIMPLE} />
      ) : (
        <Table
          dataSource={trades}
          columns={columns}
          rowKey={(_, i) => String(i)}
          size="small"
          scroll={{ x: 500 }}
          pagination={{ pageSize: 20, size: 'small', showSizeChanger: true, pageSizeOptions: ['10', '20', '50'], simple: true }}
        />
      )}
    </Card>
  );
}
