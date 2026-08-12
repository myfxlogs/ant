import { useState, useRef } from 'react';
import { Table, Tag, Button, Empty } from 'antd';
import { useTranslation } from 'react-i18next';
import { CloseOutlined } from '@ant-design/icons';
import type { QuickTradePosition, RecentTrade } from '@/pages/strategy/hooks/useStrategyWorkspaceState';
import { POSITIONS_KEY, HISTORY_KEY, NO_HISTORY_KEY, NO_OPEN_POSITIONS_KEY } from '@/gen/ant/v1/i18n/strategy_workspace_keys';
import { TRADING_SYMBOL_KEY, TRADING_SIDE_KEY, TRADING_VOLUME_KEY, TRADING_MARK_PRICE_KEY, TRADING_PNL_KEY } from '@/gen/ant/v1/i18n/trading_keys';
import { TRADE_TIME_KEY } from '@/gen/ant/v1/i18n/strategy_backtest_keys';

interface Props {
  positions: QuickTradePosition[];
  recentTrades: RecentTrade[];
  onClosePosition: (ticket: number, volume?: number) => void;
  collapsed: boolean;
  onToggleCollapsed: () => void;
  panelHeight?: number;
  onResizeStart?: (e: React.MouseEvent) => void;
  dragging?: boolean;
}

function fmtNum(v: number | undefined, d = 2): string {
  if (v == null) return '—';
  return v.toLocaleString('en-US', { minimumFractionDigits: d, maximumFractionDigits: d });
}

function fmtTime(ts?: string): string {
  if (!ts) return '—';
  const d = new Date(ts);
  const mm = String(d.getMonth() + 1).padStart(2, '0');
  const dd = String(d.getDate()).padStart(2, '0');
  const hh = String(d.getHours()).padStart(2, '0');
  const min = String(d.getMinutes()).padStart(2, '0');
  return `${mm}-${dd} ${hh}:${min}`;
}

