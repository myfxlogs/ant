import { Tag, Collapse } from 'antd';

export interface TradeItem {
  ticket: number; symbol: string; side: string;
  closePrice?: number; price?: number; profit: number;
  closeTime?: string; created_at?: string;
}

interface Props {
  trades: TradeItem[];
}

function fmtNum(v: number | undefined | null): string {
  if (v == null || v === undefined) return '—';
  return v.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 });
}

function pnlStr(v: number): string {
  const sign = v >= 0 ? '+' : '';
  return sign + '$' + fmtNum(Math.abs(v));
}

function fmtTime(ts?: string): string {
  if (!ts) return '';
  const d = new Date(ts);
  const mm = String(d.getMonth() + 1).padStart(2, '0');
  const dd = String(d.getDate()).padStart(2, '0');
  const hh = String(d.getHours()).padStart(2, '0');
  const min = String(d.getMinutes()).padStart(2, '0');
  return `${mm}-${dd} ${hh}:${min}`;
}

export default function TradeHistorySection({ trades }: Props) {
  if (trades.length === 0) return null;

  const header = (
    <div style={{ display: 'flex', alignItems: 'center', gap: 6, fontSize: 11, fontWeight: 700, color: '#595959' }}>
      <span>🕐 Recent Trades</span>
      <span style={{ fontSize: 10, color: '#8c8c8c', fontWeight: 500 }}>({trades.length})</span>
    </div>
  );

  return (
    <div style={{ borderTop: '1px solid #e8e8e8', paddingTop: 10 }}>
      <Collapse bordered={false} defaultActiveKey={[]}
        style={{ background: 'transparent' }}
        expandIcon={() => null}
      >
        <Collapse.Panel key="history" header={header}
          style={{ border: 'none', padding: 0 }}
        >
          <div style={{ display: 'flex', flexDirection: 'column', gap: 4, paddingTop: 4 }}>
            {trades.map((t, i) => {
              const isLong = t.side === 'buy' || t.side === 'long';
              const p = t.closePrice || t.price;
              const ts = t.closeTime || t.created_at;
              return (
                <div key={`${t.ticket || i}`} style={{
                  background: '#fafbfc', border: '1px solid #e8e8e8', borderRadius: 4, padding: '6px 8px',
                }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginBottom: 4 }}>
                    <Tag color={isLong ? 'success' : 'error'} style={{ fontSize: 10, lineHeight: '16px' }}>
                      {isLong ? 'LONG' : 'SHORT'}
                    </Tag>
                    <span style={{ fontSize: 10, fontWeight: 600, color: '#262626' }}>{t.symbol}</span>
                    <span style={{ fontSize: 10, color: '#8c8c8c', marginLeft: 'auto' }}>${fmtNum(p)}</span>
                  </div>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                    <Tag color={t.profit >= 0 ? 'success' : 'error'} style={{ fontSize: 10, lineHeight: '16px' }}>
                      {pnlStr(t.profit)}
                    </Tag>
                    <span style={{ fontSize: 9, color: '#bfbfbf', marginLeft: 'auto' }}>{fmtTime(ts)}</span>
                  </div>
                </div>
              );
            })}
          </div>
        </Collapse.Panel>
      </Collapse>
    </div>
  );
}
