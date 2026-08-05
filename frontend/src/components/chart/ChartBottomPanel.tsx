import { useState, useRef } from 'react';
import { Table, Tag, Button, Empty } from 'antd';
import { useTranslation } from 'react-i18next';
import { CloseOutlined } from '@ant-design/icons';
import type { QuickTradePosition, RecentTrade } from '@/pages/strategy/hooks/useStrategyWorkspaceState';
import { POSITIONS_KEY, HISTORY_KEY, BACKTEST_KEY as WS_BACKTEST_KEY, NO_RESULTS_KEY, NO_HISTORY_KEY, NO_OPEN_POSITIONS_KEY } from '@/gen/ant/v1/i18n/strategy_workspace_keys';
import { RETURN_LABEL_KEY as GEN_RETURN_KEY, MAX_DRAWDOWN_KEY as GEN_MAX_DRAWDOWN_KEY, SHARPE_KEY as GEN_SHARPE_KEY, WIN_RATE_KEY as GEN_WIN_RATE_KEY, TOTAL_TRADES_KEY as GEN_TOTAL_TRADES_KEY } from '@/gen/ant/v1/i18n/strategy_gen_keys';
import { TRADING_SYMBOL_KEY, TRADING_SIDE_KEY, TRADING_VOLUME_KEY, TRADING_MARK_PRICE_KEY, TRADING_PNL_KEY, TRADING_OPEN_TIME_KEY } from '@/gen/ant/v1/i18n/trading_keys';

