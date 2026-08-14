import {
  Badge,
  Button,
  Popconfirm,
  Space,
  Switch,
  Table,
  Tag,
  Tooltip,
  Typography,
} from "antd";
import type { ColumnsType } from "antd/es/table";
import { useMemo } from "react";
import { useTranslation } from "react-i18next"
import { FORMAT_CRON_KEY, FORMAT_INTERVAL_KEY } from '@/gen/ant/v1/i18n/strategy_schedules_keys';
import type { ScheduleRow, TemplateOption, AccountRow, TriggerContext } from '../hooks/libraryTypes';

const { Text } = Typography;

type Props = {
  schedules: ScheduleRow[];
  templates: TemplateOption[];
  accounts: AccountRow[];
  loading: boolean;
  triggering: boolean;
  triggerContext: TriggerContext | null;
  formatTime: (v: unknown) => string;
  onEdit: (row: ScheduleRow) => void;
  onToggleActive: (row: ScheduleRow, next: boolean) => void;
  onHealthCheck: (row: ScheduleRow) => void;
  onManualTrigger: (row: ScheduleRow) => void;
  onDelete: (row: ScheduleRow) => void;
  onShowLogs: (row: ScheduleRow) => void;
  highlightScheduleId?: string | null;
};

export default function ScheduleTable({
  schedules,
  templates,
  accounts,
  loading,
  triggering,
  triggerContext,
  formatTime,
  onEdit,
  onToggleActive,
  onHealthCheck,
  onManualTrigger,
  onDelete,
  onShowLogs,
  highlightScheduleId,
}: Props) {
  const { t } = useTranslation();
  const templateById = useMemo(() => {
    const m = new Map<string, TemplateOption>();
    (templates || []).forEach((item) => {
      if (item?.id) m.set(item.id, item);
    });
    return m;
  }, [templates]);
  const accountById = useMemo(() => {
    const m = new Map<string, AccountRow>();
    (accounts || []).forEach((item) => {
      if (item?.id) m.set(item.id, item);
    });
    return m;
  }, [accounts]);
  const columns: ColumnsType<ScheduleRow> = [
    {
      title: t("strategy.schedules.table.name"),
      dataIndex: "name",
      key: "name",
      render: (v: React.ReactNode, row: ScheduleRow) => (
        <Space orientation="vertical" size={0}>
          <Text strong>{v}</Text>
          <Text type="secondary" style={{ fontSize: 12 }}>
            {row?.id}
          </Text>
        </Space>
      ),
    },
    {
      title: t("strategy.schedules.table.template"),
      dataIndex: "templateId",
      key: "templateId",
      render: (id: string) => {
        const tpl = templateById.get(id);
        return (
          <Space orientation="vertical" size={0}>
            <Text>{tpl?.name || id}</Text>
            {tpl?.isPublic ? (
              <Tag color="blue">
                {t("strategy.schedules.templateVisibility.public")}
              </Tag>
            ) : (
              <Tag>{t("strategy.schedules.templateVisibility.private")}</Tag>
            )}
          </Space>
        );
      },
    },
    {
      title: t("strategy.schedules.table.account"),
      dataIndex: "accountId",
      key: "accountId",
      render: (id: string) => {
        const account = accountById.get(id);
        return (
          <Text>
            {account?.login ? `${account.login} (${account.mtType || ""})` : id}
          </Text>
        );
      },
    },
    {
      title: t("strategy.schedules.table.tradeParams"),
      key: "trade",
      render: (_: unknown, row: ScheduleRow) => (
        <Space orientation="vertical" size={0}>
          <Text>{row?.symbol}</Text>
          <Text type="secondary" style={{ fontSize: 12 }}>
            {row?.timeframe}
          </Text>
        </Space>
      ),
    },
    {
      title: t("strategy.schedules.table.schedule"),
      key: "schedule",
      render: (_: unknown, row: ScheduleRow) => <Text>{formatSchedule(t, row)}</Text>,
    },
    {
      title: t("strategy.schedules.table.status"),
      key: "status",
      render: (_: unknown, row: ScheduleRow) => (
        <Space orientation="vertical" size={0}>
          <Space>
            <Switch
              checked={!!row?.isActive}
              onChange={(v) => onToggleActive(row, v)}
              disabled={loading}
            />
            {row?.isActive ? (
              <Tag color="blue">{t("strategy.schedules.status.enabled", { defaultValue: "Enabled" })}</Tag>
            ) : (
              <Tag>{t("strategy.schedules.status.disabled")}</Tag>
            )}
          </Space>
          <Space>
            <Badge status={row?.isRunning ? "success" : "default"} />
            <Text style={{ fontSize: 12 }}>
              {row?.isRunning
                ? t("strategy.schedules.status.running", { defaultValue: "Running" })
                : t("strategy.schedules.status.idle", { defaultValue: "Idle" })}
            </Text>
            {typeof row?.signalCount === "number" && row.signalCount > 0 && (
              <Text type="secondary" style={{ fontSize: 12 }}>({row.signalCount})</Text>
            )}
          </Space>
          <Text type="secondary" style={{ fontSize: 12 }}>
            {t("strategy.schedules.nextRunAt")}: {formatTime(row?.nextRunAt)}
          </Text>
          {row?.lastError && (
            <Tooltip title={row.lastError}>
              <Text type="danger" style={{ fontSize: 11 }}>⚠ {row.lastError.slice(0, 40)}</Text>
            </Tooltip>
          )}
        </Space>
      ),
    },
    {
      title: t("strategy.schedules.table.lastRun"),
      key: "lastRunAt",
      render: (_: unknown, row: ScheduleRow) => (
        <Space orientation="vertical" size={0}>
          <Text>{formatTime(row?.lastRunAt)}</Text>
          <Text type="secondary" style={{ fontSize: 12 }}>
            {t("strategy.schedules.enableCount")}: {" "}
            {typeof row?.enableCount === "number" ? row.enableCount : "-"}
          </Text>
        </Space>
      ),
    },
    {
      title: t("strategy.schedules.table.actions"),
      key: "actions",
      render: (_: unknown, row: ScheduleRow) => (
        <Space>
          <Button size="small" onClick={() => onEdit(row)} disabled={loading}>
            {t("common.edit")}
          </Button>
          <Button
            size="small"
            onClick={() => onShowLogs(row)}
            disabled={loading}
          >
            {t("strategy.schedules.actions.logs")}
          </Button>
          <Button
            size="small"
            onClick={() => onHealthCheck(row)}
            disabled={loading}
          >
            {t("strategy.schedules.actions.healthCheck")}
          </Button>
          <Button
            size="small"
            onClick={() => onManualTrigger(row)}
            loading={triggering && triggerContext?.schedule?.id === row.id}
          >
            {t("strategy.schedules.actions.runNow")}
          </Button>
          <Popconfirm
            title={t("strategy.schedules.deleteConfirm.title")}
            okText={t("common.delete")}
            cancelText={t("common.cancel")}
            onConfirm={() => onDelete(row)}
          >
            <Button size="small" danger disabled={loading}>
              {t("common.delete")}
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];
  return (
    <Table
      scroll={{ x: "max-content" }}
      rowKey="id"
      loading={loading}
      dataSource={schedules}
      columns={columns}
      pagination={{ pageSize: 10 }}
      rowClassName={(row) => row.id === highlightScheduleId ? 'schedule-row-highlight' : ''}
    />
  );
}

function formatSchedule(t: (key: string, opts?: Record<string, unknown>) => string, row: ScheduleRow) {
  const conf = row?.scheduleConfig || {};
  if (row?.scheduleType === "interval") {
    const raw = conf?.intervalMs;
    const ms =
      typeof raw === "number"
        ? raw
        : typeof raw === "bigint"
          ? Number(raw)
          : undefined;
    if (typeof ms === "number" && Number.isFinite(ms) && ms > 0) {
      const s = Math.max(1, Math.floor(ms / 1000));
      return t(FORMAT_INTERVAL_KEY, { s } as Record<string, unknown>);
    }
    return "-";
  }
  const cron = String(conf?.cronExpression || "").trim();
  return cron ? t(FORMAT_CRON_KEY, { expr: cron } as Record<string, unknown>) : "-";
}
