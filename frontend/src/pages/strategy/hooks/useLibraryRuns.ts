import { useCallback, useEffect, useRef, useState } from 'react';
import { message } from 'antd';
import { useTranslation } from 'react-i18next'
import { MESSAGES_BACKTEST_REPORT_DELETED_KEY } from '@/gen/ant/v1/i18n/strategy_templates_keys';

;
import { pythonStrategyApi } from '@/client/pythonStrategy';
import { isTerminalRun, loadRunTitles } from '../StrategyTemplatePage.utils';
import type { BacktestRunRow } from './libraryTypes';

export function useLibraryRuns(templateId: string | undefined) {
  const { t } = useTranslation();
  const [runs, setRuns] = useState<BacktestRunRow[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [selectedRunId, setSelectedRunId] = useState('');
  const [deleting, setDeleting] = useState(false);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [total, setTotal] = useState(0);
  const [selectedKeys, setSelectedKeys] = useState<React.Key[]>([]);
  const runStreamUnsubRef = useRef<Record<string, (() => void) | undefined>>({});

  const fetchRuns = useCallback(async (p: number = page, ps: number = pageSize) => {
    setLoading(true); setError(null);
    try {
      const params: any = { limit: ps + 1, offset: (p - 1) * ps };
      if (templateId) params.templateId = templateId;
      const resp: any = await pythonStrategyApi.listBacktestRuns(params);
      const titles = loadRunTitles();
      const rawList: BacktestRunRow[] = (resp?.runs || []).map((r: any) => ({
        ...r,
        title: titles?.[String(r?.id || '')] || '',
        templateId: r.templateId || r.template_id,
        templateDraftId: r.templateDraftId || r.template_draft_id,
      }));
      const hasMore = rawList.length > ps;
      const displayList = hasMore ? rawList.slice(0, ps) : rawList;
      setRuns(displayList);
      setTotal(hasMore ? p * ps + 1 : (p - 1) * ps + displayList.length);
    } catch (e) {
      setRuns([]);
      console.error('fetchLibraryRuns failed', e);
      setError(t('common.loadingFailed'));
    } finally { setLoading(false); }
  }, [page, pageSize, templateId, t]);

  useEffect(() => { fetchRuns(); }, [fetchRuns]);

  const updateRunFromStream = useCallback((u: any) => {
    const id = String(u?.run?.id || u?.run?.runId || u?.runId || '');
    if (!id) return;
    setRuns(prev => prev.map(r => {
      if (String(r.id || '') !== id) return r;
      return { ...r, ...u?.run, status: u?.run?.status ?? r.status, error: u?.run?.error ?? r.error, metrics: u?.metrics ?? r.metrics, equityCurve: Array.isArray(u?.equityCurve) ? u.equityCurve : r.equityCurve };
    }));
    if (isTerminalRun(u?.run)) {
      runStreamUnsubRef.current[id]?.();
      delete runStreamUnsubRef.current[id];
    }
  }, []);

  useEffect(() => {
    for (const r of runs) {
      const id = String(r.id || '');
      if (!id || isTerminalRun(r)) continue;
      if (runStreamUnsubRef.current[id]) continue;
      runStreamUnsubRef.current[id] = pythonStrategyApi.watchBacktestRun(id, updateRunFromStream, () => {});
    }
  }, [runs, updateRunFromStream]);

  useEffect(() => {
    const subs = runStreamUnsubRef.current;
    return () => { for (const id of Object.keys(subs)) { try { subs[id]?.(); } catch {} delete subs[id]; } };
  }, []);

  const onViewRun = useCallback((runId: string) => {
    setSelectedRunId(runId); setDrawerOpen(true);
  }, []);

  const onDeleteRun = useCallback(async (runId: string) => {
    setDeleting(true);
    try {
      await pythonStrategyApi.deleteBacktestRun(runId);
      message.success(t(MESSAGES_BACKTEST_REPORT_DELETED_KEY));
      const newPage = runs.length <= 1 && page > 1 ? page - 1 : page;
      setPage(newPage); fetchRuns(newPage, pageSize);
    } catch (e) { console.error('deleteBacktestRun failed', e); message.error(t('common.deleteFailed')); }
    finally { setDeleting(false); }
  }, [runs.length, page, pageSize, fetchRuns, t]);

  const onBatchDelete = useCallback(async () => {
    if (!selectedKeys.length) return;
    setDeleting(true);
    try {
      await pythonStrategyApi.deleteBacktestRuns(selectedKeys.map(String));
      setSelectedKeys([]);
      const newPage = runs.length <= selectedKeys.length && page > 1 ? page - 1 : page;
      setPage(newPage); fetchRuns(newPage, pageSize);
    } catch (e) { console.error('deleteBacktestRuns failed', e); message.error(t('common.deleteFailed')); }
    finally { setDeleting(false); }
  }, [selectedKeys, runs.length, page, pageSize, fetchRuns, t]);

  const onPageChange = useCallback((p: number, ps: number) => {
    setPage(p); setPageSize(ps); setSelectedKeys([]);
    fetchRuns(p, ps);
  }, [fetchRuns]);

  return {
    runs, loading, error, drawerOpen, setDrawerOpen, selectedRunId,
    deleting, page, pageSize, total, selectedKeys, setSelectedKeys,
    fetchRuns, onViewRun, onDeleteRun, onBatchDelete, onPageChange,
  };
}
