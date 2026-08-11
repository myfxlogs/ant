import { Table, Tag, Typography, Space, Empty, Descriptions, Drawer } from 'antd';
import { MonitorOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { timestampDate } from '@bufbuild/protobuf/wkt';
import type { Timestamp } from '@bufbuild/protobuf/wkt';
import type { ActiveStrategy, StrategySignalEvent } from '@/gen/ant/v1/strategy_runtime_pb';

const { Text } = Typography;

const STATUS_COLORS: Record<string, string> = {
  running: 'green',
  stopped: 'default',
  error: 'red',
};

const MODE_COLORS: Record<string, string> = {
  live: 'red',
  paper: 'blue',
};

export function formatTime(ts?: { seconds?: bigint; nanos?: number } | null): string {
  if (!ts || !ts.seconds) return '-';
  const d = timestampDate(ts as Timestamp);
  return d.toLocaleString();
}

export function shortId(id?: string): string {
  if (!id) return '-';
  return id.slice(0, 8);
}

interface SignalDrawerProps {
  open: boolean;
  onClose: () => void;
  watchingRunId: string | null;
  signals: StrategySignalEvent[];
  activeStrategies: ActiveStrategy[];
}

export function SignalDrawer({ open, onClose, watchingRunId, signals, activeStrategies }: SignalDrawerProps) {
  const { t } = useTranslation();

  const signalColumns = [
    {
      title: t('strategy.live.time', { defaultValue: 'Time' }), dataIndex: 'timestamp', width: 160,
      render: (v: { seconds?: bigint; nanos?: number } | null) => <Text style={{ fontSize: 12 }}>{formatTime(v)}</Text>,
    },
    {
      title: t('strategy.live.signalType', { defaultValue: 'Type' }), dataIndex: 'signalType', width: 100,
      render: (v: string) => {
        const color = v === 'buy' || v === 'buy_limit' || v === 'buy_stop' ? 'green'
          : v === 'sell' || v === 'sell_limit' || v === 'sell_stop' ? 'red'
          : v === 'close' || v === 'close_all' ? 'orange'
          : 'default';
        return <Tag color={color}>{v}</Tag>;
      },
    },
    {
      title: t('strategy.live.volume', { defaultValue: 'Volume' }), dataIndex: 'volume', width: 80,
    },
    {
      title: t('strategy.live.price', { defaultValue: 'Price' }), dataIndex: 'price', width: 90,
    },
    {
      title: t('strategy.live.sl', { defaultValue: 'SL' }), dataIndex: 'stopLoss', width: 90,
      render: (v: string) => v || '-',
    },
    {
      title: t('strategy.live.tp', { defaultValue: 'TP' }), dataIndex: 'takeProfit', width: 90,
      render: (v: string) => v || '-',
    },
    {
      title: t('strategy.live.reason', { defaultValue: 'Reason' }), dataIndex: 'reason', ellipsis: true,
      render: (v: string) => v ? <Text style={{ fontSize: 12 }}>{v}</Text> : <Text type="secondary">-</Text>,
    },
  ];

  const watching = activeStrategies.find(s => s.runId === watchingRunId);

  return (
    <Drawer
      title={
        <Space>
          <MonitorOutlined />
          <span>{t('strategy.live.signalLog', { defaultValue: 'Signal Log' })}</span>
          {watchingRunId && <Text code style={{ fontSize: 12 }}>{shortId(watchingRunId)}</Text>}
        </Space>
      }
      open={open}
      onClose={onClose}
      width={800}
    >
      {watchingRunId && watching && (
        <Descriptions size="small" column={4} style={{ marginBottom: 12 }}>
          <Descriptions.Item label={t('strategy.live.symbol', { defaultValue: 'Symbol' })}>
            {watching.symbol}
          </Descriptions.Item>
          <Descriptions.Item label={t('strategy.live.mode', { defaultValue: 'Mode' })}>
            <Tag color={MODE_COLORS[watching.mode || ''] || 'default'}>
              {watching.mode}
            </Tag>
          </Descriptions.Item>
          <Descriptions.Item label={t('strategy.live.signals', { defaultValue: 'Signals' })}>
            {watching.signalCount}
          </Descriptions.Item>
          <Descriptions.Item label={t('strategy.live.errors', { defaultValue: 'Errors' })}>
            {watching.errorCount}
          </Descriptions.Item>
        </Descriptions>
      )}
      <Table
        size="small"
        dataSource={signals}
        rowKey={(_r, i) => String(i)}
        columns={signalColumns}
        pagination={{ pageSize: 50, showSizeChanger: false }}
        locale={{ emptyText: <Empty description={t('strategy.live.waitingSignals', { defaultValue: 'Waiting for signals...' })} /> }}
      />
    </Drawer>
  );
}

export { STATUS_COLORS, MODE_COLORS };
