import { Table, Tag, Typography, Button, Popconfirm } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { PauseCircleOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import type { ActiveStrategy } from '@/gen/ant/v1/strategy_runtime_pb';
import { shortId, MODE_COLORS } from '../../LiveStrategyPageSignalDrawer';
import { formatMode } from './formatMode';

const { Text } = Typography;

interface Props {
  orphanRuns: ActiveStrategy[];
  onStop: (runId: string) => void;
  stopping: string | null;
}

export default function OrphanRunsTable({ orphanRuns, onStop, stopping }: Props) {
  const { t } = useTranslation();

  if (orphanRuns.length === 0) return null;

  const columns: ColumnsType<ActiveStrategy> = [
    { title: t('strategy.live.runId', { defaultValue: 'Run ID' }), dataIndex: 'runId', width: 100, render: (v: string) => <Text code>{shortId(v)}</Text> },
    { title: t('strategy.live.strategyName', { defaultValue: 'Strategy' }), dataIndex: 'strategyName', width: 120, render: (v: string, r: ActiveStrategy) => v || <Text type="secondary">{shortId(r.runId)}</Text> },
    { title: t('strategy.live.symbol', { defaultValue: 'Symbol' }), dataIndex: 'symbol', width: 80 },
    { title: t('strategy.live.mode', { defaultValue: 'Mode' }), dataIndex: 'mode', width: 60, render: (v: string) => <Tag color={MODE_COLORS[v] || 'default'}>{formatMode(v, t)}</Tag> },
    { title: t('strategy.live.pnl', { defaultValue: 'PnL' }), dataIndex: 'pnl', width: 80, render: (v: string) => v ? <Text type={Number(v) >= 0 ? 'success' : 'danger'}>{Number(v) >= 0 ? `+${v}` : v}</Text> : <Text type="secondary">-</Text> },
    {
      title: '', width: 80,
      render: (_: unknown, r: ActiveStrategy) => (
        <Popconfirm title={t('strategy.live.confirmStop', { defaultValue: 'Stop this strategy?' })} onConfirm={() => onStop(r.runId)}>
          <Button size="small" danger loading={stopping === r.runId} icon={<PauseCircleOutlined />} />
        </Popconfirm>
      ),
    },
  ];

  return (
    <div style={{ marginTop: 16 }}>
      <Text type="secondary" style={{ fontSize: 13, marginBottom: 8, display: 'block' }}>
        {t('strategy.live.temporaryRuns', { defaultValue: 'Temporary Runs' })}
      </Text>
      <Table<ActiveStrategy> size="small" dataSource={orphanRuns} rowKey="runId" columns={columns} pagination={false} />
    </div>
  );
}
