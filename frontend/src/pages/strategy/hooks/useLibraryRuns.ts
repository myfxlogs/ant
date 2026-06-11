import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { message } from 'antd';
import { useTranslation } from 'react-i18next';
import { pythonStrategyApi } from '@/client/pythonStrategy';
import { isTerminalRun, loadRunTitles } from '../StrategyTemplatePage.utils';

export function useLibraryRuns() {
  const { t } = useTranslation();
  const [runs, setRuns] = useState<any[]>([]);
  const [loading, setLoading] = useState(false);
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [selectedRunId, setSelectedRunId] = useState('');
  const [deleting, setDeleting] = useState(false);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [total, setTotal] = useState(0);
  const [selectedKeys, setSelectedKeys] = useState<React.Key[]>([]);
  const runStreamUnsubRef = useRef<Record<string, (() => void) | undefined>>({});

  const fetchRuns = useCallback(async (p: number = page, ps: number = pageSize) => {
    setLoading(true);
    try {
      const resp: any = await pythonStrategyApi.listBacktestRuns({ limit: ps, offset: (p - 1) * ps });
      const titles = loadRunTitles();
      const list = (resp?.runs || []).map((r: any) => ({
        ...r,
        title: titles?.[String(r?.id || '')] || '',
        templateId: r.templateId || r.template_id,
        templateDraftId: r.templateDraftId || r.template_draft_id,
      }));
      setRuns(list);
      setTotal(list.length < ps ? (p - 1) * ps + list.length : p * ps + 1); // estimate
    } catch {
      setRuns([]);
    } finally { setLoading(false); }
  }, [page, pageSize]);

  useEffect(() => { fetchRuns(); }, []); // eslint-disable-line react-hooks/exhaustive-deps

  const updateRunFromStream = useCallback((u: any) => {
    const id = String(u?.run?.id || u?.run?.runId || u?.runId || '');
    if (!id) return;
    setRuns(prev => (prev || []).map((r: any) => {
      if (String(r?.id || '') !== id) return r;
      return { ...r, ...u?.run, status: u?.run?.status ?? r?.status, error: u?.run?.error ?? r?.error, metrics: u?.metrics ?? r?.metrics, equityCurve: Array.isArray(u?.equityCurve) ? u.equityCurve : r?.equityCurve };
    }));
    if (isTerminalRun(u?.run)) {
      runStreamUnsubRef.current[id]?.();
      delete runStreamUnsubRef.current[id];
    }
  }, []);

  // SSE streaming for non-terminal runs
  useEffect(() => {
    for (const r of runs || []) {
      const id = String(r?.id || '');
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
      message.success(t('strategy.templates.messages.backtestReportDeleted'));
      const newPage = runs.length <= 1 && page > 1 ? page - 1 : page;
      setPage(newPage); fetchRuns(newPage, pageSize);
    } catch { message.error(t('common.deleteFailed')); }
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
    } catch { message.error(t('common.deleteFailed')); }
    finally { setDeleting(false); }
  }, [selectedKeys, runs.length, page, pageSize, fetchRuns, t]);

  const onPageChange = useCallback((p: number, ps: number) => {
    setPage(p); setPageSize(ps); setSelectedKeys([]);
    fetchRuns(p, ps);
  }, [fetchRuns]);

  // Filter by template if one is selected (but fetch all globally — filter client-side)
  const filteredByTemplate = useCallback((templateId: string) => {
    if (!templateId) return runs;
    return runs.filter(r => String(r.templateId || '') === templateId);
  }, [runs]);

  return {
    runs, loading, drawerOpen, setDrawerOpen, selectedRunId,
    deleting, page, pageSize, total, selectedKeys, setSelectedKeys,
    fetchRuns, onViewRun, onDeleteRun, onBatchDelete, onPageChange,
    filteredByTemplate,
  };
}
