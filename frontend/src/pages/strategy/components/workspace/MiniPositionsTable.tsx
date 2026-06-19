import { useState } from 'react';
import { Tag, Button } from 'antd';
import { CaretUpOutlined, CaretDownOutlined, RiseOutlined, FallOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next'
import { CLOSE_POSITION_KEY, MARK_PRICE_KEY, PNL_KEY, PRICE_KEY, SIDE_KEY, SYMBOL_KEY, VOLUME_KEY } from '@/gen/ant/v1/i18n/trading_keys';

;
import type { QuickTradePosition } from '../../hooks/useStrategyWorkspaceState';

interface Props {
  positions: QuickTradePosition[];
  onClosePosition?: (ticket: number, volume?: number) => void;
}

function fmtNum(v: number, d = 2) { return v.toFixed(d); }
function pnlColor(v: number) { return v >= 0 ? '#26a69a' : '#ef5350'; }

export default function MiniPositionsTable({ positions, onClosePosition }: Props) {
  const { t } = useTranslation();
  const [expanded, setExpanded] = useState(true);
  if (positions.length === 0) return null;

  return (
    <div style={{ borderBottom: '1px solid #e8e8e8', background: '#fafbfc', flexShrink: 0 }}>
      <div onClick={() => setExpanded(!expanded)} role="button" tabIndex={0}
        onKeyUp={e => e.key === 'Enter' && setExpanded(!expanded)}
        style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between',
          padding: '6px 14px', cursor: 'pointer', userSelect: 'none',
          background: 'linear-gradient(180deg, #f8fafc 0%, #f1f5f9 100%)' }}>
        <span style={{ fontSize: 11, fontWeight: 700, color: '#262626' }}>
          {t(OPEN_POSITIONS_KEY, { count: positions.length })}
        </span>
        <span style={{ fontSize: 10, color: '#8c8c8c' }}>{expanded ? <CaretUpOutlined /> : <CaretDownOutlined />}</span>
      </div>
      {expanded && (
        <div style={{ overflowX: 'auto', padding: '0 14px 8px' }}>
          <table style={{ width: '100%', fontSize: 11, borderCollapse: 'collapse' }}>
            <thead>
              <tr style={{ color: '#8c8c8c', borderBottom: '1px solid #e8e8e8' }}>
                <th style={th}>{t(SYMBOL_KEY)}</th><th style={th}>{t(SIDE_KEY)}</th><th style={thR}>{t(VOLUME_KEY)}</th>
                <th style={thR}>{t(PRICE_KEY)}</th><th style={thR}>{t(MARK_PRICE_KEY)}</th>
                <th style={thR}>{t(PNL_KEY)}</th><th style={th}></th>
              </tr>
            </thead>
            <tbody>
              {positions.map(p => (
                <tr key={p.ticket} style={{ borderBottom: '1px solid #f0f0f0' }}>
                  <td style={td}>{p.symbol}</td>
                  <td style={td}>
                    <Tag color={p.side === 'long' ? 'green' : 'red'} style={{ fontSize: 9, margin: 0, lineHeight: '16px', padding: '0 4px' }}>
                      {p.side === 'long' ? <RiseOutlined /> : <FallOutlined />} {p.side.toUpperCase()}
                    </Tag>
                  </td>
                  <td style={tdR}>{p.volume.toFixed(2)}</td>
                  <td style={tdR}>{fmtNum(p.openPrice)}</td>
                  <td style={tdR}>{p.markPrice ? fmtNum(p.markPrice) : '—'}</td>
                  <td style={{ ...tdR, color: pnlColor(p.profit) }}>
                    {p.profit >= 0 ? '+' : ''}{fmtNum(p.profit)}
                  </td>
                  <td style={td}>
                    <Button size="small" danger type="link"
                      onClick={e => { e.stopPropagation(); onClosePosition?.(p.ticket, p.volume); }}
                      style={{ fontSize: 10, padding: 0, height: 20 }}>{t(CLOSE_POSITION_KEY)}</Button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}

const th: React.CSSProperties = { textAlign: 'left', padding: '4px 6px', fontWeight: 600, fontSize: 10 };
const thR: React.CSSProperties = { ...th, textAlign: 'right' };
const td: React.CSSProperties = { padding: '3px 6px' };
const tdR: React.CSSProperties = { ...td, textAlign: 'right', fontFamily: 'monospace' };
