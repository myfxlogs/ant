import {
  Alert,
  Button,
  Descriptions,
  Modal,
  Space,
  Table,
  Tag,
  Typography,
} from "antd";
import { useTranslation } from "react-i18next";

const { Text } = Typography;

type Props = {
  open: boolean;
  target: any | null;
  loading: boolean;
  summary: any | null;
  onRefresh: () => void;
  onClose: () => void;
  formatTime: (v: any) => string;
};

const toNumber = (v: any): number => {
  if (typeof v === "number") return v;
  if (typeof v === "bigint") return Number(v);
  const n = Number(v);
  return Number.isFinite(n) ? n : 0;
};

export default function ScheduleHealthModal({
  open,
  target,
  loading,
  summary,
  onRefresh,
  onClose,
  formatTime,
}: Props) {
  const { t } = useTranslation();
  const grade = getGrade(t, summary);
  return (
    <Modal
      title={t("strategy.schedules.health.title", { name: target?.name || "" })}
      open={open}
      onCancel={onClose}
      width={980}
      footer={[
        <Button key="refresh" onClick={onRefresh} loading={loading}>
          {t("common.refresh")}
        </Button>,
        <Button key="close" onClick={onClose}>
          {t("common.close")}
        </Button>,
      ]}
    >
      <Space direction="vertical" style={{ width: "100%" }} size={12}>
        <Alert
          type={
            grade.level === "red"
              ? "error"
              : grade.level === "yellow"
                ? "warning"
                : "success"
          }
          showIcon
          message={
            summary
              ? t("strategy.schedules.health.summaryBanner", {
                  grade: grade.label,
                  totalRuns: summary.totalRuns,
                  successRate: summary.successRate.toFixed(1),
                })
              : t("strategy.schedules.health.messages.clickRefresh")
          }
          description={summary ? grade.note : undefined}
        />
        <Descriptions bordered size="small" column={2}>
          <Descriptions.Item
            label={t("strategy.schedules.health.fields.grade")}
          >
            <Tag color={grade.color}>{grade.label}</Tag>
          </Descriptions.Item>
          <Descriptions.Item label={t("strategy.schedules.health.fields.rule")}>
            {summary ? grade.note : "-"}
          </Descriptions.Item>
          <Descriptions.Item
            label={t("strategy.schedules.health.fields.thresholds")}
          >
            {summary
              ? t("strategy.schedules.health.thresholdsSummary", {
                  minSampleSize: summary.minSampleSize,
                  greenSuccessRate: summary.greenSuccessRate,
                  greenMaxFailedRuns: summary.greenMaxFailedRuns,
                  yellowSuccessRate: summary.yellowSuccessRate,
                })
              : "-"}
          </Descriptions.Item>
          <Descriptions.Item
            label={t("strategy.schedules.health.fields.configKey")}
          >
            <Text code>strategy.schedule.health_grading_config</Text>
          </Descriptions.Item>
          <Descriptions.Item
            label={t("strategy.schedules.health.fields.lastRunAt")}
          >
            {summary ? formatTime(summary.lastRunAt) : "-"}
          </Descriptions.Item>
          <Descriptions.Item
            label={t("strategy.schedules.health.fields.latestTicket")}
          >
            {summary?.latestOrderTicket || "-"}
          </Descriptions.Item>
          <Descriptions.Item
            label={t("strategy.schedules.health.fields.successOverTotal")}
          >
            {summary ? `${summary.successRuns}/${summary.totalRuns}` : "-"}
          </Descriptions.Item>
          <Descriptions.Item
            label={t("strategy.schedules.health.fields.failedRuns")}
          >
            {summary ? summary.failedRuns : "-"}
          </Descriptions.Item>
          <Descriptions.Item
            label={t("strategy.schedules.health.fields.latestProfit")}
          >
            {summary?.latestOrderProfit != null
              ? summary.latestOrderProfit.toFixed(2)
              : "-"}
          </Descriptions.Item>
          <Descriptions.Item
            label={t("strategy.schedules.health.fields.latestError")}
          >
            {summary?.latestError || "-"}
          </Descriptions.Item>
        </Descriptions>

        <Text strong>{t("strategy.schedules.health.sections.runLogs")}</Text>
        <Table
          scroll={{ x: "max-content" }}
          rowKey={(row) => String(row?.id || "")}
          size="small"
          loading={loading}
          pagination={false}
          dataSource={summary?.runLogs || []}
          columns={[
            {
              title: t("strategy.scheduleLogs.execTable.time"),
              key: "createdAt",
              width: 180,
              render: (_: any, row: any) => formatTime(row?.createdAt),
            },
            {
              title: t("strategy.scheduleLogs.execTable.status"),
              dataIndex: "status",
              key: "status",
              width: 120,
            },
            {
              title: t("strategy.schedules.health.runLogs.signalType"),
              dataIndex: "signalType",
              key: "signalType",
              width: 120,
            },
            {
              title: t("strategy.scheduleLogs.execTable.durationMs"),
              dataIndex: "durationMs",
              key: "durationMs",
              width: 110,
              render: (v: any) => toNumber(v),
            },
            {
              title: t("strategy.scheduleLogs.execTable.error"),
              dataIndex: "errorMessage",
              key: "errorMessage",
              render: (v: any) => String(v || "-"),
            },
          ]}
        />

        <Text strong>{t("strategy.schedules.health.sections.orders")}</Text>
        <Table
          scroll={{ x: "max-content" }}
          rowKey={(row) => String(row?.id || row?.ticket || "")}
          size="small"
          loading={loading}
          pagination={false}
          dataSource={summary?.orders || []}
          columns={[
            {
              title: t("strategy.scheduleLogs.ordersTable.time"),
              key: "time",
              width: 180,
              render: (_: any, row: any) =>
                formatTime(row?.closeTime || row?.openTime),
            },
            {
              title: t("strategy.scheduleLogs.ordersTable.ticket"),
              dataIndex: "ticket",
              key: "ticket",
              width: 110,
            },
            {
              title: t("strategy.scheduleLogs.ordersTable.side"),
              dataIndex: "orderType",
              key: "orderType",
              width: 110,
            },
            {
              title: t("strategy.scheduleLogs.ordersTable.symbol"),
              dataIndex: "symbol",
              key: "symbol",
              width: 120,
            },
            {
              title: t("strategy.scheduleLogs.ordersTable.profit"),
              dataIndex: "profit",
              key: "profit",
              width: 100,
              render: (v: any) => toNumber(v).toFixed(2),
            },
          ]}
        />
      </Space>
    </Modal>
  );
}

