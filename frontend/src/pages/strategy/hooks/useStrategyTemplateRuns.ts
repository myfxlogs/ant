import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { message } from "antd";
import { pythonStrategyApi } from "@/client/pythonStrategy";
import * as scheduleHelpers from "../StrategyTemplatePage.schedule";
import { isTerminalRun, loadRunTitles } from "../StrategyTemplatePage.utils";
import type { ScheduleFlowState } from "../StrategyTemplateScheduleLaunchModal";

export function useStrategyTemplateRuns(t: (key: string) => string) {
  const [runs, setRuns] = useState<any[]>([]);
  const [runsLoading, setRunsLoading] = useState(false);
  const [runDrawerOpen, setRunDrawerOpen] = useState(false);
  const [selectedRunId, setSelectedRunId] = useState("");
  const [canceling, setCanceling] = useState(false);
  const runStreamUnsubRef = useRef<Record<string, (() => void) | undefined>>(
    {},
  );
  const [scoreOpen, setScoreOpen] = useState(false);
  const [scoreLoading, setScoreLoading] = useState(false);
  const [scoreRunId, setScoreRunId] = useState("");
  const [scoreSnapshot, setScoreSnapshot] = useState<{
    run: any | null;
    metrics: any | null;
  } | null>(null);
  const [scheduleFlow, setScheduleFlow] = useState<ScheduleFlowState>({
    publishing: false,
    creating: false,
    enableAfterCreate: true,
  });

  const fetchRuns = useCallback(async () => {
    setRunsLoading(true);
    try {
      const resp: any = await pythonStrategyApi.listBacktestRuns({
        limit: 50,
        offset: 0,
      });
      const titles = loadRunTitles();
      const list = (resp?.runs || []).map((r: any) => ({
        ...r,
        title: titles?.[String(r?.id || "")] || "",
        templateId: r.templateId || r.template_id,
        templateDraftId: r.templateDraftId || r.template_draft_id,
      }));
      setRuns(list);
    } catch (_e) {
      setRuns([]);
    } finally {
      setRunsLoading(false);
    }
  }, []);

  const scoreValue = useMemo(
    () => scheduleHelpers.computeScoreValue(scoreSnapshot?.metrics),
    [scoreSnapshot],
  );

  const updateRunFromStream = useCallback((u: any) => {
    const id = String(u?.run?.id || u?.run?.runId || u?.runId || "");
    if (!id) return;
    setRuns((prev) =>
      (prev || []).map((r: any) => {
        if (String(r?.id || "") !== id) return r;
        return {
          ...r,
          ...u?.run,
          status: u?.run?.status ?? r?.status,
          error: u?.run?.error ?? r?.error,
          metrics: u?.metrics ?? r?.metrics,
          equityCurve: Array.isArray(u?.equityCurve)
            ? u.equityCurve
            : r?.equityCurve,
        };
      }),
    );
    if (isTerminalRun(u?.run)) {
      runStreamUnsubRef.current[id]?.();
      delete runStreamUnsubRef.current[id];
    }
  }, []);

  useEffect(() => {
    for (const r of runs || []) {
      const id = String((r as any)?.id || "");
      if (!id) continue;
      if (isTerminalRun(r)) {
        runStreamUnsubRef.current[id]?.();
        delete runStreamUnsubRef.current[id];
        continue;
      }
      if (runStreamUnsubRef.current[id]) continue;
      runStreamUnsubRef.current[id] = pythonStrategyApi.watchBacktestRun(
        id,
        updateRunFromStream,
        () => {},
      );
    }
  }, [runs, updateRunFromStream]);

  useEffect(() => {
    const subs = runStreamUnsubRef.current;
    return () => {
      for (const [id, unsub] of Object.entries(subs)) {
        try {
          unsub?.();
        } catch {}
        delete subs[id];
      }
    };
  }, []);

  const cancelRun = async () => {
    if (!selectedRunId) return;
    setCanceling(true);
    try {
      await pythonStrategyApi.cancelBacktestRun(selectedRunId);
      message.success(t("strategy.templates.messages.backtestCancelRequested"));
      await fetchRuns();
    } catch (_e) {
      message.error(t("strategy.templates.messages.backtestCancelFailed"));
    } finally {
      setCanceling(false);
    }
  };

  const deleteRun = async (runId: string) => {
    if (!runId) return;
    try {
      const resp: any = await pythonStrategyApi.deleteBacktestRun(runId);
      if (resp?.deleted)
        message.success(t("strategy.templates.messages.backtestReportDeleted"));
      else
        message.warning(
          t("strategy.templates.messages.backtestReportNotFound"),
        );
      await fetchRuns();
    } catch (_e) {
      message.error(t("common.deleteFailed"));
    }
  };

  return {
    runs,
    setRuns,
    runsLoading,
    fetchRuns,
    runDrawerOpen,
    setRunDrawerOpen,
    selectedRunId,
    setSelectedRunId,
    canceling,
    cancelRun,
    deleteRun,
    scoreOpen,
    setScoreOpen,
    scoreLoading,
    setScoreLoading,
    scoreRunId,
    setScoreRunId,
    scoreSnapshot,
    setScoreSnapshot,
    scoreValue,
    scheduleFlow,
    setScheduleFlow,
  };
}
