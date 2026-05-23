import { scheduleHealthClient } from "./connect";

export const scheduleHealthApi = {
  getScheduleHealth: async (scheduleId: string) => {
    const response = await scheduleHealthClient.getScheduleHealth({
      scheduleId,
      runLimit: 30,
      orderLimit: 20,
    });
    const summary = response.summary;
    return {
      totalRuns: summary?.totalRuns || 0,
      successRuns: summary?.successRuns || 0,
      failedRuns: summary?.failedRuns || 0,
      successRate: summary?.successRate || 0,
      lastRunAt: summary?.lastRunAt,
      latestError: summary?.latestError || "",
      latestOrderTicket: summary?.latestOrderTicket || "-",
      latestOrderProfit: summary?.hasLatestOrderProfit
        ? summary.latestOrderProfit
        : null,
      gradeLevel: summary?.gradeLevel || "unknown",
      gradeColor: summary?.gradeColor || "default",
      gradeNoteCode: summary?.gradeNoteCode || "pending",
      greenSuccessRate: summary?.greenSuccessRate || 90,
      greenMaxFailedRuns: summary?.greenMaxFailedRuns || 1,
      yellowSuccessRate: summary?.yellowSuccessRate || 60,
      minSampleSize: summary?.minSampleSize || 1,
      runLogs: response.runLogs || [],
      orders: response.orders || [],
    };
  },
};
