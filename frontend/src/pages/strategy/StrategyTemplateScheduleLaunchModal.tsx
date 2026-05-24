import React, { useEffect, useState } from 'react';
import { Button, Modal, Space, message } from 'antd';
import { useTranslation } from 'react-i18next';
import { codeAssistApi, type RequiredParamSpec } from '@/client/codeAssist';
import { strategyApi } from '@/client/strategy';
import { isTerminalRun, pickMetric, isErrTemplateNotDraft } from './StrategyTemplatePage.utils';
import {
	StrategyTemplateScheduleLaunchForm,
	type ScheduleLaunchFormValues,
	type SubmitParams,
} from './StrategyTemplateScheduleLaunchForm';

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
	run: any;
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

// 指标格式化工具：
//   - 比率类（total_return / annual_return / max_drawdown / win_rate）→ 百分比（2 位小数）
//   - Sharpe → 保留 3 位小数
//   - 计数类（total_trades）→ 整数
// 源值可能是 number | string | undefined，兼容处理。
function formatPercent(v: any): string {
	if (v === undefined || v === null || v === '') return '-';
	const n = typeof v === 'number' ? v : Number(v);
	if (!Number.isFinite(n)) return String(v);
	return `${(n * 100).toFixed(2)}%`;
}
function formatFloat(v: any, digits = 3): string {
	if (v === undefined || v === null || v === '') return '-';
	const n = typeof v === 'number' ? v : Number(v);
	if (!Number.isFinite(n)) return String(v);
	return n.toFixed(digits);
}
function formatInt(v: any): string {
	if (v === undefined || v === null || v === '') return '-';
	const n = typeof v === 'number' ? v : Number(v);
	if (!Number.isFinite(n)) return String(v);
	return String(Math.round(n));
}

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

	const handlePublishTemplate = async () => {
		if (!isTerminalRun(scoreSnapshot?.run)) {
			message.warning(t('strategy.templates.messages.backtestRunningCannotPublish'));
			return;
		}
		const draftId = String(scheduleFlow.templateDraftId || '').trim();
		if (!draftId) {
			message.error(t('strategy.templates.messages.missingDraftIdCannotPublish'));
			return;
		}
		setScheduleFlow((p) => ({ ...p, publishing: true }));
		try {
			const draft: any = await strategyApi.getTemplate(draftId);
			const draftCode = String(draft?.code || '').trim();
			if (!draftCode) {
				message.error(t('strategy.templates.messages.strategyCodeEmptyCannotPublish'));
				return;
			}
			const resp: any = await strategyApi.publishTemplateDraft(draftId);
			const tid = String(resp?.id || resp?.template?.id || resp?.templateId || '').trim();
			if (!tid) {
				message.warning(t('strategy.templates.messages.publishedButNoTemplateId'));
				return;
			}
			setScheduleFlow((p) => ({ ...p, templateId: tid }));
			setRuns((prev) =>
				(prev || []).map((it: any) =>
					String(it?.id || '') === String(scoreRunId || '') ? { ...it, templateId: tid } : it,
				),
			);
			message.success(t('strategy.templates.messages.templatePublished'));
		} catch (e: any) {
			if (isErrTemplateNotDraft(e)) {
				try {
					const tpl: any = await strategyApi.getTemplate(draftId);
					const status = String(tpl?.status || '').trim().toLowerCase();
					if (status !== 'published') {
						const baseName =
							String(tpl?.name || t('strategy.templates.defaultDraftName')).trim() ||
							t('strategy.templates.defaultDraftName');
						const newDraft: any = await strategyApi.createTemplateDraft({ name: baseName });
						const newId = String(newDraft?.id || '').trim();
						if (!newId) {
							message.warning(t('strategy.templates.messages.cannotPublishAndCreateDraftFailed'));
							return;
						}
						const codeToPublish = String(tpl?.code || '').trim();
						if (!codeToPublish) {
							message.error(t('strategy.templates.messages.strategyCodeEmptyCannotPublish'));
							return;
						}
						await strategyApi.updateTemplateDraft({
							id: newId,
							name: String(tpl?.name || '').trim() || baseName,
							description: String(tpl?.description || '').trim(),
							code: codeToPublish,
							parameters: Array.isArray(tpl?.parameters) ? tpl.parameters : [],
							tags: Array.isArray(tpl?.tags) ? tpl.tags : [],
						});
						const pub: any = await strategyApi.publishTemplateDraft(newId);
						const tid = String(pub?.id || pub?.template?.id || pub?.templateId || '').trim();
						if (!tid) {
							message.warning(t('strategy.templates.messages.republishedButNoTemplateId'));
							return;
						}
						setScheduleFlow((p) => ({ ...p, templateId: tid }));
						setRuns((prev) =>
							(prev || []).map((it: any) =>
								String(it?.id || '') === String(scoreRunId || '')
									? { ...it, templateId: tid }
									: it,
							),
						);
						message.success(t('strategy.templates.messages.templateRepublished'));
						return;
					}
					setScheduleFlow((p) => ({ ...p, templateId: draftId }));
					setRuns((prev) =>
						(prev || []).map((it: any) =>
							String(it?.id || '') === String(scoreRunId || '')
								? { ...it, templateId: draftId }
								: it,
						),
					);
					message.info(t('strategy.templates.messages.templateAlreadyPublished'));
					return;
				} catch (_e2) {
					message.warning(t('strategy.templates.messages.templateNotDraftUnknownPublishStatus'));
					return;
				}
			}
			const errMsg =
				String(
					e?.rawMessage ||
						(e?.code !== undefined ? `code=${String(e.code)} ` : '') + (e?.message || '') ||
						e,
				) || t('strategy.templates.messages.publishFailed');
			message.error(errMsg);
		} finally {
			setScheduleFlow((p) => ({ ...p, publishing: false }));
		}
	};

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
				accountId: String((run as any)?.accountId || (run as any)?.account_id || ''),
				symbol: String((run as any)?.symbol || ''),
				timeframe: String((run as any)?.timeframe || 'H1'),
				scheduleType: 'kline_close',
				intervalMs: 300_000,
				enableAfterCreate: true,
		  }
		: {};
	const isTerminal = isTerminalRun(scoreSnapshot?.run);
	const formDisabled = !hasRun && canBypassRun ? false : !isTerminal;

	return (
		<Modal
			title={t('strategy.templates.scheduleLaunch.title')}
			open={open}
			onCancel={onCancel}
			footer={null}
			width={720}
			destroyOnClose
		>
			{scoreLoading ? (
				<div className="text-sm text-gray-500">{t('common.loading')}</div>
			) : !hasRun && !canBypassRun ? (
				<div className="text-sm text-gray-500">{t('strategy.templates.scheduleLaunch.noRun')}</div>
			) : (
				<div>
					{hasRun ? (
						<>
							<div className="text-sm text-gray-700">
								{t('strategy.templates.scheduleLaunch.score')}
							</div>
							<div className="text-2xl font-semibold mt-1">
								{typeof scoreValue === 'number' ? `${scoreValue}%` : '-'}
							</div>
							<div className="mt-3 text-sm text-gray-700">
								{t('strategy.templates.scheduleLaunch.keyMetrics')}
							</div>
							<div className="mt-2 text-sm text-gray-600 whitespace-pre-wrap">
								{t('strategy.templates.scheduleLaunch.metrics.totalReturn')}:{' '}
								{formatPercent(pickMetric(scoreSnapshot.metrics, ['totalReturn', 'total_return']))}
								{'\n'}
								{t('strategy.templates.scheduleLaunch.metrics.annualReturn')}:{' '}
								{formatPercent(pickMetric(scoreSnapshot.metrics, ['annualReturn', 'annual_return']))}
								{'\n'}
								{t('strategy.templates.scheduleLaunch.metrics.maxDrawdown')}:{' '}
								{formatPercent(pickMetric(scoreSnapshot.metrics, ['maxDrawdown', 'max_drawdown']))}
								{'\n'}
								{t('strategy.templates.scheduleLaunch.metrics.sharpe')}:{' '}
								{formatFloat(pickMetric(scoreSnapshot.metrics, ['sharpeRatio', 'sharpe_ratio']))}
								{'\n'}
								{t('strategy.templates.scheduleLaunch.metrics.winRate')}:{' '}
								{formatPercent(pickMetric(scoreSnapshot.metrics, ['winRate', 'win_rate']))}
								{'\n'}
								{t('strategy.templates.scheduleLaunch.metrics.totalTrades')}:{' '}
								{formatInt(pickMetric(scoreSnapshot.metrics, ['totalTrades', 'total_trades']))}
							</div>
						</>
					) : null}

					<div className="mt-5 border-t pt-4">
						<div className="text-sm text-gray-700 mb-2">
							{t('strategy.templates.scheduleLaunch.launchSection')}
						</div>
						{hasRun && !isTerminal ? (
							<div className="text-xs text-gray-500 mb-2">
								{t('strategy.templates.scheduleLaunch.backtestRunningHint')}
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
									{t('strategy.templates.scheduleLaunch.actions.publishTemplate')}
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
