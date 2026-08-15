import { useState, useEffect, useCallback, useRef } from 'react';
import { Table, Tag, Typography, Tabs, Descriptions, Empty, Spin, message } from 'antd';
import { useTranslation } from 'react-i18next';
import { strategyClient } from '@/client/connect';
import { logApi } from '@/client/log';
import { tradingApi } from '@/client/trading';
import type { MtPositionSnapshotItem } from '@/gen/ant/v1/mt_position_snapshot_pb';
import type { ScheduleRunLog } from '@/gen/ant/v1/log_schedule_pb';
import type { StrategySignalEvent } from '@/gen/ant/v1/strategy_runtime_pb';
import { strategyActiveApi } from '@/client/strategy';
import { shortId } from '../../LiveStrategyPageSignalDrawer';
import type { JoinedRow } from './strategyJoin';
import { LIVE_EXPAND_COL_WIDTH } from './strategyJoin';
import { formatMode } from './formatMode';
import { COL_PCT, buildPositionColumns, buildSignalColumns, buildLogColumns, buildParamsStr } from './expandedRowColumns';

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

  const positionColumns = buildPositionColumns(t, positionsWithLive, closingTicket, closingAll, handleClosePosition, handleCloseAll);
  const signalColumns = buildSignalColumns(t);
  const logColumns = buildLogColumns(t);
  const paramsStr = buildParamsStr(row);

  return (
    <div className="live-expanded-align" style={{ marginLeft: LIVE_EXPAND_COL_WIDTH }}>
    <style>{`
      .live-expanded-align .ant-tabs-nav-list { width: 100%; }
      ${COL_PCT.slice(0, 4).map((w, i) => `.live-expanded-align .ant-tabs-tab:nth-child(${i + 1}) { width: ${w}; justify-content: flex-start; }`).join('\n')}
    `}</style>
    <Tabs
      size="small"
      tabBarGutter={0}
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
