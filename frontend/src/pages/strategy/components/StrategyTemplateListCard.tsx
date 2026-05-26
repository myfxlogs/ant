import { Badge, Card, Segmented, Space, Table, Typography } from "antd";
import type { ColumnsType } from "antd/es/table";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import type { StrategyTemplate } from "@/client/strategy";
import { StatusResult } from "@/components/common/StatusResult";

const { Text } = Typography;

type TemplateGroup = "system" | "user";

type Props = {
  dataSource: StrategyTemplate[];
  templatesCount: number;
  templateGroup: TemplateGroup;
  loading: boolean;
  error?: string | null;
  onRetry?: () => void;
  columns: ColumnsType<StrategyTemplate>;
  highlightTemplateId: string;
  onTemplateGroupChange: (value: TemplateGroup) => void;
};

export default function StrategyTemplateListCard({
  dataSource,
  templatesCount,
  templateGroup,
  loading,
  error,
  onRetry,
  columns,
  highlightTemplateId,
  onTemplateGroupChange,
}: Props) {
  const { t } = useTranslation();
  // Split into system (preset) vs user-created templates. 权威字段是 isSystem，
  // 兼容后端没返回时回落到 tags.preset 或 id 前缀 default-*。
  const { systemTemplates, userTemplates } = useMemo(() => {
    const system: StrategyTemplate[] = [];
    const user: StrategyTemplate[] = [];
    for (const tpl of dataSource || []) {
      const rawTags = tpl?.tags;
      const tags = Array.isArray(rawTags) ? rawTags : [];
      const isSystem =
        Boolean(tpl?.isSystem) ||
        tags.includes("preset") ||
        String(tpl?.id || "").startsWith("default-");
      (isSystem ? system : user).push(tpl);
    }
    return { systemTemplates: system, userTemplates: user };
  }, [dataSource]);
  const activeTemplates =
    templateGroup === "system" ? systemTemplates : userTemplates;
  return (
    <Card
      title={
        <TemplateTabs
          t={(key: string, fallback?: string) => String(t(key, { defaultValue: fallback }))}
          group={templateGroup}
          systemCount={systemTemplates.length}
          userCount={userTemplates.length}
          onChange={onTemplateGroupChange}
        />
      }
    >
      <StatusResult error={error} onRetry={onRetry}>
        <Table
          columns={columns}
          dataSource={activeTemplates}
          rowKey="id"
          loading={loading}
          scroll={{ x: "max-content" }}
          rowClassName={(record) =>
            highlightTemplateId && record.id === highlightTemplateId
              ? "bg-amber-50"
              : ""
          }
          locale={{
            emptyText:
              templatesCount === 0
                ? t("strategy.templates.table.loadingDefault")
                : templateGroup === "user"
                  ? t(
                      "strategy.templates.table.emptyUser",
                    )
                  : t("common.noData"),
          }}
          pagination={{
            defaultPageSize: 10,
            pageSizeOptions: ["10", "20", "50"],
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (total) => t("common.totalItems", { total }),
          }}
        />
        {templatesCount === 0 && templateGroup === "system" && (
          <div
            style={{ textAlign: "center", padding: "16px 0", color: "#8A9AA5" }}
          >
            <Text type="secondary">
              {t("strategy.templates.table.defaultHint")}
            </Text>
          </div>
        )}
      </StatusResult>
    </Card>
  );
}

function TemplateTabs({
  t,
  group,
  systemCount,
  userCount,
  onChange,
}: {
  t: (key: string, fallback?: string) => string;
  group: TemplateGroup;
  systemCount: number;
  userCount: number;
  onChange: (value: TemplateGroup) => void;
}) {
  return (
    <Segmented
      value={group}
      onChange={(v) => onChange(v as TemplateGroup)}
      options={[
        {
          label: (
            <TabLabel
              text={t("strategy.templates.tabs.system")}
              count={systemCount}
              active={group === "system"}
            />
          ),
          value: "system",
        },
        {
          label: (
            <TabLabel
              text={t("strategy.templates.tabs.user")}
              count={userCount}
              active={group === "user"}
            />
          ),
          value: "user",
        },
      ]}
    />
  );
}

function TabLabel({
  text,
  count,
  active,
}: {
  text: string;
  count: number;
  active: boolean;
}) {
  const color = active ? "#1890ff" : "#bfbfbf";
  return (
    <Space size={6}>
      <span>{text}</span>
      <Badge
        count={count}
        showZero
        color={color}
        style={{ backgroundColor: color }}
      />
    </Space>
  );
}
