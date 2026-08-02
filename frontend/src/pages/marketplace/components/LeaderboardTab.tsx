import { useState } from 'react';
import { Card, Table, Tag, Typography, Segmented, Space, Tooltip } from 'antd';
import { FireOutlined, RocketOutlined, RiseOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { marketplaceClient } from '@/client/connect';
import { useRpcQuery } from '@/hooks/useRpcQuery';
import type { LeaderboardEntry } from '@/gen/ant/v1/marketplace_service_pb';

const { Text } = Typography;

type LBType = 'return' | 'popular' | 'new';
type LBPeriod = 'week' | 'month' | 'quarter' | 'all';

function rankColor(rank: number): string {
  if (rank === 1) return '#faad14';
  if (rank === 2) return '#d9d9d9';
  if (rank === 3) return '#d48806';
  return '#bfbfbf';
}

export default function LeaderboardTab() {
  const { t } = useTranslation();
  const [lbType, setLbType] = useState<LBType>('popular');
  const [period, setPeriod] = useState<LBPeriod>('all');

  const { data: entries } = useRpcQuery(
    ['marketplace', 'leaderboard', lbType, period],
    async () => {
      const resp = await marketplaceClient.listLeaderboard({
        type: lbType,
        period,
        limit: 50,
      });
      return (resp.entries || []) as LeaderboardEntry[];
    },
  );

  const showPeriod = lbType === 'return';

  const typeOptions = [
    { label: <span><FireOutlined /> {t('marketplace.leaderboard.popular')}</span>, value: 'popular' },
    { label: <span><RiseOutlined /> {t('marketplace.leaderboard.return')}</span>, value: 'return' },
    { label: <span><RocketOutlined /> {t('marketplace.leaderboard.new')}</span>, value: 'new' },
  ];

  const periodOptions = [
    { label: t('marketplace.leaderboard.week'), value: 'week' },
    { label: t('marketplace.leaderboard.month'), value: 'month' },
    { label: t('marketplace.leaderboard.quarter'), value: 'quarter' },
    { label: t('marketplace.leaderboard.all'), value: 'all' },
  ];

  const columns = [
    {
      title: '#', key: 'rank', width: 60,
      render: (_: unknown, row: LeaderboardEntry) => (
        <span style={{ fontWeight: 700, fontSize: 16, color: rankColor(row.rank) }}>#{row.rank}</span>
      ),
    },
    {
      title: t('marketplace.leaderboard.strategy'), key: 'title',
      render: (_: unknown, row: LeaderboardEntry) => (
        <Space direction="vertical" size={0}>
          <Text strong>{row.title || '-'}</Text>
          <Text type="secondary" style={{ fontSize: 12 }}>{row.publisherName || '-'}</Text>
        </Space>
      ),
    },
    {
      title: t('marketplace.detail.price'), key: 'price', width: 100,
      render: (_: unknown, row: LeaderboardEntry) => (
        <Tag color={row.priceModel === 'free' ? 'green' : 'gold'}>
          {row.priceModel === 'free' ? t('marketplace.card.free') : `$${row.priceAmount || '0'}`}
        </Tag>
      ),
    },
    {
      title: t('marketplace.author.subscribers'), dataIndex: 'totalSubscribers', key: 'subs', width: 100,
      sorter: (a: LeaderboardEntry, b: LeaderboardEntry) => a.totalSubscribers - b.totalSubscribers,
    },
    {
      title: t('marketplace.author.avgRating'), key: 'rating', width: 80,
      render: (_: unknown, row: LeaderboardEntry) => (
        <Tooltip title={`${row.ratingCount || 0} ratings`}>
          <span>{Number(row.avgRating || 0).toFixed(1)}</span>
        </Tooltip>
      ),
    },
  ];

  // Add return-specific columns when type=return
  if (lbType === 'return') {
    columns.splice(4, 0,
      {
        title: t('marketplace.leaderboard.totalReturn'), key: 'return', width: 110,
        render: (_: unknown, row: LeaderboardEntry) => {
          const val = Number(row.totalReturn || 0);
          return <Text type={val >= 0 ? 'success' : 'danger'} strong>{val >= 0 ? '+' : ''}{val.toFixed(2)}%</Text>;
        },
      },
      {
        title: t('marketplace.leaderboard.maxDD'), key: 'dd', width: 90,
        render: (_: unknown, row: LeaderboardEntry) => `${Number(row.maxDrawdown || 0).toFixed(2)}%`,
      },
      {
        title: t('marketplace.leaderboard.sharpe'), key: 'sharpe', width: 80,
        render: (_: unknown, row: LeaderboardEntry) => row.sharpeRatio ? Number(row.sharpeRatio).toFixed(2) : '-',
      },
    );
  }

  return (
    <div>
      <Card size="small" style={{ marginBottom: 16 }}>
        <Space style={{ width: '100%', justifyContent: 'space-between' }}>
          <Segmented options={typeOptions} value={lbType} onChange={v => setLbType(v as LBType)} />
          {showPeriod && <Segmented options={periodOptions} value={period} onChange={v => setPeriod(v as LBPeriod)} size="small" />}
        </Space>
      </Card>

      <Table<LeaderboardEntry>
        rowKey="strategyId"
        dataSource={entries || []}
        pagination={false}
        size="small"
        columns={columns}
        locale={{ emptyText: t('marketplace.leaderboard.empty') }}
      />
    </div>
  );
}
