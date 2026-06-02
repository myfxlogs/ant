import { Button, Card, Space, Typography } from "antd";
import { StatusResult } from "@/components/common/StatusResult";
import EditScheduleModal from "./components/EditScheduleModal";
import TriggerModal from "./components/TriggerModal";
import ScheduleHealthModal from "./components/ScheduleHealthModal";
import ScheduleTable from "./components/ScheduleTable";
import { useTranslation } from "react-i18next";
import { useStrategySchedulePage } from "./useStrategySchedulePage";
const { Title } = Typography;

export default function StrategySchedulePage() {
  const { t } = useTranslation();
  const ctx = useStrategySchedulePage();

  return (
    <div style={{ padding: 24 }}>
      <Space orientation="vertical" size="large" style={{ width: "100%" }}>
        <div className="flex justify-between items-center">
          <Title level={4} style={{ margin: 0 }}>{t("strategy.schedules.title")}</Title>
          <Button type="primary" onClick={ctx.openCreate}>{t("strategy.schedules.createSchedule")}</Button>
        </div>

        <Card>
          <StatusResult error={ctx.error} onRetry={ctx.refresh}>
            <ScheduleTable
              schedules={ctx.schedules}
              loading={ctx.loading}
              formatTime={ctx.formatTime}
              onRefresh={ctx.refresh}
              onOpenEdit={ctx.openUpdate}
              onToggleActive={ctx.onToggleActive}
              onDelete={ctx.onDelete}
              onManualTrigger={ctx.onManualTrigger}
              onHealthCheck={(row) => { ctx.setHealthTarget(row); ctx.loadScheduleHealth(row); ctx.setHealthOpen(true); }}
            />
          </StatusResult>
        </Card>

        <EditScheduleModal
          editing={ctx.editing}
          open={ctx.openEdit}
          loading={ctx.loading}
          form={ctx.form}
          templates={ctx.templatesForSelect}
          accounts={ctx.accounts}
          symbols={ctx.symbols}
          symbolsLoading={ctx.symbolsLoading}
          accountIdWatch={ctx.accountIdWatch}
          onCancel={() => { ctx.setOpenEdit(false); ctx.setEditing(null); ctx.form.resetFields(); }}
          onOk={ctx.submitEdit}
        />

        <TriggerModal
          open={ctx.openTrigger}
          triggering={ctx.triggering}
          result={ctx.triggerResult}
          context={ctx.triggerContext}
          onCancel={() => { ctx.setOpenTrigger(false); ctx.setTriggerContext(null); ctx.setTriggerResult(null); }}
          onOrderSend={ctx.doOrderSend}
        />

        <ScheduleHealthModal
          open={ctx.healthOpen}
          loading={ctx.healthLoading}
          target={ctx.healthTarget}
          summary={ctx.healthSummary}
          onClose={() => { ctx.setHealthOpen(false); ctx.setHealthTarget(null); ctx.setHealthSummary(null); }}
        />
      </Space>
    </div>
  );
}
