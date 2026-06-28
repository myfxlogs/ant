// PaperAccountPanel — virtual paper trading account management + real-time portfolio display.
// Subscribes via SSE (WatchPaperAccount) for push-first portfolio updates.

import { useState, useEffect, useCallback, useRef } from 'react';
import { Button, Card, Input, List, Modal, Space, Tag, Typography, message, Statistic, Row, Col, Form, Select } from 'antd';
import { PlusOutlined, PlayCircleOutlined, StopOutlined, RiseOutlined, FallOutlined, ReloadOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next'
import { TRADING_BALANCE_KEY, TRADING_EQUITY_KEY } from '@/gen/ant/v1/i18n/trading_keys';
import { NO_ACCOUNTS_KEY } from '@/gen/ant/v1/i18n/dashboard_keys';
import { CREATE_KEY } from '@/gen/ant/v1/i18n/strategy_library_keys';
import { TRADING_TITLE_KEY } from '@/gen/ant/v1/i18n/strategy_market_regime_keys';
import { ACCOUNT_NAME_KEY, CREATE_ACCOUNT_KEY, MESSAGES_CREATED_KEY, MESSAGES_CREATE_FAILED_KEY, MESSAGES_ENTER_NAME_KEY, MESSAGES_PASTE_CODE_KEY, MESSAGES_START_FAILED_KEY, MESSAGES_STOP_FAILED_KEY, MESSAGES_STRATEGY_STARTED_KEY, MESSAGES_STRATEGY_STOPPED_KEY, PAPER_KEY, RUNNING_KEY, START_KEY, START_STRATEGY_KEY, TRADING_STOP_KEY, STRATEGY_CODE_KEY, TRADING_SYMBOL_KEY, TIMEFRAME_KEY, WATCH_KEY } from '@/gen/ant/v1/i18n/strategy_paper_keys';

;
import { paperTradingClient, paperTradingStreamClient } from '@/client/connect';
import { strategyActiveApi } from '@/client/strategy';
import type { PaperAccount, PaperAccountUpdate } from '@/gen/ant/v1/paper_trading_pb';
import { useAuthStore } from '@/stores/authStore';
import { TIMEFRAMES } from '@/constants/timeframes';

const { Text, Title } = Typography;

interface RunningStrategy {
  symbol: string;
  timeframe: string;
  runId?: string;
}

export default function PaperAccountPanel() {
  const { t } = useTranslation();
  const userId = useAuthStore(s => s.user?.id);
  const [accounts, setAccounts] = useState<PaperAccount[]>([]);
  const [loading, setLoading] = useState(false);
  const [createName, setCreateName] = useState('');
  const [createBalance, setCreateBalance] = useState('10000');

  // Start modal state
  const [startModalOpen, setStartModalOpen] = useState(false);
  const [startTargetId, setStartTargetId] = useState('');
  const [startSymbol, setStartSymbol] = useState('XAUUSD');
  const [startTimeframe, setStartTimeframe] = useState('5m');
  const [startCode, setStartCode] = useState('');
  const [starting, setStarting] = useState(false);

  // Running strategies: accountId → RunningStrategy
  const [running, setRunning] = useState<Record<string, RunningStrategy>>({});
  const [stopping, setStopping] = useState<Record<string, boolean>>({});

  // SSE subscriptions ref
  const streamRefs = useRef<Map<string, { abort: () => void }>>(new Map());

  const loadAccounts = useCallback(async () => {
    if (!userId) return;
    setLoading(true);
    try {
      const resp = await paperTradingClient.listPaperAccounts({});
      setAccounts(resp.accounts || []);
    } catch { /* non-critical */ }
    setLoading(false);
  }, [userId]);

  useEffect(() => { loadAccounts(); }, [loadAccounts]);

  // ── SSE WatchPaperAccount ──
  const subscribeAccount = useCallback((accountId: string) => {
    // Prevent duplicate subscriptions.
    if (streamRefs.current.has(accountId)) return;

    const abort = new AbortController();
    let cancelled = false;
    streamRefs.current.set(accountId, { abort: () => { cancelled = true; abort.abort(); } });

    (async () => {
      try {
        const stream = paperTradingStreamClient.watchPaperAccount(
          { paperAccountId: accountId },
          { signal: abort.signal },
        );
        for await (const update of stream) {
          if (cancelled) break;
          const u = update as PaperAccountUpdate;
          setAccounts(prev =>
            prev.map(a => (a.id === accountId ? { ...u.account! } : a)),
          );
        }
      } catch (err) {
        if (!cancelled) {
          console.warn('[PaperAccountPanel] SSE stream ended for', accountId, err);
        }
      } finally {
        streamRefs.current.delete(accountId);
      }
    })();
  }, []);

  const unsubscribeAccount = useCallback((accountId: string) => {
    const ref = streamRefs.current.get(accountId);
    if (ref) {
      ref.abort();
      streamRefs.current.delete(accountId);
    }
  }, []);

  // Subscribe to all accounts on load.
  useEffect(() => {
    for (const a of accounts) {
      subscribeAccount(a.id);
    }
    return () => {
      // Cleanup all on unmount.
      for (const [id] of streamRefs.current) {
        unsubscribeAccount(id);
      }
    };
  }, [accounts.map(a => a.id).join(','), subscribeAccount, unsubscribeAccount]);

  // ── Handlers ──
  const handleCreate = useCallback(async () => {
    if (!createName.trim()) { message.warning(t(MESSAGES_ENTER_NAME_KEY)); return; }
    try {
      const resp = await paperTradingClient.createPaperAccount({
        name: createName,
        initialBalance: createBalance,
      });
      message.success(t(MESSAGES_CREATED_KEY));
      setCreateName('');
      setCreateBalance('10000');
      // Add to frontend state immediately and subscribe.
      setAccounts(prev => [...prev, resp]);
      subscribeAccount(resp.id);
    } catch { message.error(t(MESSAGES_CREATE_FAILED_KEY)); }
  }, [createName, createBalance, subscribeAccount]);

  const openStartModal = useCallback((accountId: string) => {
    setStartTargetId(accountId);
    setStartSymbol('XAUUSD');
    setStartTimeframe('5m');
    setStartCode('');
    setStartModalOpen(true);
  }, []);

  const handleStart = useCallback(async () => {
    if (!startCode.trim()) { message.warning(t(MESSAGES_PASTE_CODE_KEY)); return; }
    setStarting(true);
    try {
      const resp = await strategyActiveApi.start({
        accountId: startTargetId,
        strategyCode: startCode,
        symbol: startSymbol,
        timeframe: startTimeframe,
        mode: 'paper',
        params: {},
      });
      message.success(t(MESSAGES_STRATEGY_STARTED_KEY));
      setRunning(prev => ({
        ...prev,
        [startTargetId]: { symbol: startSymbol, timeframe: startTimeframe, runId: resp.runId || undefined },
      }));
      setStartModalOpen(false);
    } catch { message.error(t(MESSAGES_START_FAILED_KEY)); }
    setStarting(false);
  }, [startCode, startTargetId, startSymbol, startTimeframe]);

  const handleStop = useCallback(async (accountId: string) => {
    setStopping(prev => ({ ...prev, [accountId]: true }));
    try {
      const runId = running[accountId]?.runId;
      if (!runId) {
        message.error(t(MESSAGES_STOP_FAILED_KEY));
        return;
      }
      await strategyActiveApi.stop(runId);
      message.success(t(MESSAGES_STRATEGY_STOPPED_KEY));
      setRunning(prev => {
        const next = { ...prev };
        delete next[accountId];
        return next;
      });
    } catch { message.error(t(MESSAGES_STOP_FAILED_KEY)); }
    setStopping(prev => ({ ...prev, [accountId]: false }));
  }, [running]);

  return (
    <div style={{ padding: 16 }}>
      <Title level={5} style={{ marginBottom: 16 }}>{t(TRADING_TITLE_KEY)}</Title>

      {/* Create account */}
      <Card size="small" style={{ marginBottom: 16, background: '#fafafa' }}>
        <Space direction="vertical" style={{ width: '100%' }}>
          <Text strong style={{ fontSize: 12 }}>{t(CREATE_ACCOUNT_KEY)}</Text>
          <Space>
            <Input size="small" placeholder={t(ACCOUNT_NAME_KEY)} value={createName}
              onChange={e => setCreateName(e.target.value)} style={{ width: 160 }} />
            <Input size="small" placeholder={t(TRADING_BALANCE_KEY)} value={createBalance}
              onChange={e => setCreateBalance(e.target.value)} style={{ width: 100 }} />
            <Button size="small" type="primary" icon={<PlusOutlined />}
              onClick={handleCreate}>{t(CREATE_KEY)}</Button>
          </Space>
        </Space>
      </Card>

      {/* Account list */}
      <List
        loading={loading}
        dataSource={accounts}
        locale={{ emptyText: t(NO_ACCOUNTS_KEY) }}
        renderItem={(a: PaperAccount) => {
          const isRunning = !!running[a.id];
          const isBusy = stopping[a.id] || false;
          const eq = parseFloat(a.equity || '0');
          const ib = parseFloat(a.initialBalance || '0');
          return (
            <Card size="small" style={{ marginBottom: 8 }} key={a.id}>
              <Row gutter={12}>
                <Col span={8}>
                  <Text strong>{a.name}</Text>
                  <br />
                  <Text type="secondary" style={{ fontSize: 11 }}>{a.id?.slice(0, 8)}</Text>
                </Col>
                <Col span={8}>
                  <Statistic title={t(TRADING_BALANCE_KEY)} value={a.currentBalance || '0'}
                    valueStyle={{ fontSize: 14 }} prefix="$" />
                </Col>
                <Col span={8}>
                  <Statistic title={t(TRADING_EQUITY_KEY)} value={a.equity || '0'}
                    valueStyle={{ fontSize: 14, color: eq >= ib ? '#26a69a' : '#ef5350' }}
                    prefix={eq >= ib ? <RiseOutlined /> : <FallOutlined />} />
                </Col>
              </Row>
              <Space style={{ marginTop: 8 }}>
                {isRunning ? (
                  <>
                    <Tag color="green">
                      {t(RUNNING_KEY, { symbol: running[a.id].symbol, timeframe: running[a.id].timeframe })}
                    </Tag>
                    <Button size="small" icon={<StopOutlined />} danger
                      loading={isBusy}
                      onClick={() => handleStop(a.id)}>
                      {t(TRADING_STOP_KEY)}
                    </Button>
                  </>
                ) : (
                  <>
                    <Button size="small" icon={<PlayCircleOutlined />}
                      onClick={() => openStartModal(a.id)}>
                      {t(START_KEY)}
                    </Button>
                    <Button size="small" icon={<StopOutlined />} disabled>{t(TRADING_STOP_KEY)}</Button>
                  </>
                )}
                <Button size="small" icon={<ReloadOutlined />}
                  onClick={() => subscribeAccount(a.id)}>
                  {t(WATCH_KEY)}
                </Button>
                <Tag color="blue">{t(PAPER_KEY)}</Tag>
              </Space>
            </Card>
          );
        }}
      />

      {/* Start Strategy Modal */}
      <Modal
        title={t(START_STRATEGY_KEY)}
        open={startModalOpen}
        onCancel={() => setStartModalOpen(false)}
        onOk={handleStart}
        confirmLoading={starting}
        okText={t(START_KEY)}
      >
        <Form layout="vertical" size="small" style={{ marginTop: 16 }}>
          <Form.Item label={t(TRADING_SYMBOL_KEY)}>
            <Input
              value={startSymbol}
              onChange={e => setStartSymbol(e.target.value.toUpperCase())}
              placeholder="XAUUSD"
            />
          </Form.Item>
          <Form.Item label={t(TIMEFRAME_KEY)}>
            <Select value={startTimeframe} onChange={setStartTimeframe}>
              {TIMEFRAMES.map(tf => (
                <Select.Option key={tf} value={tf}>{tf}</Select.Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item label={t(STRATEGY_CODE_KEY)}>
            <Input.TextArea
              rows={6}
              value={startCode}
              onChange={e => setStartCode(e.target.value)}
              placeholder="class MyStrategy(StrategyBase):&#10;    def on_bar(self):&#10;        ..."
              style={{ fontFamily: 'monospace', fontSize: 12 }}
            />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
