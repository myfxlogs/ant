/** QuoteBar displays real-time bid/ask prices above the K-line chart. */
interface QuoteBarProps {
  symbol: string;
  bid: string;
  ask: string;
}

export default function QuoteBar({ symbol, bid, ask }: QuoteBarProps) {
  const spread = (() => {
    if (!bid || !ask) return '';
    const b = Number(bid);
    const a = Number(ask);
    if (isNaN(b) || isNaN(a) || b <= 0) return '';
    return ((a - b) / b * 10000).toFixed(1);
  })();

  return (
    <div style={{
      display: 'flex', alignItems: 'center', gap: 16, padding: '4px 12px',
      background: 'rgba(255,255,255,0.04)', borderRadius: 6,
      border: '1px solid rgba(255,255,255,0.06)',
      fontFamily: 'monospace', fontSize: 13,
    }}>
      <span style={{ color: '#64748b', fontWeight: 600, fontSize: 11, minWidth: 60 }}>
        {symbol}
      </span>
      <span style={{ color: '#ef5350', fontWeight: 700 }}>
        {bid || '—'}
      </span>
      <span style={{ color: '#26a69a', fontWeight: 700 }}>
        {ask || '—'}
      </span>
      {spread && (
        <span style={{ color: '#64748b', fontSize: 10 }}>
          spread {spread}
        </span>
      )}
    </div>
  );
}
