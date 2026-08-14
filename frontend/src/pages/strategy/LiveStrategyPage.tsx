import { useState, useEffect, useCallback, useMemo } from 'react';
import { Card, message, Tabs, Alert } from 'antd';
import { EyeOutlined, ClockCircleOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { useSearchParams, useNavigate } from 'react-router-dom';
import { strategyActiveApi } from '@/client/strategy';
import { strategyScheduleV2Api, strategyTemplateApi } from '@/client/strategy-schedules';
import { scheduleHealthApi } from '@/client/scheduleHealth';
import type { ActiveStrategy } from '@/gen/ant/v1/strategy_runtime_pb';
import { formatTime } from './LiveStrategyPageSignalDrawer';
import ScheduleLogsModal from './components/ScheduleLogsModal';
import ScheduleHealthModal from './components/ScheduleHealthModal';
import MyStrategiesTable from './components/live/MyStrategiesTable';
import RunHistoryTab from './components/live/RunHistoryTab';
import EditParamsModal from './components/live/EditParamsModal';
import { joinSchedulesWithActive, findOrphanRuns } from './components/live/strategyJoin';
import type { JoinedRow } from './components/live/strategyJoin';
import type { ScheduleRow, TemplateOption, ScheduleHealthSummary } from './hooks/libraryTypes';
import { useAccountsAndSymbols } from './hooks/useAccountsAndSymbols';

function mapTabParam(tab: string | null): string {
  if (tab === 'history') return 'history';
  return 'strategies';
}

export default function LiveStrategyPage() {
  const { t } = useTranslation();
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const initialTab = mapTabParam(searchParams.get('tab'));
  const highlightScheduleId = searchParams.get('scheduleId') || null;
  const healthId = searchParams.get('healthId') || null;
  const [activeTab, setActiveTab] = useState(initialTab);
  const [activeStrategies, setActiveStrategies] = useState<ActiveStrategy[]>([]);
  const [schedules, setSchedules] = useState<ScheduleRow[]>([]);
  const [templates, setTemplates] = useState<TemplateOption[]>([]);
  const [loading, setLoading] = useState(false);
  const [stopping, setStopping] = useState<string | null>(null);
  const [streamError, setStreamError] = useState(false);
  const [logsModalScheduleId, setLogsModalScheduleId] = useState<string | null>(null);
  const [activeVersion, setActiveVersion] = useState(0);
  const [healthOpen, setHealthOpen] = useState(false);
  const [healthLoading, setHealthLoading] = useState(false);
  const [healthTarget, setHealthTarget] = useState<ScheduleRow | null>(null);
  const [healthSummary, setHealthSummary] = useState<ScheduleHealthSummary | null>(null);
  const { accounts: scheduleAccounts, fetchAccounts } = useAccountsAndSymbols();

  useEffect(() => { void fetchAccounts(); }, [fetchAccounts]);

  useEffect(() => {
    if (activeTab !== 'strategies') return;
    let active = true;
    const connect = async () => {
      while (active) {
        const ctrl = new AbortController();
        try {
          setLoading(true);
          for await (const event of strategyActiveApi.watchActive('', ctrl.signal)) {
            if (event.heartbeat) continue;
            setActiveStrategies((event.strategies || []) as ActiveStrategy[]);
            setActiveVersion(v => v + 1);
            setLoading(false);
            setStreamError(false);
          }
        } catch { /* reconnect */ }
        ctrl.abort();
        if (!active) break;
        setStreamError(true);
        await new Promise(r => setTimeout(r, 2000));
      }
    };
    connect();
    return () => { active = false; };
  }, [activeTab]);

  useEffect(() => {
    if (activeTab !== 'strategies') return;
    let active = true;
    const connect = async () => {
      const [tpls, schs] = await Promise.all([strategyTemplateApi.list(), strategyScheduleV2Api.list()]);
      if (!active) return;
      const tplsOpts: TemplateOption[] = [];
      (tpls || []).forEach((tpl: { id?: string; name?: string; isPublic?: boolean }) => {
        if (tpl?.id) tplsOpts.push({ id: tpl.id, name: tpl.name || '', isPublic: tpl.isPublic });
      });
      setTemplates(tplsOpts);
      setSchedules(schs as ScheduleRow[]);
      while (active) {
        const ctrl = new AbortController();
        try {
          const streamDone = (async () => {
            for await (const event of strategyScheduleV2Api.watch(ctrl.signal)) {
              if (!active) break;
              setSchedules((event.schedules || []) as ScheduleRow[]);
            }
          })();
          await Promise.race([streamDone, new Promise(r => setTimeout(r, 90_000))]);
          ctrl.abort();
        } catch { /* reconnect */ }
        if (!active) break;
        await new Promise(r => setTimeout(r, 2000));
      }
    };
    connect();
    return () => { active = false; };
  }, [activeTab]);

  const joinedRows = useMemo<JoinedRow[]>(
    () => joinSchedulesWithActive(schedules, activeStrategies),
    [schedules, activeStrategies],
  );

  const orphanRuns = useMemo<ActiveStrategy[]>(
    () => findOrphanRuns(activeStrategies, schedules),
    [activeStrategies, schedules],
  );

  const handleStop = async (runId: string) => {
    setStopping(runId);
    try {
      const r = await strategyActiveApi.stop(runId);
      if (r.success) message.success(t('strategy.live.stopSuccess', { defaultValue: 'Strategy stopped' }));
      else message.error(r.error || t('strategy.live.stopFailed', { defaultValue: 'Failed to stop' }));
    } catch (e: unknown) {
      message.error(e instanceof Error ? e.message : t('strategy.live.stopFailed', { defaultValue: 'Failed to stop' }));
    }
    setStopping(null);
  };

  const onToggleActive = useCallback(async (row: ScheduleRow, next: boolean) => {
    try {
      await strategyScheduleV2Api.toggle(row.id, next);
      message.success(next ? t('common.enabled', { defaultValue: 'Enabled' }) : t('common.disabled', { defaultValue: 'Disabled' }));
    } catch (e: unknown) {
      message.error(e instanceof Error ? e.message : t('common.operationFailed', { defaultValue: 'Operation failed' }));
    }
  }, [t]);

  const onManualTrigger = useCallback(async (row: ScheduleRow) => {
    try {
      const tpl = await strategyTemplateApi.get(row.templateId);
      const code = String(tpl?.code || '');
      if (!code) throw new Error('Template code empty');
      const resp = await strategyActiveApi.start({
        accountId: row.accountId, strategyCode: code, symbol: row.symbol,
        timeframe: row.timeframe, mode: 'paper', strategyId: row.templateId, params: row.parameters,
      });
      if (!resp.success) throw new Error(resp.error || 'StartStrategy failed');
      message.success(t('strategy.live.runStarted', { defaultValue: 'Run started' }));
    } catch (e: unknown) {
      message.error(e instanceof Error ? e.message : t('strategy.live.runStartFailed', { defaultValue: 'Failed to start' }));
    }
  }, [t]);

  const onEdit = useCallback((row: ScheduleRow) => {
    if (row.templateId) navigate(`/strategy/${row.templateId}/edit`);
    else navigate('/strategy');
  }, [navigate]);

  const [editParamsRow, setEditParamsRow] = useState<ScheduleRow | null>(null);

  const onDelete = useCallback(async (row: ScheduleRow) => {
    try {
      await strategyScheduleV2Api.delete(row.id);
      message.success(t('common.deleted', { defaultValue: 'Deleted' }));
    } catch (e: unknown) {
      message.error(e instanceof Error ? e.message : t('common.deleteFailed', { defaultValue: 'Delete failed' }));
    }
  }, [t]);

  const loadScheduleHealth = useCallback(async (row: ScheduleRow) => {
    if (!row?.id) return;
    setHealthLoading(true);
    try { setHealthSummary(await scheduleHealthApi.getScheduleHealth(row.id) as ScheduleHealthSummary); }
    catch { setHealthSummary(null); } finally { setHealthLoading(false); }
  }, []);

  const onHealthCheck = useCallback((row: ScheduleRow) => {
    setHealthTarget(row);
    void loadScheduleHealth(row);
    setHealthOpen(true);
  }, [loadScheduleHealth]);

  useEffect(() => {
    if (!healthId || !schedules.length) return;
    const target = schedules.find(s => s.id === healthId);
    if (target) onHealthCheck(target);
  }, [healthId, schedules, onHealthCheck]);

  return (
    <div style={{ padding: '0 0 12px' }}>
      <h2 style={{ margin: '0 0 12px', fontSize: 18, fontWeight: 600 }}>{t('strategy.live.title', { defaultValue: 'Live Strategy Monitor' })}</h2>

      {streamError && activeTab === 'strategies' && (
        <Alert type="warning" showIcon style={{ marginBottom: 12 }}
          message={t('strategy.live.streamDisconnected', { defaultValue: 'Connection interrupted, reconnecting…' })} />
      )}

      <Tabs activeKey={activeTab} onChange={setActiveTab} items={[
        {
          key: 'strategies',
          label: <span><ClockCircleOutlined /> {t('strategy.live.myStrategies', { defaultValue: 'My Strategies' })}</span>,
          children: (
            <Card size="small">
              <MyStrategiesTable
                schedules={joinedRows} orphanRuns={orphanRuns} templates={templates}
                accounts={scheduleAccounts} loading={loading} activeVersion={activeVersion}
                highlightScheduleId={highlightScheduleId}
                onToggleActive={onToggleActive} onManualTrigger={onManualTrigger}
                onEdit={onEdit} onEditParams={(row) => setEditParamsRow(row)} onDelete={onDelete}
                onShowLogs={(id) => setLogsModalScheduleId(id)}
                onHealthCheck={onHealthCheck} onStop={handleStop} stopping={stopping}
              />
            </Card>
          ),
        },
        {
          key: 'history',
          label: <span><EyeOutlined /> {t('strategy.live.historyTab', { defaultValue: 'Run History' })}</span>,
          children: <RunHistoryTab />,
        },
      ]} />
      <EditParamsModal
        open={editParamsRow !== null}
        schedule={editParamsRow}
        accounts={scheduleAccounts}
        onClose={() => setEditParamsRow(null)}
        onUpdated={() => setActiveVersion(v => v + 1)}
      />

      <ScheduleLogsModal open={logsModalScheduleId !== null} scheduleId={logsModalScheduleId}
        onClose={() => setLogsModalScheduleId(null)} />
      <ScheduleHealthModal open={healthOpen} loading={healthLoading}
        target={healthTarget as unknown as Record<string, unknown> | null}
        summary={healthSummary as unknown as Record<string, unknown> | null}
        onRefresh={() => { if (healthTarget) loadScheduleHealth(healthTarget); }}
        onClose={() => { setHealthOpen(false); setHealthTarget(null); setHealthSummary(null); }}
        formatTime={(v: unknown) => formatTime(v as { seconds?: bigint; nanos?: number } | null)} />
    </div>
  );
}
