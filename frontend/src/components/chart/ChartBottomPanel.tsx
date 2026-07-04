import { useState } from 'react';
import { Table, Tag, Button, Empty } from 'antd';
import { useTranslation } from 'react-i18next';
import { CloseOutlined } from '@ant-design/icons';
import type { QuickTradePosition, RecentTrade } from '@/pages/strategy/hooks/useStrategyWorkspaceState';

interface Props {
  positions: QuickTradePosition[];
  recentTrades: RecentTrade[];
  onClosePosition: (ticket: number, volume?: number) => void;
  collapsed: boolean;
  onToggleCollapsed: () => void;
  backtestMetrics?: { totalReturn?: number; maxDrawdown?: number; sharpeRatio?: number; winRate?: number; totalTrades?: number } | null;
  backtestStatus?: string;
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

export default function ChartBottomPanel({ positions, recentTrades, onClosePosition, collapsed, onToggleCollapsed, backtestMetrics, backtestStatus }: Props) {
  const { t } = useTranslation();
  const [tab, setTab] = useState<'positions' | 'history' | 'backtest'>('positions');

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
        <span>▲ {t('strategy.workspace.positions')} ({positions.length})</span>
        <span>·</span>
        <span>{t('strategy.workspace.history')} ({recentTrades.length})</span>
        <span>·</span>
        <span>{t('strategy.workspace.backtest')}</span>
      </div>
    );
  }

  const positionColumns = [
    {
      title: t('trading.symbol'), dataIndex: 'symbol', key: 'symbol',
      width: 100, render: (v: string) => <span style={{ fontSize: 11, fontWeight: 600 }}>{v}</span>,
    },
    {
      title: t('trading.side'), dataIndex: 'side', key: 'side', width: 60,
      render: (v: string) => <Tag color={v === 'buy' ? 'success' : 'error'} style={{ fontSize: 10 }}>{v?.toUpperCase()}</Tag>,
    },
    {
      title: t('trading.volume'), dataIndex: 'volume', key: 'volume', width: 70,
      render: (v: number) => <span style={{ fontSize: 11 }}>{fmtNum(v, 2)}</span>,
    },
    {
      title: t('trading.entryPrice'), dataIndex: 'openPrice', key: 'openPrice', width: 90,
      render: (v: number) => <span style={{ fontSize: 11 }}>{fmtNum(v, 5)}</span>,
    },
    {
      title: t('trading.markPrice'), dataIndex: 'markPrice', key: 'markPrice', width: 90,
      render: (v: number) => <span style={{ fontSize: 11 }}>{fmtNum(v, 5)}</span>,
    },
    {
      title: t('trading.pnl'), dataIndex: 'profit', key: 'profit', width: 80,
      render: (v: number) => <span style={{ fontSize: 11, fontWeight: 600, color: v >= 0 ? '#3fb950' : '#f85149' }}>
        {v >= 0 ? '+' : ''}{fmtNum(v, 2)}
      </span>,
    },
    {
      title: '', key: 'action', width: 50,
      render: (_: any, r: any) => (
        <Button size="small" type="text" danger icon={<CloseOutlined />}
          onClick={(e) => { e.stopPropagation(); onClosePosition(r.ticket); }} />
      ),
    },
  ];

  const historyColumns = [
    {
      title: t('trading.symbol'), dataIndex: 'symbol', key: 'symbol',
      width: 100, render: (v: string) => <span style={{ fontSize: 11, fontWeight: 600 }}>{v}</span>,
    },
    {
      title: t('trading.side'), dataIndex: 'side', key: 'side', width: 60,
      render: (v: string) => <Tag color={v === 'buy' || v === 'long' ? 'success' : 'error'} style={{ fontSize: 10 }}>{v?.toUpperCase()}</Tag>,
    },
    {
      title: t('trading.volume'), dataIndex: 'volume', key: 'volume', width: 70,
      render: (v: number) => <span style={{ fontSize: 11 }}>{fmtNum(v, 2)}</span>,
    },
    {
      title: t('trading.closePrice'), dataIndex: 'closePrice', key: 'closePrice', width: 90,
      render: (v: number) => <span style={{ fontSize: 11 }}>{fmtNum(v, 5)}</span>,
    },
    {
      title: t('trading.pnl'), dataIndex: 'profit', key: 'profit', width: 80,
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
      height: 200, flexShrink: 0, borderTop: '1px solid var(--ant-color-border)',
      background: 'var(--ant-color-bg-elevated)', display: 'flex', flexDirection: 'column',
    }}>
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
          {t('strategy.workspace.positions')}
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
          {t('strategy.workspace.history')}
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
          {t('strategy.workspace.backtest')}
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
            <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={t('strategy.workspace.noOpenPositions')}
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
            <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={t('strategy.workspace.noHistory')}
              style={{ margin: '20px 0' }} />
          )
        ) : (
          <div style={{ padding: 16 }}>
            {backtestMetrics ? (
              <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: 8 }}>
                {[
                  { label: t('strategy.gen.return'), value: backtestMetrics.totalReturn != null ? `${backtestMetrics.totalReturn.toFixed(1)}%` : '—', color: (backtestMetrics.totalReturn ?? 0) >= 0 ? '#3fb950' : '#f85149' },
                  { label: t('strategy.gen.maxDrawdown'), value: backtestMetrics.maxDrawdown != null ? `${backtestMetrics.maxDrawdown.toFixed(1)}%` : '—', color: '#f85149' },
                  { label: t('strategy.gen.sharpe'), value: backtestMetrics.sharpeRatio != null ? backtestMetrics.sharpeRatio.toFixed(2) : '—' },
                  { label: t('strategy.gen.winRate'), value: backtestMetrics.winRate != null ? `${backtestMetrics.winRate.toFixed(1)}%` : '—' },
                  { label: t('strategy.gen.totalTrades'), value: backtestMetrics.totalTrades != null ? String(backtestMetrics.totalTrades) : '—' },
                ].map((m, i) => (
                  <div key={i} style={{ background: 'var(--ant-color-fill-quaternary)', borderRadius: 6, padding: '8px 12px', textAlign: 'center' }}>
                    <div style={{ fontSize: 14, fontWeight: 700, color: m.color }}>{m.value}</div>
                    <div style={{ fontSize: 10, color: 'var(--ant-color-text-tertiary)' }}>{m.label}</div>
                  </div>
                ))}
              </div>
            ) : (
              <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={t('strategy.workspace.noResults')} />
            )}
          </div>
        )}
      </div>
    </div>
  );
}
