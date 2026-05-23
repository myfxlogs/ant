import {
  Button,
  Popconfirm,
  Space,
  Switch,
  Table,
  Tag,
  Typography,
} from "antd";
import type { ColumnsType } from "antd/es/table";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router-dom";

const { Text } = Typography;

type Props = {
  schedules: any[];
  templates: any[];
  accounts: any[];
  loading: boolean;
  triggering: boolean;
  triggerContext: { schedule: any; accountId: string } | null;
  formatTime: (v: any) => string;
  onEdit: (row: any) => void;
  onToggleActive: (row: any, next: boolean) => void;
  onHealthCheck: (row: any) => void;
  onManualTrigger: (row: any) => void;
  onDelete: (row: any) => void;
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
}: Props) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const templateById = useMemo(() => {
    const m = new Map<string, any>();
    (templates || []).forEach((item: any) => {
      if (item?.id) m.set(item.id, item);
    });
    return m;
  }, [templates]);
  const accountById = useMemo(() => {
    const m = new Map<string, any>();
    (accounts || []).forEach((item: any) => {
      if (item?.id) m.set(item.id, item);
    });
    return m;
  }, [accounts]);
  const columns: ColumnsType<any> = [
    {
      title: t("strategy.schedules.table.name"),
      dataIndex: "name",
      key: "name",
      render: (v: any, row: any) => (
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
      render: (_: any, row: any) => (
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
      render: (_: any, row: any) => <Text>{formatSchedule(row)}</Text>,
    },
    {
      title: t("strategy.schedules.table.status"),
      key: "status",
      render: (_: any, row: any) => (
        <Space orientation="vertical" size={0}>
          <Space>
            <Switch
              checked={!!row?.isActive}
              onChange={(v) => onToggleActive(row, v)}
              disabled={loading}
            />
            {row?.isActive ? (
              <Tag color="green">{t("strategy.schedules.status.running")}</Tag>
            ) : (
              <Tag>{t("strategy.schedules.status.disabled")}</Tag>
            )}
          </Space>
          <Text type="secondary" style={{ fontSize: 12 }}>
            {t("strategy.schedules.nextRunAt")}: {formatTime(row?.nextRunAt)}
          </Text>
        </Space>
      ),
    },
    {
      title: t("strategy.schedules.table.lastRun"),
      key: "lastRunAt",
      render: (_: any, row: any) => (
        <Space orientation="vertical" size={0}>
          <Text>{formatTime(row?.lastRunAt)}</Text>
          <Text type="secondary" style={{ fontSize: 12 }}>
            {t("strategy.schedules.enableCount")}:{" "}
            {typeof row?.enableCount === "number" ? row.enableCount : "-"}
          </Text>
        </Space>
      ),
    },
    {
      title: t("strategy.schedules.table.actions"),
      key: "actions",
      render: (_: any, row: any) => (
        <Space>
          <Button size="small" onClick={() => onEdit(row)} disabled={loading}>
            {t("common.edit")}
          </Button>
          <Button
            size="small"
            onClick={() => navigate(`/strategy/schedules/${row.id}/logs`)}
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
    />
  );
}

function formatSchedule(row: any) {
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
      return `interval: ${Math.max(1, Math.floor(ms / 1000))}s`;
    }
    return "-";
  }
  const cron = String(conf?.cronExpression || "").trim();
  return cron ? `cron: ${cron}` : "-";
}
