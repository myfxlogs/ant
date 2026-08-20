import { Tag, Typography, Button, Popconfirm } from 'antd';
import { CloseCircleOutlined, CopyOutlined } from '@ant-design/icons';
import type { TFunction } from 'i18next';
import type { MtPositionSnapshotItem } from '@/gen/ant/v1/mt_position_snapshot_pb';
import type { ScheduleRunLog } from '@/gen/ant/v1/log_schedule_pb';
import { formatTime } from '../../LiveStrategyPageSignalDrawer';
import type { JoinedRow } from './strategyJoin';

const { Text } = Typography;

export const COL_PCT = [
  '9.7561%', '7.3171%', '8.5366%', '10.9756%', '10.9756%',
  '9.7561%', '9.7561%', '9.7561%', '8.5366%', '14.6341%',
];

export function buildPositionColumns(
  t: TFunction,
  positionsWithLive: MtPositionSnapshotItem[],
  closingTicket: bigint | null,
  closingAll: boolean,
  handleClosePosition: (ticket: bigint, volume: string) => void,
  handleCloseAll: () => void,
) {
  return [
    { title: t('strategy.live.symbol', { defaultValue: 'Symbol' }), dataIndex: 'symbol', width: COL_PCT[0], onHeaderCell: () => ({ style: { paddingLeft: 0 } }), onCell: () => ({ style: { paddingLeft: 0 } }) },
    { title: t('strategy.live.signalType', { defaultValue: 'Type' }), dataIndex: 'type', width: COL_PCT[1], onHeaderCell: () => ({ style: { paddingLeft: 0 } }), onCell: () => ({ style: { paddingLeft: 0 } }), render: (v: string) => <Tag color={v === 'buy' ? 'green' : 'red'}>{v}</Tag> },
    { title: t('strategy.live.volume', { defaultValue: 'Volume' }), dataIndex: 'volume', width: COL_PCT[2], onHeaderCell: () => ({ style: { paddingLeft: 0 } }), onCell: () => ({ style: { paddingLeft: 0 } }) },
    { title: t('common.openPrice', { defaultValue: 'Open Price' }), dataIndex: 'openPrice', width: COL_PCT[3], onHeaderCell: () => ({ style: { paddingLeft: 0 } }), onCell: () => ({ style: { paddingLeft: 0 } }) },
    { title: t('common.currentPrice', { defaultValue: 'Current' }), dataIndex: 'currentPrice', width: COL_PCT[4] },
    { title: t('strategy.live.pnl', { defaultValue: 'PnL' }), dataIndex: 'profit', width: COL_PCT[5], render: (v: string) => {
      if (!v) return <Text type="secondary">-</Text>;
      const n = Number(v); const color = n >= 0 ? 'success' : 'danger';
      return <Text type={color}>{n >= 0 ? `+${v}` : v}</Text>;
    } },
    { title: t('strategy.live.sl', { defaultValue: 'SL' }), dataIndex: 'stopLoss', width: COL_PCT[6], render: (v: string) => v || '-' },
    { title: t('strategy.live.tp', { defaultValue: 'TP' }), dataIndex: 'takeProfit', width: COL_PCT[7], render: (v: string) => v || '-' },
    { title: 'Magic', dataIndex: 'magicNumber', width: COL_PCT[8], render: (v: bigint) => { const n = Number(v); return n ? <Text style={{ fontSize: 12, fontVariantNumeric: 'tabular-nums' }} type="secondary">{n}</Text> : <Text type="secondary">-</Text>; } },
    { title: <span style={{ display: 'inline-flex', alignItems: 'center', gap: 2 }}>{positionsWithLive.length > 0 && (
      <Popconfirm title={t('strategy.live.confirmCloseAll', { defaultValue: 'Close all positions?' })}
        onConfirm={handleCloseAll}>
        <Button size="small" type="text" danger icon={<CloseCircleOutlined />} loading={closingAll}
          style={{ padding: '0 4px', height: 22 }}>
          {t('strategy.live.closeAll', { defaultValue: 'Close All' })}
        </Button>
      </Popconfirm>
    )}</span>, key: 'close', width: COL_PCT[9], align: 'right' as const, render: (_: unknown, r: MtPositionSnapshotItem) => (
      <Popconfirm title={t('strategy.live.confirmClose', { defaultValue: 'Close this position?' })}
        onConfirm={() => handleClosePosition(r.ticket, r.volume)}>
        <Button size="small" type="text" danger icon={<CloseCircleOutlined />}
          loading={closingTicket === r.ticket} style={{ height: 22 }} />
      </Popconfirm>
    ) },
  ];
}

export function buildSignalColumns(t: TFunction) {
  return [
    { title: t('strategy.live.time', { defaultValue: 'Time' }), dataIndex: 'timestamp', width: 140, render: (v: { seconds?: bigint; nanos?: number } | null) => <Text style={{ fontSize: 12 }}>{formatTime(v)}</Text> },
    { title: t('strategy.live.signalType', { defaultValue: 'Type' }), dataIndex: 'signalType', width: 80, render: (v: string) => <Tag color={v === 'buy' || v === 'buy_limit' ? 'green' : v === 'sell' || v === 'sell_limit' ? 'red' : 'default'}>{v}</Tag> },
    { title: t('strategy.live.volume', { defaultValue: 'Volume' }), dataIndex: 'volume', width: 70 },
    { title: t('strategy.live.price', { defaultValue: 'Price' }), dataIndex: 'price', width: 80 },
  ];
}

export function buildLogColumns(t: TFunction) {
  return [
    { title: t('strategy.live.time', { defaultValue: 'Time' }), dataIndex: 'createdAt', width: 140, render: (v: unknown) => <Text style={{ fontSize: 12 }}>{formatTime(v as { seconds?: bigint; nanos?: number } | null)}</Text> },
    { title: t('common.status', { defaultValue: 'Status' }), dataIndex: 'status', width: 80, render: (v: string) => <Tag color={v === 'success' ? 'green' : v === 'failed' ? 'red' : 'default'}>{v}</Tag> },
    { title: t('common.message', { defaultValue: 'Message' }), key: 'message', ellipsis: true, render: (_: unknown, row: ScheduleRunLog) => {
      const error = row.errorMessage?.trim();
      const message = error || [row.kind, row.action, row.signalType].filter(Boolean).join(' / ');
      if (!message) return <Text type="secondary">-</Text>;
      return <Text style={{ fontSize: 12 }} copyable={{ text: message, icon: <CopyOutlined /> }} ellipsis={{ tooltip: message }}>{message}</Text>;
    } },
  ];
}

export function buildParamsStr(row: JoinedRow): string {
  return row.parameters ? Object.entries(row.parameters).map(([k, v]) => `${k}=${v}`).join(', ') : '-';
}