export default function ChartBottomPanel({ positions, recentTrades, onClosePosition, collapsed, onToggleCollapsed, panelHeight, onResizeStart, dragging }: Props) {
  const { t } = useTranslation();
  const [tab, setTab] = useState<'positions' | 'history'>('positions');
  const resizeRef = useRef<HTMLDivElement>(null);

  if (collapsed) {
    return (
      <div
        onClick={onToggleCollapsed}
        style={{
          height: 28, flexShrink: 0, borderTop: '1px solid var(--ant-color-border)',
          background: 'var(--ant-color-bg-elevated)', cursor: 'pointer',
          display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 12,
        }}
      >
        <span style={{ fontSize: 11, color: 'var(--ant-color-text-tertiary)' }}>
          {t(POSITIONS_KEY)}{positions.length > 0 ? ` (${positions.length})` : ''} · {t(HISTORY_KEY)}{recentTrades.length > 0 ? ` (${recentTrades.length})` : ''}
        </span>
        <span style={{ fontSize: 10 }}>▲</span>
      </div>
    );
  }

  const positionColumns = [
    { title: t(TRADING_SYMBOL_KEY), dataIndex: 'symbol', key: 'symbol', width: 80 },
    { title: t(TRADING_SIDE_KEY), dataIndex: 'side', key: 'side', width: 50, render: (v: string) => <Tag color={v === 'BUY' ? 'green' : 'red'}>{v}</Tag> },
    { title: t(TRADING_VOLUME_KEY), dataIndex: 'volume', key: 'volume', width: 60 },
    { title: t(TRADING_MARK_PRICE_KEY), dataIndex: 'openPrice', key: 'openPrice', width: 80 },
    { title: t(TRADING_PNL_KEY), dataIndex: 'profit', key: 'profit', width: 80, render: (v: number | undefined) => <span style={{ color: (v ?? 0) >= 0 ? '#3fb950' : '#f85149' }}>{fmtNum(v)}</span> },
    { title: '', key: 'action', width: 40, render: (_: unknown, r: { ticket: number }) => (
      <Button size="small" type="text" danger icon={<CloseOutlined />} onClick={() => onClosePosition(r.ticket)} />
    )},
  ];

  const historyColumns = [
    { title: t(TRADING_SYMBOL_KEY), dataIndex: 'symbol', key: 'symbol', width: 80 },
    { title: t(TRADING_SIDE_KEY), dataIndex: 'side', key: 'side', width: 50, render: (v: string) => <Tag color={v === 'BUY' ? 'green' : 'red'}>{v}</Tag> },
    { title: t(TRADING_VOLUME_KEY), dataIndex: 'volume', key: 'volume', width: 60 },
    { title: t(TRADING_MARK_PRICE_KEY), dataIndex: 'price', key: 'price', width: 80 },
    { title: t(TRADING_PNL_KEY), dataIndex: 'pnl', key: 'pnl', width: 80, render: (v: number | undefined) => <span style={{ color: (v ?? 0) >= 0 ? '#3fb950' : '#f85149' }}>{fmtNum(v)}</span> },
    { title: t(TRADE_TIME_KEY), dataIndex: 'time', key: 'time', width: 100, render: (v: string) => fmtTime(v) },
  ];

  return (
    <div style={{
      ...(panelHeight ? { height: panelHeight } : { height: 160 }),
      flexShrink: 0, borderTop: '1px solid var(--ant-color-border)',
      background: 'var(--ant-color-bg-elevated)', display: 'flex', flexDirection: 'column',
      userSelect: dragging ? 'none' : 'auto',
    }}>
      {onResizeStart && (
        <div ref={resizeRef} onMouseDown={onResizeStart} style={{
          height: 5, cursor: 'row-resize', background: dragging ? '#58a6ff' : 'transparent', flexShrink: 0,
        }} />
      )}
      <div style={{
        display: 'flex', alignItems: 'center', gap: 0, flexShrink: 0,
        borderBottom: '1px solid var(--ant-color-border)', height: 32,
      }}>
        <div
          onClick={() => setTab('positions')}
          style={{
            padding: '0 16px', height: '100%', display: 'flex', alignItems: 'center', gap: 6,
            cursor: 'pointer', fontSize: 12, fontWeight: 600,
            color: tab === 'positions' ? '#58a6ff' : 'var(--ant-color-text-secondary)',
            borderBottom: tab === 'positions' ? '2px solid #58a6ff' : 'none',
          }}
        >
          {t(POSITIONS_KEY)}
          {positions.length > 0 && <span style={{ fontSize: 10, color: 'var(--ant-color-text-tertiary)' }}>({positions.length})</span>}
        </div>
        <div
          onClick={() => setTab('history')}
          style={{
            padding: '0 16px', height: '100%', display: 'flex', alignItems: 'center', gap: 6,
            cursor: 'pointer', fontSize: 12, fontWeight: 600,
            color: tab === 'history' ? '#58a6ff' : 'var(--ant-color-text-secondary)',
            borderBottom: tab === 'history' ? '2px solid #58a6ff' : 'none',
          }}
        >
          {t(HISTORY_KEY)}
          {recentTrades.length > 0 && <span style={{ fontSize: 10, color: 'var(--ant-color-text-tertiary)' }}>({recentTrades.length})</span>}
        </div>
        <div style={{ flex: 1 }} />
          <div
          onClick={onToggleCollapsed}
          style={{ padding: '0 12px', cursor: 'pointer', fontSize: 11, color: 'var(--ant-color-text-tertiary)', height: '100%', display: 'flex', alignItems: 'center' }}
        >
          ▼
        </div>
      </div>

      <div style={{ flex: 1, overflow: 'auto' }}>
        {tab === 'positions' ? (
          positions.length > 0 ? (
            <Table
              dataSource={positions}
              columns={positionColumns}
              rowKey="ticket"
              size="small"
              pagination={false}
              style={{ fontSize: 11 }}
            />
          ) : (
            <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={t(NO_OPEN_POSITIONS_KEY)}
              style={{ margin: '20px 0' }} />
          )
        ) : (
          recentTrades.length > 0 ? (
            <Table
              dataSource={recentTrades}
              columns={historyColumns}
              rowKey="ticket"
              size="small"
              pagination={{ pageSize: 20, size: 'small' }}
              style={{ fontSize: 11 }}
            />
          ) : (
            <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={t(NO_HISTORY_KEY)}
              style={{ margin: '20px 0' }} />
          )
        )}
      </div>
    </div>
  );
}
