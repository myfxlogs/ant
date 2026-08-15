import { useState, useEffect, useCallback, useRef } from 'react';
import { Table, Tag, Typography, Tabs, Descriptions, Empty, Spin, Button, Popconfirm, message } from 'antd';
import { CloseCircleOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { strategyClient } from '@/client/connect';
import { logApi } from '@/client/log';
import { tradingApi } from '@/client/trading';
import type { MtPositionSnapshotItem } from '@/gen/ant/v1/mt_position_snapshot_pb';
import type { ScheduleRunLog } from '@/gen/ant/v1/log_schedule_pb';
import type { StrategySignalEvent } from '@/gen/ant/v1/strategy_runtime_pb';
import { strategyActiveApi } from '@/client/strategy';
import { formatTime, shortId } from '../../LiveStrategyPageSignalDrawer';
import type { JoinedRow } from './strategyJoin';
import { LIVE_EXPAND_COL_WIDTH } from './strategyJoin';
import { formatMode } from './formatMode';

const { Text } = Typography;

interface Props {
  row: JoinedRow;
  activeVersion: number;
  liveBid?: string;
  liveAsk?: string;
}

export default function ScheduleExpandedRow({ row, activeVersion, liveBid, liveAsk }: Props) {
  const { t } = useTranslation();
  const [positions, setPositions] = useState<MtPositionSnapshotItem[]>([]);
  const [positionsLoading, setPositionsLoading] = useState(false);
  const [logs, setLogs] = useState<ScheduleRunLog[]>([]);
  const [logsLoading, setLogsLoading] = useState(false);
  const [signals, setSignals] = useState<StrategySignalEvent[]>([]);
  const [signalsLoading, setSignalsLoading] = useState(false);
  const [closingTicket, setClosingTicket] = useState<bigint | null>(null);
  const [closingAll, setClosingAll] = useState(false);
  const signalAbortRef = useRef<AbortController | null>(null);
  const positionsLoadedRef = useRef(false);

  const fetchPositions = useCallback(async (silent = false) => {
    if (!row.id) return;
    if (!silent || !positionsLoadedRef.current) setPositionsLoading(true);
    try {
      const resp = await strategyClient.getSchedulePositions({ scheduleId: row.id });
      setPositions(resp.positions || []);
      positionsLoadedRef.current = true;
    } catch { setPositions([]); }
    setPositionsLoading(false);
  }, [row.id]);

  const fetchLogs = useCallback(async () => {
    if (!row.id) return;
    setLogsLoading(true);
    try {
      const resp = await logApi.getScheduleRunLogs({ scheduleId: row.id, page: 1, pageSize: 20 });
      setLogs(resp.logs || []);
    } catch { setLogs([]); }
    setLogsLoading(false);
  }, [row.id]);

  const fetchSignals = useCallback(async () => {
    const runId = row.active?.runId;
    if (!runId) { setSignals([]); return; }
    if (signalAbortRef.current) signalAbortRef.current.abort();
    const ctrl = new AbortController();
    signalAbortRef.current = ctrl;
    setSignalsLoading(true);
    let receivedAny = false;
    (async () => {
      try {
        for await (const ev of strategyActiveApi.watchSignals(runId, ctrl.signal)) {
          if (!receivedAny) { receivedAny = true; setSignalsLoading(false); }
          if (!ev.signalType) continue;
          setSignals(prev => [...prev.slice(-19), ev as StrategySignalEvent]);
        }
      } catch { /* stream ended */ }
      setSignalsLoading(false);
    })();
    setTimeout(() => { if (!receivedAny) setSignalsLoading(false); }, 3000);
  }, [row.active?.runId]);

  useEffect(() => {
    void fetchPositions();
    void fetchLogs();
    void fetchSignals();
    return () => { if (signalAbortRef.current) signalAbortRef.current.abort(); };
  }, [fetchPositions, fetchLogs, fetchSignals]);

  useEffect(() => {
    if (activeVersion > 0 && row.active) void fetchPositions(true);
  }, [activeVersion, row.active, fetchPositions]);

  const handleClosePosition = useCallback(async (ticket: bigint, volume: string) => {
    if (!row.accountId) return;
    setClosingTicket(ticket);
    try {
      const result = await tradingApi.orderClose({
        accountId: row.accountId,
        ticket,
        volume: Number(volume) || undefined,
      });
      if (result.error) message.error(result.error);
      else { message.success(t('strategy.live.positionClosed', { defaultValue: 'Position closed' })); void fetchPositions(true); }
    } catch { message.error(t('strategy.live.closeFailed', { defaultValue: 'Close failed' })); }
    setClosingTicket(null);
  }, [row.accountId, fetchPositions, t]);

  const handleCloseAll = useCallback(async () => {
    if (!row.accountId || positions.length === 0) return;
    setClosingAll(true);
    let ok = 0, fail = 0;
    for (const p of positions) {
      const result = await tradingApi.orderClose({
        accountId: row.accountId,
        ticket: p.ticket,
        volume: Number(p.volume) || undefined,
      });
      if (result.error) fail++; else ok++;
    }
    setClosingAll(false);
    if (ok > 0) { message.success(t('strategy.live.positionClosed', { defaultValue: 'Position closed' })); void fetchPositions(true); }
    if (fail > 0) message.error(`${fail} position(s) failed to close`);
  }, [row.accountId, positions, fetchPositions, t]);

  const livePrice = liveBid || liveAsk || '';
  const positionsWithLive = positions.map(p => {
    if (!livePrice) return p;
    const cp = p.type === 'buy' ? (liveBid || liveAsk || p.currentPrice) : (liveAsk || liveBid || p.currentPrice);
    const openNum = Number(p.openPrice);
    const curNum = Number(cp);
    const volNum = Number(p.volume);
    let liveProfit = p.profit;
    if (openNum && curNum && volNum) {
      const diff = p.type === 'buy' ? (curNum - openNum) : (openNum - curNum);
      liveProfit = (diff * volNum).toFixed(2);
    }
    return { ...p, currentPrice: cp, profit: liveProfit };
  });

  const positionColumns = [
    { title: t('strategy.live.symbol', { defaultValue: 'Symbol' }), dataIndex: 'symbol', width: 80, onHeaderCell: () => ({ style: { paddingLeft: 0 } }), onCell: () => ({ style: { paddingLeft: 0 } }) },
    { title: t('strategy.live.signalType', { defaultValue: 'Type' }), dataIndex: 'type', width: 60, onHeaderCell: () => ({ style: { paddingLeft: 0 } }), onCell: () => ({ style: { paddingLeft: 0 } }), render: (v: string) => <Tag color={v === 'buy' ? 'green' : 'red'}>{v}</Tag> },
    { title: t('strategy.live.volume', { defaultValue: 'Volume' }), dataIndex: 'volume', width: 70, onHeaderCell: () => ({ style: { paddingLeft: 0 } }), onCell: () => ({ style: { paddingLeft: 0 } }) },
    { title: t('common.openPrice', { defaultValue: 'Open Price' }), dataIndex: 'openPrice', width: 90, onHeaderCell: () => ({ style: { paddingLeft: 0 } }), onCell: () => ({ style: { paddingLeft: 0 } }) },
    { title: t('common.currentPrice', { defaultValue: 'Current' }), dataIndex: 'currentPrice', width: 90 },
    { title: t('strategy.live.pnl', { defaultValue: 'PnL' }), dataIndex: 'profit', width: 80, render: (v: string) => {
      if (!v) return <Text type="secondary">-</Text>;
      const n = Number(v); const color = n >= 0 ? 'success' : 'danger';
      return <Text type={color}>{n >= 0 ? `+${v}` : v}</Text>;
    } },
    { title: t('strategy.live.sl', { defaultValue: 'SL' }), dataIndex: 'stopLoss', width: 80, render: (v: string) => v || '-' },
    { title: t('strategy.live.tp', { defaultValue: 'TP' }), dataIndex: 'takeProfit', width: 80, render: (v: string) => v || '-' },
    { title: <span style={{ display: 'inline-flex', alignItems: 'center', gap: 2 }}>{positionsWithLive.length > 0 && (
      <Popconfirm title={t('strategy.live.confirmCloseAll', { defaultValue: 'Close all positions?' })}
        onConfirm={handleCloseAll}>
        <Button size="small" type="text" danger icon={<CloseCircleOutlined />} loading={closingAll}
          style={{ padding: '0 4px', height: 22 }}>
          {t('strategy.live.closeAll', { defaultValue: 'Close All' })}
        </Button>
      </Popconfirm>
    )}</span>, key: 'close', width: 120, render: (_: unknown, r: MtPositionSnapshotItem) => (
      <Popconfirm title={t('strategy.live.confirmClose', { defaultValue: 'Close this position?' })}
        onConfirm={() => handleClosePosition(r.ticket, r.volume)}>
        <Button size="small" type="text" danger icon={<CloseCircleOutlined />}
          loading={closingTicket === r.ticket} style={{ height: 22 }} />
      </Popconfirm>
    ) },
    { title: 'Magic', dataIndex: 'magicNumber', width: 70, render: (v: bigint) => { const n = Number(v); return n ? <Text style={{ fontSize: 12, fontVariantNumeric: 'tabular-nums' }} type="secondary">{n}</Text> : <Text type="secondary">-</Text>; } },
  ];

  const signalColumns = [
    { title: t('strategy.live.time', { defaultValue: 'Time' }), dataIndex: 'timestamp', width: 140, render: (v: { seconds?: bigint; nanos?: number } | null) => <Text style={{ fontSize: 12 }}>{formatTime(v)}</Text> },
    { title: t('strategy.live.signalType', { defaultValue: 'Type' }), dataIndex: 'signalType', width: 80, render: (v: string) => <Tag color={v === 'buy' || v === 'buy_limit' ? 'green' : v === 'sell' || v === 'sell_limit' ? 'red' : 'default'}>{v}</Tag> },
    { title: t('strategy.live.volume', { defaultValue: 'Volume' }), dataIndex: 'volume', width: 70 },
    { title: t('strategy.live.price', { defaultValue: 'Price' }), dataIndex: 'price', width: 80 },
  ];

  const logColumns = [
    { title: t('strategy.live.time', { defaultValue: 'Time' }), dataIndex: 'createdAt', width: 140, render: (v: unknown) => <Text style={{ fontSize: 12 }}>{formatTime(v as { seconds?: bigint; nanos?: number } | null)}</Text> },
    { title: t('common.status', { defaultValue: 'Status' }), dataIndex: 'status', width: 80, render: (v: string) => <Tag color={v === 'success' ? 'green' : v === 'failed' ? 'red' : 'default'}>{v}</Tag> },
    { title: t('common.message', { defaultValue: 'Message' }), dataIndex: 'message', ellipsis: true, render: (v: string) => v ? <Text style={{ fontSize: 12 }}>{v}</Text> : <Text type="secondary">-</Text> },
  ];

  const paramsStr = row.parameters ? Object.entries(row.parameters).map(([k, v]) => `${k}=${v}`).join(', ') : '-';

  return (
    <div className="live-expanded-align" style={{ marginLeft: LIVE_EXPAND_COL_WIDTH }}>
    <Tabs
      size="small"
      tabBarStyle={{ marginBottom: 0 }}
      items={[
        {
          key: 'positions',
          label: <span>{t('strategy.live.positions', { defaultValue: 'Positions' })} {positions.length > 0 && <Tag color="blue">{positions.length}</Tag>}</span>,
          children: (
            <div>
            <Spin spinning={positionsLoading}>
              <Table size="small" dataSource={positionsWithLive} rowKey="ticket" columns={positionColumns} pagination={false}
                locale={{ emptyText: <Empty description={t('strategy.live.noPositions', { defaultValue: 'No open positions' })} /> }} />
            </Spin>
            </div>
          ),
        },
        {
          key: 'signals',
          label: <span>{t('strategy.live.signals', { defaultValue: 'Signals' })} {signals.length > 0 && <Tag color="blue">{signals.length}</Tag>}</span>,
          children: (
            <div>
            <Spin spinning={signalsLoading}>
              <Table size="small" dataSource={signals} rowKey={(_r, i) => String(i)} columns={signalColumns} pagination={false}
                locale={{ emptyText: <Empty description={t('strategy.live.waitingSignals', { defaultValue: 'Waiting for signals...' })} /> }} />
            </Spin>
            </div>
          ),
        },
        {
          key: 'logs',
          label: <span>{t('strategy.live.logs', { defaultValue: 'Logs' })}</span>,
          children: (
            <div>
            <Spin spinning={logsLoading}>
              <Table size="small" dataSource={logs} rowKey="id" columns={logColumns} pagination={false}
                locale={{ emptyText: <Empty description={t('common.noData', { defaultValue: 'No data' })} /> }} />
            </Spin>
            </div>
          ),
        },
        {
          key: 'config',
          label: <span>{t('strategy.live.config', { defaultValue: 'Config' })}</span>,
          children: (
            <div>
            <Descriptions size="small" column={2} bordered>
              <Descriptions.Item label={t('strategy.live.symbol', { defaultValue: 'Symbol' })}>{row.symbol}</Descriptions.Item>
              <Descriptions.Item label={t('strategy.live.timeframe', { defaultValue: 'TF' })}>{row.timeframe}</Descriptions.Item>
              <Descriptions.Item label={t('strategy.schedules.table.schedule', { defaultValue: 'Schedule' })}>{row.scheduleType}</Descriptions.Item>
              <Descriptions.Item label={t('strategy.live.mode', { defaultValue: 'Mode' })}>{formatMode(row.active?.mode, t)}</Descriptions.Item>
              <Descriptions.Item label={t('strategy.live.parameters', { defaultValue: 'Parameters' })} span={2}>{paramsStr}</Descriptions.Item>
              {row.active && (
                <Descriptions.Item label={t('strategy.live.runId', { defaultValue: 'Run ID' })}>
                  <Text code>{shortId(row.active.runId)}</Text>
                </Descriptions.Item>
              )}
            </Descriptions>
            </div>
          ),
        },
      ]}
    />
    </div>
  );
}
