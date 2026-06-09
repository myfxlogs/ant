// PaperAccountPanel — virtual paper trading account management + real-time portfolio display.
// Subscribes via SSE (WatchPaperAccount) for push-first portfolio updates.

import { useState, useEffect, useCallback, useRef } from 'react';
import { Button, Card, Input, List, Modal, Space, Tag, Typography, message, Statistic, Row, Col, Form, Select } from 'antd';
import { PlusOutlined, PlayCircleOutlined, StopOutlined, RiseOutlined, FallOutlined, ReloadOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { paperTradingClient, paperTradingStreamClient } from '@/client/connect';
import type { PaperAccount, PaperAccountUpdate } from '@/gen/ant/v1/paper_trading_pb';
import { useAuthStore } from '@/stores/authStore';

const { Text, Title } = Typography;

const TIMEFRAMES = ['M1', 'M5', 'M15', 'M30', 'H1', 'H4', 'D1', 'W1'];

interface RunningStrategy {
  symbol: string;
  timeframe: string;
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
  const [startTimeframe, setStartTimeframe] = useState('M5');
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
    if (!createName.trim()) { message.warning('Enter a name'); return; }
    try {
      const resp = await paperTradingClient.createPaperAccount({
        name: createName,
        initialBalance: createBalance,
      });
      message.success('Paper account created');
      setCreateName('');
      setCreateBalance('10000');
      // Add to frontend state immediately and subscribe.
      setAccounts(prev => [...prev, resp]);
      subscribeAccount(resp.id);
    } catch { message.error('Create failed'); }
  }, [createName, createBalance, subscribeAccount]);

  const openStartModal = useCallback((accountId: string) => {
    setStartTargetId(accountId);
    setStartSymbol('XAUUSD');
    setStartTimeframe('M5');
    setStartCode('');
    setStartModalOpen(true);
  }, []);

  const handleStart = useCallback(async () => {
    if (!startCode.trim()) { message.warning('Paste your strategy code'); return; }
    setStarting(true);
    try {
      await paperTradingClient.startPaperStrategy({
        paperAccountId: startTargetId,
        strategyCode: startCode,
        symbol: startSymbol,
        timeframe: startTimeframe,
        params: {},
      });
      message.success('Paper strategy started');
      setRunning(prev => ({
        ...prev,
        [startTargetId]: { symbol: startSymbol, timeframe: startTimeframe },
      }));
      setStartModalOpen(false);
    } catch { message.error('Start failed'); }
    setStarting(false);
  }, [startCode, startTargetId, startSymbol, startTimeframe]);

  const handleStop = useCallback(async (accountId: string) => {
    setStopping(prev => ({ ...prev, [accountId]: true }));
    try {
      await paperTradingClient.stopPaperStrategy({ paperAccountId: accountId });
      message.success('Paper strategy stopped');
      setRunning(prev => {
        const next = { ...prev };
        delete next[accountId];
        return next;
      });
    } catch { message.error('Stop failed'); }
    setStopping(prev => ({ ...prev, [accountId]: false }));
  }, []);

  return (
    <div style={{ padding: 16 }}>
      <Title level={5} style={{ marginBottom: 16 }}>📊 Paper Trading</Title>

      {/* Create account */}
      <Card size="small" style={{ marginBottom: 16, background: '#fafafa' }}>
        <Space direction="vertical" style={{ width: '100%' }}>
          <Text strong style={{ fontSize: 12 }}>Create Paper Account</Text>
          <Space>
            <Input size="small" placeholder="Account name" value={createName}
              onChange={e => setCreateName(e.target.value)} style={{ width: 160 }} />
            <Input size="small" placeholder="Balance" value={createBalance}
              onChange={e => setCreateBalance(e.target.value)} style={{ width: 100 }} />
            <Button size="small" type="primary" icon={<PlusOutlined />}
              onClick={handleCreate}>Create</Button>
          </Space>
        </Space>
      </Card>

      {/* Account list */}
      <List
        loading={loading}
        dataSource={accounts}
        locale={{ emptyText: 'No paper accounts. Create one to start simulated trading.' }}
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
                  <Statistic title="Balance" value={a.currentBalance || '0'}
                    valueStyle={{ fontSize: 14 }} prefix="$" />
                </Col>
                <Col span={8}>
                  <Statistic title="Equity" value={a.equity || '0'}
                    valueStyle={{ fontSize: 14, color: eq >= ib ? '#26a69a' : '#ef5350' }}
                    prefix={eq >= ib ? <RiseOutlined /> : <FallOutlined />} />
                </Col>
              </Row>
              <Space style={{ marginTop: 8 }}>
                {isRunning ? (
                  <>
                    <Tag color="green">
                      Running {running[a.id].symbol} {running[a.id].timeframe}
                    </Tag>
                    <Button size="small" icon={<StopOutlined />} danger
                      loading={isBusy}
                      onClick={() => handleStop(a.id)}>
                      Stop
                    </Button>
                  </>
                ) : (
                  <>
                    <Button size="small" icon={<PlayCircleOutlined />}
                      onClick={() => openStartModal(a.id)}>
                      Start
                    </Button>
                    <Button size="small" icon={<StopOutlined />} disabled>Stop</Button>
                  </>
                )}
                <Button size="small" icon={<ReloadOutlined />}
                  onClick={() => subscribeAccount(a.id)}>
                  Watch
                </Button>
                <Tag color="blue">Paper</Tag>
              </Space>
            </Card>
          );
        }}
      />

      {/* Start Strategy Modal */}
      <Modal
        title="Start Paper Strategy"
        open={startModalOpen}
        onCancel={() => setStartModalOpen(false)}
        onOk={handleStart}
        confirmLoading={starting}
        okText="Start"
      >
        <Form layout="vertical" size="small" style={{ marginTop: 16 }}>
          <Form.Item label="Symbol">
            <Input
              value={startSymbol}
              onChange={e => setStartSymbol(e.target.value.toUpperCase())}
              placeholder="XAUUSD"
            />
          </Form.Item>
          <Form.Item label="Timeframe">
            <Select value={startTimeframe} onChange={setStartTimeframe}>
              {TIMEFRAMES.map(tf => (
                <Select.Option key={tf} value={tf}>{tf}</Select.Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item label="Strategy Code (Python)">
            <Input.TextArea
              rows={6}
              value={startCode}
              onChange={e => setStartCode(e.target.value)}
              placeholder="def run(context):&#10;    ..."
              style={{ fontFamily: 'monospace', fontSize: 12 }}
            />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