function getGrade(
  t: (key: string, opts?: Record<string, any>) => string,
  summary: any | null,
) {
  if (!summary)
    return {
      level: "unknown",
      label: t("strategy.schedules.health.grade.pending"),
      color: "default",
      note: t("strategy.schedules.health.notes.pending"),
    };
  const code = String(summary.gradeNoteCode || "pending");
  if (code === "no_sample")
    return {
      level: summary.gradeLevel,
      label: t("strategy.schedules.health.grade.noSample"),
      color: summary.gradeColor,
      note: t("strategy.schedules.health.notes.noSample", {
        minSampleSize: summary.minSampleSize,
      }),
    };
  if (code === "healthy")
    return {
      level: summary.gradeLevel,
      label: t("strategy.schedules.health.grade.healthy"),
      color: summary.gradeColor,
      note: t("strategy.schedules.health.notes.healthy", {
        greenSuccessRate: summary.greenSuccessRate,
        greenMaxFailedRuns: summary.greenMaxFailedRuns,
      }),
    };
  if (code === "watch")
    return {
      level: summary.gradeLevel,
      label: t("strategy.schedules.health.grade.watch"),
      color: summary.gradeColor,
      note: t("strategy.schedules.health.notes.watch", {
        yellowSuccessRate: summary.yellowSuccessRate,
      }),
    };
  return {
    level: summary.gradeLevel,
    label: t("strategy.schedules.health.grade.alert"),
    color: summary.gradeColor,
    note: t("strategy.schedules.health.notes.alert"),
  };
}
