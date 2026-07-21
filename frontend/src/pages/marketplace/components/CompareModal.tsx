import { useState, useCallback, useMemo } from 'react';
import { Modal, Table, Typography, Space, Empty, Button, Tag } from 'antd';
import { SwapOutlined, CloseOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { marketplaceClient } from '@/client/connect';
import { useRpcQuery } from '@/hooks/useRpcQuery';
import type { StrategyComparison } from '@/gen/ant/v1/marketplace_service_pb';

const { Text } = Typography;

interface Props {
  open: boolean;
  strategyIds: string[];
  onClose: () => void;
  onRemove: (id: string) => void;
}

type CompareDirection = 'higher' | 'lower';

interface MetricDef {
  key: string;
  label: string;
  getValue: (s: StrategyComparison) => string;
  direction?: CompareDirection;
}

export default function CompareModal({ open, strategyIds, onClose, onRemove }: Props) {
  const { t } = useTranslation();

  const { data: strategies } = useRpcQuery(
    ['marketplace', 'compare', strategyIds.join(',')],
    async () => {
      if (strategyIds.length === 0) return [] as StrategyComparison[];
      const resp = await marketplaceClient.compareStrategies({ strategyIds });
      return (resp.strategies || []) as StrategyComparison[];
    },
    { enabled: strategyIds.length > 0 },
  );

  const rows = strategies || [];

  const metrics: MetricDef[] = useMemo(() => [
    { key: 'price', label: t('marketplace.detail.price'), getValue: s => s.priceModel === 'free' ? '0' : (s.priceAmount || '0'), direction: 'lower' },
    { key: 'subs', label: t('marketplace.author.subscribers'), getValue: s => String(s.totalSubscribers || 0), direction: 'higher' },
    { key: 'rating', label: t('marketplace.author.avgRating'), getValue: s => Number(s.avgRating || 0).toFixed(1), direction: 'higher' },
    { key: 'btReturn', label: t('marketplace.backtest.totalReturn', { defaultValue: 'Total Return' }), getValue: s => s.backtestTotalReturn || '-', direction: 'higher' },
    { key: 'btDD', label: t('marketplace.backtest.maxDrawdown', { defaultValue: 'Max Drawdown' }), getValue: s => s.backtestMaxDrawdown || '-', direction: 'lower' },
    { key: 'btSharpe', label: t('marketplace.backtest.sharpe', { defaultValue: 'Sharpe' }), getValue: s => s.backtestSharpeRatio || '-', direction: 'higher' },
    { key: 'btWinRate', label: t('marketplace.backtest.winRate', { defaultValue: 'Win Rate' }), getValue: s => s.backtestWinRate || '-', direction: 'higher' },
    { key: 'btTrades', label: t('marketplace.backtest.totalTrades', { defaultValue: 'Total Trades' }), getValue: s => String(s.backtestTotalTrades || 0), direction: 'higher' },
    { key: 'liveReturn', label: t('marketplace.leaderboard.totalReturn', { defaultValue: 'Live Return' }), getValue: s => s.liveTotalReturn || '-', direction: 'higher' },
    { key: 'liveDD', label: t('marketplace.leaderboard.maxDD', { defaultValue: 'Live Max DD' }), getValue: s => s.liveMaxDrawdown || '-', direction: 'lower' },
    { key: 'liveSharpe', label: t('marketplace.leaderboard.sharpe', { defaultValue: 'Live Sharpe' }), getValue: s => s.liveSharpeRatio || '-', direction: 'higher' },
  ], [t]);

  // Compute best strategy ID per metric for highlighting.
  const bestByMetric = useMemo(() => {
    const result: Record<string, string> = {};
    if (rows.length < 2) return result;

    for (const m of metrics) {
      if (!m.direction) continue;
      const parsed = rows
        .map(s => ({ id: s.strategyId, val: parseFloat(m.getValue(s)) }))
        .filter(x => !isNaN(x.val));
      if (parsed.length === 0) continue;

      const best = parsed.reduce((acc, x) => {
        if (m.direction === 'higher') return x.val > acc.val ? x : acc;
        return x.val < acc.val ? x : acc;
      });
      result[m.key] = best.id;
    }
    return result;
  }, [rows, metrics]);

  const columns = [
    {
      title: t('marketplace.compare.metric', { defaultValue: 'Metric' }),
      dataIndex: 'metric', key: 'metric', width: 140, fixed: 'left' as const,
      render: (v: string) => <Text strong>{v}</Text>,
    },
    ...rows.map(s => ({
      title: (
        <Space>
          <Text strong style={{ fontSize: 13 }}>{s.title || '-'}</Text>
          <Button type="text" size="small" icon={<CloseOutlined />} onClick={() => onRemove(s.strategyId)} />
        </Space>
      ),
      dataIndex: s.strategyId, key: s.strategyId,
      render: (v: string, record: Record<string, string>) => {
        const isBest = bestByMetric[record.key] === s.strategyId;
        if (isBest) {
          return <Tag color="green" style={{ fontWeight: 600 }}>{v || '-'}</Tag>;
        }
        return <Text>{v || '-'}</Text>;
      },
    })),
  ];

  const dataSource = metrics.map(m => {
    const row: Record<string, string> = { metric: m.label, key: m.key };
    rows.forEach(s => { row[s.strategyId] = m.getValue(s); });
    return row;
  });

  return (
    <Modal
      title={<span><SwapOutlined /> {t('marketplace.compare.title', { defaultValue: 'Compare Strategies' })}</span>}
      open={open}
      onCancel={onClose}
      footer={null}
      width={Math.max(600, 160 + rows.length * 200)}
      destroyOnClose
    >
      {rows.length === 0 ? (
        <Empty description={t('marketplace.compare.empty', { defaultValue: 'Select strategies to compare' })} />
      ) : (
        <Table
          dataSource={dataSource}
          columns={columns}
          pagination={false}
          size="small"
          rowKey="key"
          scroll={{ x: 'max-content' }}
        />
      )}
    </Modal>
  );
}

// Hook for managing compare selection state.
export function useCompareSelection() {
  const [compareIds, setCompareIds] = useState<string[]>([]);
  const [compareOpen, setCompareOpen] = useState(false);

  const toggleCompare = useCallback((id: string) => {
    setCompareIds(prev => {
      if (prev.includes(id)) return prev.filter(x => x !== id);
      if (prev.length >= 4) return prev;
      return [...prev, id];
    });
  }, []);

  const removeFromCompare = useCallback((id: string) => {
    setCompareIds(prev => prev.filter(x => x !== id));
  }, []);

  const clearCompare = useCallback(() => {
    setCompareIds([]);
    setCompareOpen(false);
  }, []);

  return {
    compareIds,
    compareOpen,
    setCompareOpen,
    toggleCompare,
    removeFromCompare,
    clearCompare,
  };
}
