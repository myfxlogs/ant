import BacktestRunsCard from "../BacktestRunsCard";
import { pythonStrategyApi } from "@/client/pythonStrategy";
import { getRunTemplateRef } from "../StrategyTemplatePage.utils";

type Props = {
  runs: any[];
  loading: boolean;
  onRefresh: () => void;
  actions: {
    setSelectedRunId: (id: string) => void;
    setRunDrawerOpen: (open: boolean) => void;
    setScoreRunId: (id: string) => void;
    setScoreOpen: (open: boolean) => void;
    setScoreLoading: (loading: boolean) => void;
    setScoreSnapshot: (
      snapshot: { run: any | null; metrics: any | null } | null,
    ) => void;
    setScheduleFlow: (updater: any) => void;
    setRuns: (updater: any) => void;
  };
  onDelete: (runId: string) => void;
};

export default function StrategyTemplateBacktestRunsPanel({
  runs,
  loading,
  onRefresh,
  actions,
  onDelete,
}: Props) {
  const onViewScore = async (runId: string) => {
    actions.setScoreRunId(String(runId || ""));
    actions.setScoreOpen(true);
    actions.setScoreLoading(true);
    actions.setScoreSnapshot(null);
    try {
      const snapshot: any = await pythonStrategyApi.getBacktestRun(
        String(runId || ""),
      );
      actions.setScoreSnapshot({
        run: snapshot?.run ?? null,
        metrics: snapshot?.metrics ?? null,
      });
      const runObj = snapshot?.run ?? null;
      if (runObj) {
        const ref = getRunTemplateRef(runObj);
        actions.setScheduleFlow((p: any) => ({
          ...p,
          templateId: ref.templateId,
          templateDraftId: ref.templateDraftId,
          enableAfterCreate: true,
        }));
        actions.setRuns((prev: any[]) =>
          (prev || []).map((it: any) =>
            String(it?.id || "") === String(runId || "")
              ? { ...it, ...ref }
              : it,
          ),
        );
      }
    } catch (_e) {
      actions.setScoreSnapshot({ run: null, metrics: null });
    } finally {
      actions.setScoreLoading(false);
    }
  };
  return (
    <BacktestRunsCard
      runs={runs}
      loading={loading}
      onRefresh={onRefresh}
      onView={(runId) => {
        actions.setSelectedRunId(runId);
        actions.setRunDrawerOpen(true);
      }}
      onViewScore={onViewScore}
      onDelete={onDelete}
    />
  );
}
