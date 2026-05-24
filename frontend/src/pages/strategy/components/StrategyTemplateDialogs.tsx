import { lazy, Suspense } from "react";
import { useTranslation } from "react-i18next";
import { StrategyTemplateBacktestModal } from "../StrategyTemplateBacktestModal";
import { StrategyTemplateScheduleLaunchModal } from "../StrategyTemplateScheduleLaunchModal";
import {
  StrategyTemplateCodeViewModal,
  StrategyTemplateEditModal,
} from "../StrategyTemplateEditModal";
import * as scheduleHelpers from "../StrategyTemplatePage.schedule";

const BacktestRunDrawer = lazy(
  () => import("@/components/strategy/BacktestRunDrawer"),
);

type Props = {
  edit: any;
  schedule: any;
  code: any;
  backtest: any;
  drawer: any;
};

export default function StrategyTemplateDialogs(props: Props) {
  const { t } = useTranslation();
  return (
    <>
      <StrategyTemplateEditModal
        open={props.edit.modalVisible}
        editingTemplate={props.edit.editingTemplate}
        form={props.edit.form}
        codeValidating={props.edit.codeValidating}
        lastValidatedCode={props.edit.lastValidatedCode}
        onCancel={() => {
          props.edit.setModalVisible(false);
          props.edit.setLastValidatedCode("");
        }}
        onValidate={() => void props.edit.validateTemplateCode()}
        onSubmit={props.edit.handleSave}
      />
      <StrategyTemplateScheduleLaunchModal
        open={props.schedule.scoreOpen}
        allowWithoutRun
        scoreLoading={props.schedule.scoreLoading}
        scoreRunId={props.schedule.scoreRunId}
        scoreSnapshot={props.schedule.scoreSnapshot}
        scoreValue={props.schedule.scoreValue}
        scheduleFlow={props.schedule.scheduleFlow}
        setScheduleFlow={props.schedule.setScheduleFlow}
        setRuns={props.schedule.setRuns}
        accounts={props.schedule.accounts}
        symbols={props.schedule.symbols}
        symbolsLoading={props.schedule.symbolsLoading}
        onAccountChange={props.schedule.loadSymbols}
        onCancel={() => {
          props.schedule.setScoreOpen(false);
          props.schedule.setScoreRunId("");
          props.schedule.setScoreSnapshot(null);
          props.schedule.setScheduleFlow((p: any) => ({
            ...p,
            publishing: false,
            creating: false,
          }));
        }}
        onCreateSchedule={(input) =>
          void scheduleHelpers.createScheduleFromRun(
            {
              templateId: input.templateId,
              run: input.run,
              enableAfterCreate: input.form?.enableAfterCreate ?? true,
              form: input.form,
              parameters: input.parameters,
            },
            {
              t,
              setScheduleFlow: props.schedule.setScheduleFlow,
              fetchTemplates: props.schedule.fetchTemplates,
              fetchRuns: props.schedule.fetchRuns,
              onClose: () => props.schedule.setScoreOpen(false),
            },
          )
        }
      />
      <StrategyTemplateCodeViewModal
        open={props.code.codeModalVisible}
        code={props.code.viewingCode}
        onClose={() => props.code.setCodeModalVisible(false)}
        onCopy={props.code.handleCopyCode}
      />
      <StrategyTemplateBacktestModal
        open={props.backtest.backtestModalOpen}
        template={props.backtest.backtestTemplate}
        form={props.backtest.backtestForm}
        submitting={props.backtest.backtestSubmitting}
        accounts={props.backtest.accounts}
        symbols={props.backtest.symbols}
        symbolsLoading={props.backtest.symbolsLoading}
        quickRange={props.backtest.quickRange}
        watchedRange={props.backtest.watchedRange}
        requiredParams={props.backtest.backtestRequiredParams}
        paramValues={props.backtest.backtestParamValues}
        onParamValuesChange={props.backtest.setBacktestParamValues}
        onCancel={() => {
          props.backtest.setBacktestModalOpen(false);
          props.backtest.setBacktestRequiredParams([]);
          props.backtest.setBacktestParamValues({});
        }}
        onSubmit={props.backtest.submitBacktest}
        onApplyQuickRange={props.backtest.applyQuickRange}
        onSetQuickRange={props.backtest.setQuickRange}
        onAccountChange={props.backtest.loadSymbols}
      />
      <Suspense fallback={null}>
        <BacktestRunDrawer
          open={props.drawer.runDrawerOpen}
          runId={props.drawer.selectedRunId}
          onClose={() => props.drawer.setRunDrawerOpen(false)}
          onCancel={props.drawer.cancelRun}
          canceling={props.drawer.canceling}
        />
      </Suspense>
    </>
  );
}
