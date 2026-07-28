import { useState, useEffect, useCallback } from 'react';
import type React from 'react';

export interface HistoryState {
  drawerOpen: boolean;
  runId: string;
  modalOpen: boolean;
  runs: unknown[];
  loading: boolean;
  page: number;
  pageSize: number;
  total: number;
  selectedRowKeys: React.Key[];
  deleting: boolean;
  open: (templateId?: string) => void;
  close: () => void;
  closeModal: () => void;
  onPageChange: (p: number, ps: number) => void;
  onViewRun: (runId: string) => void;
  onDeleteRun: (runId: string) => void;
  onBatchDelete: () => void;
  onRefresh: () => void;
  setSelectedRowKeys: (keys: React.Key[]) => void;
}

export function useHistoryState(accountId: string): HistoryState {
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [runId, setRunId] = useState('');
  const [modalOpen, setModalOpen] = useState(false);
  const [runs, setRuns] = useState<unknown[]>([]);
  const [loading, setLoading] = useState(false);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [total, setTotal] = useState(0);
  const [selectedKeys, setSelectedKeys] = useState<React.Key[]>([]);
  const [deleting, setDeleting] = useState(false);

  const fetchRuns = useCallback(async (p: number, ps: number, templateId?: string) => {
    setLoading(true);
    try {
      const { strategyRuntimeApi } = await import('@/client/strategyRuntime');
      const resp = await strategyRuntimeApi.listBacktestRuns({ accountId: accountId || undefined, templateId: templateId || undefined, limit: ps, offset: (p - 1) * ps });
      const r = resp.runs ?? [];
      setRuns(r);
      setTotal(r.length < ps ? (p - 1) * ps + r.length : p * ps + 1);
    } catch (e) {
      setRuns([]);
      console.error('fetchHistoryRuns failed', e);
    } finally { setLoading(false); }
  }, [accountId]);

  const open = useCallback((templateId?: string) => {
    setModalOpen(true);
    setPage(1); setPageSize(20); setSelectedKeys([]);
    fetchRuns(1, 20, templateId);
  }, [fetchRuns]);

  const close = useCallback(() => { setDrawerOpen(false); setRunId(''); }, []);
  const closeModal = useCallback(() => { setModalOpen(false); setSelectedKeys([]); }, []);
  const onViewRun = useCallback((id: string) => { setRunId(id); setDrawerOpen(true); }, []);

  const onPageChange = useCallback((p: number, ps: number) => {
    setPage(p); setPageSize(ps); setSelectedKeys([]);
    fetchRuns(p, ps);
  }, [fetchRuns]);

  const onDeleteRun = useCallback(async (runId: string) => {
    setDeleting(true);
    try {
      const { strategyRuntimeApi } = await import('@/client/strategyRuntime');
      await strategyRuntimeApi.deleteBacktestRun(runId);
      const newPage = runs.length <= 1 && page > 1 ? page - 1 : page;
      setPage(newPage);
      fetchRuns(newPage, pageSize);
    } catch (e) {
      console.error('deleteBacktestRun failed', e);
    } finally { setDeleting(false); }
  }, [runs.length, page, pageSize, fetchRuns]);

  const onBatchDelete = useCallback(async () => {
    if (!selectedKeys.length) return;
    setDeleting(true);
    try {
      const { strategyRuntimeApi } = await import('@/client/strategyRuntime');
      await strategyRuntimeApi.deleteBacktestRuns(selectedKeys.map(String));
      setSelectedKeys([]);
      const newPage = runs.length <= selectedKeys.length && page > 1 ? page - 1 : page;
      setPage(newPage);
      fetchRuns(newPage, pageSize);
    } catch (e) {
      console.error('deleteBacktestRuns failed', e);
    } finally { setDeleting(false); }
  }, [selectedKeys, runs.length, page, pageSize, fetchRuns]);

  useEffect(() => {
    if (modalOpen) { setPage(1); fetchRuns(1, pageSize); }
  }, [accountId]); // eslint-disable-line react-hooks/exhaustive-deps

  return {
    drawerOpen, runId, modalOpen, runs, loading, page, pageSize, total,
    selectedRowKeys: selectedKeys, setSelectedRowKeys: setSelectedKeys,
    deleting, open, close, closeModal, onPageChange, onViewRun, onDeleteRun, onBatchDelete,
    onRefresh: () => fetchRuns(page, pageSize),
  };
}
