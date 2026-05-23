import dayjs from 'dayjs';
import { message } from 'antd';
import type { TFunction } from 'i18next';
import { strategyApi } from '@/client/strategy';
import { pickMetric } from './StrategyTemplatePage.utils';

// 把这两个"计算 + 副作用"辅助函数从 StrategyTemplatePage.tsx 抽出来，
// 单纯是为了让主文件保持在 800 行内。它们不是独立 hook —— 依然由主组件持有
// 状态 setters，这里只是纯函数壳子。

export const buildScheduleName = (run: any, t: TFunction): string => {
	const symbol = String(run?.symbol || '').trim();
	const timeframe = String(run?.timeframe || '').trim();
	const title = String(run?.title || '').trim();
	if (title) return title;
	const nowText = dayjs().format('YYYY-MM-DD HH:mm');
	return String(t('strategy.templates.scheduleName', { symbol, timeframe, nowText })).trim();
};

// 0-100 的综合评分：回撤/夏普/胜率/收益的加权平均。
export const computeScoreValue = (metrics: any | null | undefined): number | undefined => {
	if (!metrics) return undefined;
	const totalReturn = pickMetric(metrics, ['totalReturn', 'total_return']);
	const maxDrawdown = pickMetric(metrics, ['maxDrawdown', 'max_drawdown']);
	const sharpe = pickMetric(metrics, ['sharpeRatio', 'sharpe_ratio']);
	const winRate = pickMetric(metrics, ['winRate', 'win_rate']);
	const clamp01 = (x: number) => (x < 0 ? 0 : x > 1 ? 1 : x);
	const ddScore = maxDrawdown === undefined ? 0.5 : clamp01(1 - maxDrawdown / 0.3);
	const srScore = sharpe === undefined ? 0.5 : clamp01((sharpe + 0.5) / 3);
	const wrScore = winRate === undefined ? 0.5 : clamp01((winRate - 0.3) / 0.5);
	const retScore = totalReturn === undefined ? 0.5 : clamp01((totalReturn + 0.2) / 0.6);
	return Math.round((ddScore * 0.35 + srScore * 0.25 + wrScore * 0.2 + retScore * 0.2) * 100);
};

export type CreateScheduleFromRunDeps = {
	t: TFunction;
	setScheduleFlow: React.Dispatch<React.SetStateAction<any>>;
	fetchTemplates: () => void | Promise<void>;
	fetchRuns: () => void | Promise<void>;
	onClose: () => void;
};

export type CreateScheduleFromRunParams = {
	templateId: string;
	run: any;
	enableAfterCreate: boolean;
	form?: {
		accountId: string;
		symbol: string;
		timeframe: string;
		scheduleType: 'interval' | 'kline_close' | 'hf_quote';
		intervalMs: number;
		hfCooldownMs: number;
	};
	parameters?: Record<string, string>;
};

// createScheduleFromRun 会：
//   1) 根据 run / form 凑出 accountId / symbol / timeframe / schedule_config
//   2) 验证模板已发布
//   3) 调后端 createSchedule，若 enableAfterCreate=true 再调一次 toggle
//   4) 成功后关闭弹窗并刷新模板/回测列表
//
// 所有 state 依然由调用方持有，这里只负责流程编排。
export async function createScheduleFromRun(
	params: CreateScheduleFromRunParams,
	deps: CreateScheduleFromRunDeps,
): Promise<void> {
	const { t, setScheduleFlow, fetchTemplates, fetchRuns, onClose } = deps;

	const accountId = String(
		params.form?.accountId || params.run?.accountId || params.run?.account_id || '',
	).trim();
	const symbol = String(params.form?.symbol || params.run?.symbol || '').trim();
	const timeframe = String(params.form?.timeframe || params.run?.timeframe || '').trim();
	const scheduleType = params.form?.scheduleType || 'kline_close';
	const scheduleConfig: any = {};
	if (scheduleType === 'interval') {
		scheduleConfig.intervalMs = BigInt(params.form?.intervalMs || 300_000);
	} else if (scheduleType === 'hf_quote') {
		scheduleConfig.triggerMode = 'hf_quote_stream';
		scheduleConfig.hfCooldownMs = BigInt(params.form?.hfCooldownMs || 1_000);
	} else if (scheduleType === 'kline_close') {
		scheduleConfig.triggerMode = 'stable_kline';
	}
	// sanitize parameters to a plain string->string map to avoid JSON encoding issues
	const parameters: Record<string, string> = Object.fromEntries(
		Object.entries(params.parameters || {}).map(([k, v]) => [String(k), String(v as any)]),
	);
	// derive schedule name; prefer user-provided name
	const customName = String(parameters['__schedule.name'] || '').trim();
	const fallbackName = buildScheduleName(params.run, t);
	const scheduleName = customName || String(fallbackName);

	if (!params.templateId || !accountId || !symbol || !timeframe) {
		message.error(String(t('strategy.templates.messages.missingScheduleInfo')));
		return;
	}

	try {
		const tpl: any = await strategyApi.getTemplate(String(params.templateId || '').trim());
		const status = String(tpl?.status || '').trim().toLowerCase();
		if (status !== 'published') {
			message.warning(String(t('strategy.templates.messages.templateNotPublishedCannotCreateSchedule')));
			return;
		}
	} catch (_e) {
		message.warning(String(t('strategy.templates.messages.readTemplateStatusFailed')));
		return;
	}

	setScheduleFlow((p: any) => ({ ...p, creating: true }));
	try {
		const resp: any = await strategyApi.createSchedule({
			templateId: params.templateId,
			accountId,
			name: scheduleName,
			symbol,
			timeframe,
			parameters,
			scheduleType,
			scheduleConfig,
		} as any);
		const scheduleId = String(resp?.id || resp?.schedule?.id || '').trim();
		if (scheduleId && params.enableAfterCreate) {
			await strategyApi.toggleSchedule(scheduleId, true);
		}
		message.success(
			String(
				params.enableAfterCreate
					? t('strategy.templates.messages.scheduleCreatedAndEnabled')
					: t('strategy.templates.messages.scheduleCreated'),
			),
		);
		void fetchTemplates();
		void fetchRuns();
		onClose();
	} catch (e: any) {
		message.error(String(e?.message || e || t('strategy.templates.messages.createScheduleFailed')));
	} finally {
		setScheduleFlow((p: any) => ({ ...p, creating: false }));
	}
}
