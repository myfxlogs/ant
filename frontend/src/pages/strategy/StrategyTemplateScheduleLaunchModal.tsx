import React, { useEffect, useState } from 'react';
import { Button, Modal, Space } from 'antd';
import { useTranslation } from 'react-i18next'
import { SCHEDULE_LAUNCH_ACTIONS_PUBLISH_TEMPLATE_KEY, SCHEDULE_LAUNCH_BACKTEST_RUNNING_HINT_KEY, SCHEDULE_LAUNCH_KEY_METRICS_KEY, SCHEDULE_LAUNCH_LAUNCH_SECTION_KEY, SCHEDULE_LAUNCH_METRICS_ANNUAL_RETURN_KEY, SCHEDULE_LAUNCH_METRICS_MAX_DRAWDOWN_KEY, SCHEDULE_LAUNCH_METRICS_SHARPE_KEY, SCHEDULE_LAUNCH_METRICS_TOTAL_RETURN_KEY, SCHEDULE_LAUNCH_METRICS_TOTAL_TRADES_KEY, SCHEDULE_LAUNCH_METRICS_WIN_RATE_KEY, SCHEDULE_LAUNCH_NO_RUN_KEY, SCHEDULE_LAUNCH_SCORE_KEY, SCHEDULE_LAUNCH_TITLE_KEY } from '@/gen/ant/v1/i18n/strategy_templates_keys';

;
import { codeAssistApi, type RequiredParamSpec } from '@/client/codeAssist';
import { strategyApi } from '@/client/strategy';
import { DEFAULT_TIMEFRAME } from '@/constants/timeframes';
import { isTerminalRun, pickMetric } from './StrategyTemplatePage.utils';
import {
	StrategyTemplateScheduleLaunchForm,
	type ScheduleLaunchFormValues,
	type SubmitParams,
} from './StrategyTemplateScheduleLaunchForm';
import { useSchedulePublish, formatPercent, formatFloat, formatInt } from './useSchedulePublish';

export type ScheduleFlowState = {
	templateId?: string;
	templateDraftId?: string;
	publishing: boolean;
	creating: boolean;
	enableAfterCreate: boolean;
};

// onCreateSchedule 现在接收完整的表单输出（而不是只有 enableAfterCreate）。
// 上层 StrategyTemplatePage.createScheduleFromRun 据此构造 CreateScheduleRequest。
export type CreateScheduleInput = {
	templateId: string;
	run: { metrics?: Record<string, unknown>; templateId?: string; templateDraftId?: string; status?: unknown };
	form: ScheduleLaunchFormValues;
	parameters: Record<string, string>;
};

export type StrategyTemplateScheduleLaunchModalProps = {
	open: boolean;
	scoreLoading: boolean;
	scoreRunId: string;
	scoreSnapshot: { run: any | null; metrics: any | null } | null;
	scoreValue: number | undefined;
	scheduleFlow: ScheduleFlowState;
	setScheduleFlow: React.Dispatch<React.SetStateAction<ScheduleFlowState>>;
	setRuns: React.Dispatch<React.SetStateAction<any[]>>;
	accounts: any[];
	symbols: { value: string; label: string }[];
	symbolsLoading: boolean;
	onAccountChange?: (accountId: string) => void | Promise<void>;
	onCancel: () => void;
	onCreateSchedule: (params: CreateScheduleInput) => void;
	allowWithoutRun?: boolean;
};

