import { scheduleHealthClient } from "./connect";
import { create } from "@bufbuild/protobuf";
import { ScheduleHealthSummarySchema, type GetScheduleHealthResponse, type ScheduleHealthSummary } from "@/gen/ant/v1/schedule_health_pb";

function num<T>(v: T | undefined | null, fallback: number): number {
  return (v ?? fallback) as number;
}

function str<T>(v: T | undefined | null, fallback: string): string {
  return (v ?? fallback) as string;
}

export const scheduleHealthApi = {
  getScheduleHealth: async (scheduleId: string) => {
    const response = await scheduleHealthClient.getScheduleHealth({
      scheduleId,
      runLimit: 30,
      orderLimit: 20,
    });
    return mapScheduleHealthResponse(response);
  },
};

function mapScheduleHealthResponse(response: GetScheduleHealthResponse) {
  const s: ScheduleHealthSummary = response.summary ?? create(ScheduleHealthSummarySchema);
  return {
    totalRuns: num(s.totalRuns, 0),
    successRuns: num(s.successRuns, 0),
    failedRuns: num(s.failedRuns, 0),
    successRate: num(s.successRate, 0),
    lastRunAt: s.lastRunAt,
    latestError: str(s.latestError, ""),
    latestOrderTicket: str(s.latestOrderTicket, "-"),
    latestOrderProfit: s.hasLatestOrderProfit ? s.latestOrderProfit : null,
    gradeLevel: str(s.gradeLevel, "unknown"),
    gradeColor: str(s.gradeColor, "default"),
    gradeNoteCode: str(s.gradeNoteCode, "pending"),
    greenSuccessRate: num(s.greenSuccessRate, 90),
    greenMaxFailedRuns: num(s.greenMaxFailedRuns, 1),
    yellowSuccessRate: num(s.yellowSuccessRate, 60),
    minSampleSize: num(s.minSampleSize, 1),
    runLogs: response.runLogs ?? [],
    orders: response.orders ?? [],
  };
}
