import { message } from 'antd';
import { strategyApi } from '@/client/strategy';
import { strategyRuntimeApi } from '@/client/strategyRuntime';
import type { useWsCode, useWsHistory } from '../../WorkspaceContext';

type CodeCtx = ReturnType<typeof useWsCode>;
type HistoryCtx = ReturnType<typeof useWsHistory>;

export function useSidebarActions(code: CodeCtx, history: HistoryCtx) {
  const onDeleteTemplate = async (id: string) => {
    try { await strategyApi.deleteTemplate(id); message.success('Deleted'); code.loadTemplates(); }
    catch (e) { message.error((e as Error)?.message || 'Delete failed'); }
  };

  const onRenameTemplate = async (id: string, name: string) => {
    try { await strategyApi.updateTemplate({ id, name }); message.success('Renamed'); code.loadTemplates(); }
    catch (e) { message.error((e as Error)?.message || 'Rename failed'); }
  };

  const onBatchDeleteTemplates = async (ids: string[]) => {
    try { await Promise.all(ids.map(id => strategyApi.deleteTemplate(id))); message.success(`Deleted ${ids.length}`); code.loadTemplates(); }
    catch (e) { message.error((e as Error)?.message || 'Delete failed'); }
  };

  const onBatchDeleteRuns = async (runIds: string[]) => {
    try { await strategyRuntimeApi.deleteBacktestRuns(runIds); message.success(`Deleted ${runIds.length}`); history.refresh(); }
    catch (e) { message.error((e as Error)?.message || 'Delete failed'); }
  };

  return { onDeleteTemplate, onRenameTemplate, onBatchDeleteTemplates, onBatchDeleteRuns };
}