export const StrategyTemplateScheduleLaunchModal: React.FC<StrategyTemplateScheduleLaunchModalProps> = ({
	open,
	scoreLoading,
	scoreRunId,
	scoreSnapshot,
	scoreValue,
	scheduleFlow,
	setScheduleFlow,
	setRuns,
	accounts,
	symbols,
	symbolsLoading,
	onAccountChange,
	onCancel,
	onCreateSchedule,
	allowWithoutRun,
}) => {
	const { t } = useTranslation();
	const hasRun = Boolean(scoreSnapshot?.run);
	const canBypassRun = Boolean(allowWithoutRun);
	const [requiredParams, setRequiredParams] = useState<RequiredParamSpec[]>([]);
	const [paramValues, setParamValues] = useState<Record<string, unknown>>({});

	useEffect(() => {
		if (!open || !scheduleFlow.templateId) {
			setRequiredParams([]);
			setParamValues({});
			return;
		}
		let cancelled = false;
		setRequiredParams([]);
		setParamValues({});
		void (async () => {
			try {
				const tpl: any = await strategyApi.getTemplate(String(scheduleFlow.templateId || ''));
				const code = String(tpl?.code || '');
				if (!code.trim()) return;
				const ext = await codeAssistApi.validateExtended(code);
				if (!cancelled && ext.valid) {
					setRequiredParams(ext.parameters || []);
				}
			} catch {
				if (!cancelled) {
					setRequiredParams([]);
				}
			}
		})();
		return () => {
			cancelled = true;
		};
	}, [open, scheduleFlow.templateId]);

	const { handlePublishTemplate } = useSchedulePublish({
		scheduleFlow,
		setScheduleFlow,
		setRuns,
		scoreRunId,
		t,
	});

	const handleFormSubmit = (params: SubmitParams) => {
		onCreateSchedule({
			templateId: String(scheduleFlow.templateId || ''),
			run: scoreSnapshot?.run,
			form: params.form,
			parameters: params.buildParameters(),
		});
	};

	const run = scoreSnapshot?.run;
	const formDefaults: Partial<ScheduleLaunchFormValues> = hasRun
		? {
				accountId: String(run?.accountId || run?.account_id || ''),
				symbol: String(run?.symbol || ''),
				timeframe: String(run?.timeframe || DEFAULT_TIMEFRAME),
				scheduleType: 'kline_close',
				intervalMs: 300_000,
				enableAfterCreate: true,
		  }
		: {};
	const isTerminal = isTerminalRun(scoreSnapshot?.run);
	const formDisabled = !hasRun && canBypassRun ? false : !isTerminal;

	return (
		<Modal
			title={t(SCHEDULE_LAUNCH_TRADING_TITLE_KEY)}
			open={open}
			onCancel={onCancel}
			footer={null}
			width={720}
			destroyOnClose
		>
			{scoreLoading ? (
				<div className="text-sm text-gray-500">{t('common.loading')}</div>
			) : !hasRun && !canBypassRun ? (
				<div className="text-sm text-gray-500">{t(SCHEDULE_LAUNCH_NO_RUN_KEY)}</div>
			) : (
				<div>
					{hasRun ? (
						<>
							<div className="text-sm text-gray-700">
								{t(SCHEDULE_LAUNCH_SCORE_KEY)}
							</div>
							<div className="text-2xl font-semibold mt-1">
								{typeof scoreValue === 'number' ? `${scoreValue}%` : '-'}
							</div>
							<div className="mt-3 text-sm text-gray-700">
								{t(SCHEDULE_LAUNCH_KEY_METRICS_KEY)}
							</div>
							<div className="mt-2 text-sm text-gray-600 whitespace-pre-wrap">
								{t(SCHEDULE_LAUNCH_METRICS_TOTAL_RETURN_KEY)}:{' '}
								{formatPercent(pickMetric(scoreSnapshot.metrics, ['totalReturn', 'total_return']))}
								{'\n'}
								{t(SCHEDULE_LAUNCH_METRICS_ANNUAL_RETURN_KEY)}:{' '}
								{formatPercent(pickMetric(scoreSnapshot.metrics, ['annualReturn', 'annual_return']))}
								{'\n'}
								{t(SCHEDULE_LAUNCH_METRICS_MAX_DRAWDOWN_KEY)}:{' '}
								{formatPercent(pickMetric(scoreSnapshot.metrics, ['maxDrawdown', 'max_drawdown']))}
								{'\n'}
								{t(SCHEDULE_LAUNCH_METRICS_SHARPE_KEY)}:{' '}
								{formatFloat(pickMetric(scoreSnapshot.metrics, ['sharpeRatio', 'sharpe_ratio']))}
								{'\n'}
								{t(SCHEDULE_LAUNCH_METRICS_WIN_RATE_KEY)}:{' '}
								{formatPercent(pickMetric(scoreSnapshot.metrics, ['winRate', 'win_rate']))}
								{'\n'}
								{t(SCHEDULE_LAUNCH_METRICS_TOTAL_TRADES_KEY)}:{' '}
								{formatInt(pickMetric(scoreSnapshot.metrics, ['totalTrades', 'total_trades']))}
							</div>
						</>
					) : null}

					<div className="mt-5 border-t pt-4">
						<div className="text-sm text-gray-700 mb-2">
							{t(SCHEDULE_LAUNCH_LAUNCH_SECTION_KEY)}
						</div>
						{hasRun && !isTerminal ? (
							<div className="text-xs text-gray-500 mb-2">
								{t(SCHEDULE_LAUNCH_BACKTEST_RUNNING_HINT_KEY)}
							</div>
						) : null}

						{!scheduleFlow.templateId && scheduleFlow.templateDraftId ? (
							<Space direction="vertical" style={{ width: '100%' }}>
								<Button
									type="primary"
									block
									disabled={!isTerminal}
									loading={scheduleFlow.publishing}
									onClick={() => void handlePublishTemplate()}
								>
									{t(SCHEDULE_LAUNCH_ACTIONS_PUBLISH_TEMPLATE_KEY)}
								</Button>
							</Space>
						) : null}

						{scheduleFlow.templateId ? (
							<StrategyTemplateScheduleLaunchForm
								open={open}
								accounts={accounts}
								symbols={symbols}
								symbolsLoading={symbolsLoading}
								onAccountChange={onAccountChange}
								defaults={formDefaults}
								requiredParams={requiredParams}
								paramValues={paramValues}
								onParamValuesChange={setParamValues}
								submitting={scheduleFlow.creating}
								disabled={formDisabled}
								onSubmit={handleFormSubmit}
							/>
						) : null}
					</div>
				</div>
			)}
		</Modal>
	);
};