interface Props {
  positions: QuickTradePosition[];
  recentTrades: RecentTrade[];
  onClosePosition: (ticket: number, volume?: number) => void;
  collapsed: boolean;
  onToggleCollapsed: () => void;
  backtestMetrics?: { totalReturn?: number; maxDrawdown?: number; sharpeRatio?: number; winRate?: number; totalTrades?: number } | null;
  backtestStatus?: string;
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

export default function ChartBottomPanel({ positions, recentTrades, onClosePosition, collapsed, onToggleCollapsed, backtestMetrics, panelHeight, onResizeStart, dragging }: Props) {
  const { t } = useTranslation();
  const [tab, setTab] = useState<'positions' | 'history' | 'backtest'>('positions');
  const resizeRef = useRef<HTMLDivElement>(null);

  if (collapsed) {
    return (
      <div
        onClick={onToggleCollapsed}
        style={{
          height: 28, flexShrink: 0, borderTop: '1px solid var(--ant-color-border)',
          background: 'var(--ant-color-bg-elevated)', cursor: 'pointer',
          display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 12,
          fontSize: 11, color: 'var(--ant-color-text-secondary)', userSelect: 'none',
        }}
      >
        <span>▲ {t(POSITIONS_KEY)} ({positions.length})</span>
        <span>·</span>
        <span>{t(HISTORY_KEY)} ({recentTrades.length})</span>
        <span>·</span>
        <span>{t(WS_BACKTEST_KEY)}</span>
      </div>
    );
  }

  const positionColumns = [
    {
      title: t(TRADING_SYMBOL_KEY), dataIndex: 'symbol', key: 'symbol',
      width: 100, render: (v: string) => <span style={{ fontSize: 11, fontWeight: 600 }}>{v}</span>,
    },
    {
      title: t(TRADING_SIDE_KEY), dataIndex: 'side', key: 'side', width: 60,
      render: (v: string) => <Tag color={v === 'buy' ? 'success' : 'error'} style={{ fontSize: 10 }}>{v?.toUpperCase()}</Tag>,
    },
    {
      title: t(TRADING_VOLUME_KEY), dataIndex: 'volume', key: 'volume', width: 70,
      render: (v: number) => <span style={{ fontSize: 11 }}>{fmtNum(v, 2)}</span>,
    },
    {
      title: t('trading.entryPrice'), dataIndex: 'openPrice', key: 'openPrice', width: 90,
      render: (v: number) => <span style={{ fontSize: 11 }}>{fmtNum(v, 5)}</span>,
    },
    {
      title: t(TRADING_MARK_PRICE_KEY), dataIndex: 'markPrice', key: 'markPrice', width: 90,
      render: (v: number) => <span style={{ fontSize: 11 }}>{fmtNum(v, 5)}</span>,
    },
    {
      title: t(TRADING_PNL_KEY), dataIndex: 'profit', key: 'profit', width: 80,
      render: (v: number) => <span style={{ fontSize: 11, fontWeight: 600, color: v >= 0 ? '#3fb950' : '#f85149' }}>
        {v >= 0 ? '+' : ''}{fmtNum(v, 2)}
      </span>,
    },
    {
      title: '', key: 'action', width: 50,
      render: (_: unknown, r: unknown) => (
        <Button size="small" type="text" danger icon={<CloseOutlined />}
          onClick={(e) => { e.stopPropagation(); onClosePosition(r.ticket); }} />
      ),
    },
  ];

  const historyColumns = [
    {
      title: t(TRADING_SYMBOL_KEY), dataIndex: 'symbol', key: 'symbol',
      width: 100, render: (v: string) => <span style={{ fontSize: 11, fontWeight: 600 }}>{v}</span>,
    },
    {
      title: t(TRADING_SIDE_KEY), dataIndex: 'side', key: 'side', width: 60,
      render: (v: string) => <Tag color={v === 'buy' || v === 'long' ? 'success' : 'error'} style={{ fontSize: 10 }}>{v?.toUpperCase()}</Tag>,
    },
    {
      title: t(TRADING_VOLUME_KEY), dataIndex: 'volume', key: 'volume', width: 70,
      render: (v: number) => <span style={{ fontSize: 11 }}>{fmtNum(v, 2)}</span>,
    },
    {
      title: t('trading.closePrice'), dataIndex: 'closePrice', key: 'closePrice', width: 90,
      render: (v: number) => <span style={{ fontSize: 11 }}>{fmtNum(v, 5)}</span>,
    },
    {
      title: t(TRADING_PNL_KEY), dataIndex: 'profit', key: 'profit', width: 80,
      render: (v: number) => <span style={{ fontSize: 11, fontWeight: 600, color: v >= 0 ? '#3fb950' : '#f85149' }}>
        {v >= 0 ? '+' : ''}{fmtNum(v, 2)}
      </span>,
    },
    {
      title: t('trading.closeTime'), dataIndex: 'closeTime', key: 'closeTime', width: 100,
      render: (v: string) => <span style={{ fontSize: 10, color: 'var(--ant-color-text-tertiary)' }}>{fmtTime(v)}</span>,
    },
  ];

  return (
    <div style={{
      ...(panelHeight ? { height: panelHeight } : { height: 160 }),
      flexShrink: 0, borderTop: '1px solid var(--ant-color-border)',
      background: 'var(--ant-color-bg-elevated)', display: 'flex', flexDirection: 'column',
      userSelect: dragging ? 'none' : 'auto',
    }}>
      {/* Resize handle */}
      {onResizeStart && (
        <div ref={resizeRef} onMouseDown={onResizeStart} style={{
          height: 5, cursor: 'row-resize', background: dragging ? '#58a6ff' : 'transparent', flexShrink: 0,
        }} />
      )}
      {/* Tab bar + collapse */}
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
        <div
          onClick={() => setTab('backtest')}
          style={{
            padding: '0 16px', height: '100%', display: 'flex', alignItems: 'center', gap: 6,
            cursor: 'pointer', fontSize: 12, fontWeight: 600,
            color: tab === 'backtest' ? '#58a6ff' : 'var(--ant-color-text-secondary)',
            borderBottom: tab === 'backtest' ? '2px solid #58a6ff' : 'none',
          }}
        >
          {t(WS_BACKTEST_KEY)}
        </div>
        <div style={{ flex: 1 }} />
          <div
          onClick={onToggleCollapsed}
          style={{ padding: '0 12px', cursor: 'pointer', fontSize: 11, color: 'var(--ant-color-text-tertiary)', height: '100%', display: 'flex', alignItems: 'center' }}
        >
          ▼
        </div>
      </div>

      {/* Table area */}
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
        ) : tab === 'history' ? (
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
        ) : (
          <div style={{ padding: 16 }}>
            {backtestMetrics ? (
              <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: 8 }}>
                {[
                  { label: t(GEN_RETURN_KEY), value: backtestMetrics.totalReturn != null ? `${backtestMetrics.totalReturn.toFixed(1)}%` : '—', color: (backtestMetrics.totalReturn ?? 0) >= 0 ? '#3fb950' : '#f85149' },
                  { label: t(GEN_MAX_DRAWDOWN_KEY), value: backtestMetrics.maxDrawdown != null ? `${backtestMetrics.maxDrawdown.toFixed(1)}%` : '—', color: '#f85149' },
                  { label: t(GEN_SHARPE_KEY), value: backtestMetrics.sharpeRatio != null ? backtestMetrics.sharpeRatio.toFixed(2) : '—' },
                  { label: t(GEN_WIN_RATE_KEY), value: backtestMetrics.winRate != null ? `${backtestMetrics.winRate.toFixed(1)}%` : '—' },
                  { label: t(GEN_TOTAL_TRADES_KEY), value: backtestMetrics.totalTrades != null ? String(backtestMetrics.totalTrades) : '—' },
                ].map((m, i) => (
                  <div key={i} style={{ background: 'var(--ant-color-fill-quaternary)', borderRadius: 6, padding: '8px 12px', textAlign: 'center' }}>
                    <div style={{ fontSize: 14, fontWeight: 700, color: m.color }}>{m.value}</div>
                    <div style={{ fontSize: 10, color: 'var(--ant-color-text-tertiary)' }}>{m.label}</div>
                  </div>
                ))}
              </div>
            ) : (
              <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={t(NO_RESULTS_KEY)} />
            )}
          </div>
        )}
      </div>
    </div>
  );
}
