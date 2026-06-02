import React from "react";
import { Space } from "antd";
import StrategyTemplateBacktestRunsPanel from "./components/StrategyTemplateBacktestRunsPanel";
import StrategyTemplateDialogs from "./components/StrategyTemplateDialogs";
import StrategyTemplateHeaderCard from "./components/StrategyTemplateHeaderCard";
import StrategyTemplateListCard from "./components/StrategyTemplateListCard";
import { useStrategyTemplatePage } from "./useStrategyTemplatePage";
import type { StrategyTemplate } from "@/client/strategy";

const StrategyTemplatePage: React.FC = () => {
  const ctx = useStrategyTemplatePage();

  return (
    <div style={{ padding: 24 }}>
      <Space orientation="vertical" size="large" style={{ width: "100%" }}>
        <StrategyTemplateHeaderCard onRefresh={ctx.fetchTemplates} onCreate={ctx.handleCreate} />

        <StrategyTemplateListCard
          dataSource={ctx.dataSource as StrategyTemplate[]}
          templatesCount={ctx.templates.length}
          templateGroup={ctx.templateGroup}
          loading={ctx.loading}
          error={ctx.error}
          onRetry={ctx.fetchTemplates}
          columns={ctx.columns}
          highlightTemplateId={ctx.highlightTemplateId}
          onTemplateGroupChange={ctx.setTemplateGroup}
        />

        <StrategyTemplateBacktestRunsPanel
          runs={ctx.runs}
          loading={ctx.runsLoading}
          onRefresh={ctx.fetchRuns}
          actions={ctx.dialogProps.runPanelActions}
          onDelete={ctx.deleteRun}
        />

        <StrategyTemplateDialogs
          edit={ctx.dialogProps.edit}
          schedule={ctx.dialogProps.schedule}
          code={ctx.dialogProps.code}
          backtest={ctx.dialogProps.backtest}
          drawer={ctx.dialogProps.drawer}
        />
      </Space>
    </div>
  );
};

export default StrategyTemplatePage;
