import { Tag, Button } from 'antd';
import { useTranslation } from 'react-i18next'
import { TRADING_CLOSE_POSITION_KEY, TRADING_POSITION_ENTRY_PRICE_KEY, TRADING_POSITION_LEVERAGE_KEY, TRADING_POSITION_LONG_KEY, TRADING_POSITION_MARK_PRICE_KEY, TRADING_POSITION_SHORT_KEY, TRADING_POSITION_SIDE_KEY, TRADING_POSITION_SIZE_KEY, TRADING_POSITION_UNREALIZED_PN_L_KEY } from '@/gen/ant/v1/i18n/trading_keys';

;

export interface PositionItem {
  ticket: number; side: string; volume: number;
  openPrice: number; markPrice?: number; profit: number; leverage?: number;
}

interface Props {
  symbol: string;
  positions: PositionItem[];
  closingTicket: number | null;
  onClosePosition: (ticket: number, volume?: number) => void;
}

function num(v: number | undefined | null): string {
  if (v == null || v === undefined) return '—';
  if (Math.abs(v) >= 100) return v.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 });
  if (Math.abs(v) >= 1) return v.toLocaleString('en-US', { minimumFractionDigits: 4, maximumFractionDigits: 4 });
  return v.toLocaleString('en-US', { minimumFractionDigits: 5, maximumFractionDigits: 5 });
}

function pnl(v: number): string {
  const sign = v >= 0 ? '+' : '';
  return sign + '$' + num(Math.abs(v));
}

export default function PositionSection({ symbol, positions, closingTicket, onClosePosition }: Props) {
  const { t } = useTranslation();
  if (positions.length === 0) {
    return (
      <div style={{ borderTop: '1px solid #e8e8e8', paddingTop: 10 }}>
        <div style={{ fontSize: 11, fontWeight: 700, color: '#595959', marginBottom: 6 }}>{t('common.currentPosition')}</div>
        <div style={{ textAlign: 'center', padding: 10, color: '#8c8c8c', fontSize: 11 }}>
          {t('common.noOpenPositionsForSymbol', { symbol: symbol || 'this symbol' })}
        </div>
      </div>
    );
  }

  return (
    <div style={{ borderTop: '1px solid #e8e8e8', paddingTop: 10, display: 'flex', flexDirection: 'column', gap: 6 }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
        <span style={{ fontSize: 11, fontWeight: 700, color: '#595959' }}>{t('common.currentPosition')}</span>
        {positions.length > 1 && <span style={{ fontSize: 10, color: '#8c8c8c' }}>({positions.length})</span>}
      </div>

      {positions.map((pos, idx) => {
        const isLong = pos.side === 'buy' || pos.side === 'long';
        return (
          <div key={`pos-${idx}-${pos.ticket}`} style={{
            background: '#fafbfc', border: '1px solid #e8e8e8', borderRadius: 6, padding: '8px 10px',
            borderLeft: `3px solid ${isLong ? '#26a69a' : '#ef5350'}`,
            display: 'flex', flexDirection: 'column', gap: 3,
          }}>
            <Row label={t(TRADING_POSITION_SIDE_KEY)}>
              <Tag color={isLong ? 'success' : 'error'} style={{ fontSize: 10, lineHeight: '16px' }}>
                {isLong ? t(TRADING_POSITION_LONG_KEY) : t(TRADING_POSITION_SHORT_KEY)}
              </Tag>
            </Row>
            <Row label={t(TRADING_POSITION_SIZE_KEY)}><span>{pos.volume}</span></Row>
            <Row label={t(TRADING_POSITION_ENTRY_PRICE_KEY)}><span>${num(pos.openPrice)}</span></Row>
            {pos.markPrice && <Row label={t(TRADING_POSITION_MARK_PRICE_KEY)}><span>${num(pos.markPrice)}</span></Row>}
            {pos.leverage && pos.leverage > 1 && <Row label={t(TRADING_POSITION_LEVERAGE_KEY)}><span>{pos.leverage}x</span></Row>}
            <Row label={t(TRADING_POSITION_UNREALIZED_PN_L_KEY)}>
              <span style={{ color: pos.profit >= 0 ? '#26a69a' : '#ef5350', fontWeight: 700 }}>
                {pnl(pos.profit)}
              </span>
            </Row>
            <Button danger size="small" block ghost loading={closingTicket === pos.ticket}
              style={{ marginTop: 4 }} onClick={() => onClosePosition(pos.ticket, pos.volume)}>
              {t(TRADING_CLOSE_POSITION_KEY)}
            </Button>
          </div>
        );
      })}
    </div>
  );
}

function Row({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', fontSize: 10 }}>
      <span style={{ color: '#8c8c8c' }}>{label}</span>
      <span style={{ color: '#262626', fontWeight: 600 }}>{children}</span>
    </div>
  );
}
