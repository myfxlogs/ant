import React, { useEffect, useMemo, useState } from 'react';
import { Button, Modal, Space, Tag, Typography } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';
import { useTranslation } from 'react-i18next'
import { ACTIONS_CANCEL_KEY, STATUS_CANCELED_KEY, STATUS_CANCELING_KEY, STATUS_COMPLETED_KEY, STATUS_ENDED_KEY, STATUS_FAILED_KEY, STATUS_QUEUED_KEY, STATUS_RUNNING_KEY, TITLE_KEY, TRADES_CLOSE_PRICE_KEY, TRADES_CLOSE_TIME_KEY, TRADES_COMMISSION_KEY, TRADES_OPEN_PRICE_KEY, TRADES_OPEN_TIME_KEY, TRADES_PNL_KEY, TRADES_REASON_KEY, TRADES_SIDE_BUY_KEY, TRADES_SIDE_KEY, TRADES_SIDE_SELL_KEY, TRADES_SUMMARY_KEY, TRADES_TICKET_KEY, TRADES_VOLUME_KEY } from '@/gen/ant/v1/i18n/strategy_backtest_run_keys';
;

import { useWatchBacktestRun } from '@/hooks/useWatchBacktestRun';
import { backtestRunsApi, type BacktestTrade, type BacktestTradeSummary } from '@/client/backtestRuns';
import { isSucceededRun } from '@/pages/strategy/StrategyTemplatePage.utils';
import BacktestRunDrawerContent from './BacktestRunDrawerContent';

type Props = {
	open: boolean;
	runId: string;
	onClose: () => void;
	onCancel: () => void;
	canceling?: boolean;
};

const fmt = (n: number | null | undefined, digits = 4): string =>
	n === null || n === undefined || Number.isNaN(n) ? '-' : Number(n).toFixed(digits);

const fmtTs = (ms: number | undefined): string =>
	!ms || ms <= 0 ? '-' : dayjs(ms).format('YYYY-MM-DD HH:mm:ss');

const BacktestRunDrawer: React.FC<Props> = ({ open, runId, onClose, onCancel, canceling }) => {
	const { t } = useTranslation();
	const watched = useWatchBacktestRun(runId || null);
	const [trades, setTrades] = useState<BacktestTrade[]>([]);
	const [tradeSummary, setTradeSummary] = useState<BacktestTradeSummary | null>(null);
	const [tradesLoading, setTradesLoading] = useState(false);
	const [tradesError, setTradesError] = useState<string | null>(null);

	const isCompleted = isSucceededRun(watched.run);

	// 抽屉打开 + 当前 run 已完成时再拉成交记录。原先在 effect 头部
	// 同步 setTrades([])/setTradesError(null) 触发 react-hooks/set-state-in-effect；
	// 改为「不满足条件时直接不进入 fetch 分支」，渲染时用下方 visible* gate
	// 屏蔽旧数据，避免上一个 runId 的成交闪现到新一次抽屉打开。
	useEffect(() => {
		if (!open || !runId || !isCompleted) return;
		let cancelled = false;
		// 启动一次新 fetch 前的合法 loading/错误重置；规则误报，按实际语义白名单。
		// eslint-disable-next-line react-hooks/set-state-in-effect
		setTradesLoading(true);
		setTradesError(null);
		backtestRunsApi
			.getTrades(runId)
			.then((result) => {
				if (cancelled) return;
				setTrades(result.trades);
				setTradeSummary(result.summary);
			})
			.catch((e: unknown) => {
				if (cancelled) return;
				setTradesError(e instanceof Error ? e.message : String(e));
			})
			.finally(() => {
				if (cancelled) return;
				setTradesLoading(false);
			});
		return () => {
			cancelled = true;
		};
	}, [open, runId, isCompleted]);

	const tradesActive = open && !!runId && isCompleted;
	const visibleTrades = tradesActive ? trades : [];
	const visibleTradeSummary = tradesActive ? tradeSummary : null;
	const visibleTradesError = tradesActive ? tradesError : null;

	const statusText = (() => {
		switch (watched.run?.status) {
			case 1:
				return t(STATUS_QUEUED_KEY);
			case 2:
				return t(STATUS_RUNNING_KEY);
			case 3:
				return t(STATUS_COMPLETED_KEY);
			case 4:
				return t(STATUS_FAILED_KEY);
			case 5:
				return t(STATUS_CANCELING_KEY);
			case 6:
				return t(STATUS_CANCELED_KEY);
			default:
				return watched.run?.status != null ? String(watched.run.status) : '-';
		}
	})();

	const summary = useMemo(() => {
		if (!visibleTradeSummary || !visibleTradeSummary.count) return null;
		return t(TRADES_SUMMARY_KEY, {
			count: visibleTradeSummary.count,
			wins: visibleTradeSummary.wins,
			losses: visibleTradeSummary.losses,
			pnl: (visibleTradeSummary.netPnl ?? 0).toFixed(2),
		});
	}, [visibleTradeSummary, t]);

	const columns = useMemo<ColumnsType<BacktestTrade>>(
		() => [
			{ title: t(TRADES_TICKET_KEY), dataIndex: 'ticket', key: 'ticket', width: 70 },
			{
				title: t(TRADES_SIDE_KEY),
				dataIndex: 'side',
				key: 'side',
				width: 70,
				render: (v: string) => {
					const isBuy = String(v).toLowerCase() === 'buy';
					return (
						<Tag color={isBuy ? 'green' : 'red'}>
							{isBuy ? t(TRADES_SIDE_BUY_KEY) : t(TRADES_SIDE_SELL_KEY)}
						</Tag>
					);
				},
			},
			{
				title: t(TRADES_VOLUME_KEY),
				dataIndex: 'volume',
				key: 'volume',
				width: 80,
				render: (v: number) => fmt(v, 2),
			},
			{
				title: t(TRADES_OPEN_TIME_KEY),
				dataIndex: 'open_ts',
				key: 'open_ts',
				render: (v: number) => fmtTs(v),
			},
			{
				title: t(TRADES_OPEN_PRICE_KEY),
				dataIndex: 'open_price',
				key: 'open_price',
				width: 100,
				render: (v: number) => fmt(v, 5),
			},
			{
				title: t(TRADES_CLOSE_TIME_KEY),
				dataIndex: 'close_ts',
				key: 'close_ts',
				render: (v: number) => fmtTs(v),
			},
			{
				title: t(TRADES_CLOSE_PRICE_KEY),
				dataIndex: 'close_price',
				key: 'close_price',
				width: 100,
				render: (v: number) => fmt(v, 5),
			},
			{
				title: t(TRADES_PNL_KEY),
				dataIndex: 'pnl',
				key: 'pnl',
				width: 100,
				align: 'right',
				render: (v: number) => (
					<Typography.Text type={v > 0 ? 'success' : v < 0 ? 'danger' : undefined}>{fmt(v, 2)}</Typography.Text>
				),
				sorter: (a, b) => a.pnl - b.pnl,
			},
			{
				title: t(TRADES_COMMISSION_KEY),
				dataIndex: 'commission',
				key: 'commission',
				width: 100,
				render: (v: number) => fmt(v, 2),
			},
			{
				title: t(TRADES_REASON_KEY),
				dataIndex: 'reason',
				key: 'reason',
				width: 110,
				render: (v: string) => t(`strategy.backtestRun.trades.reasons.${v}`, { defaultValue: v || '-' }),
			},
		],
		[t],
	);

	return (
		<Modal
			title={t(TITLE_KEY)}
			open={open}
			onCancel={onClose}
			destroyOnClose
			width={1100}
			styles={{ body: { maxHeight: 'calc(100vh - 200px)', overflowY: 'auto' } }}
			footer={
				<Space>
					{watched.isTerminal ? (
						<Button disabled>{statusText || t(STATUS_ENDED_KEY)}</Button>
					) : (
						<Button onClick={onCancel} loading={!!canceling} disabled={!runId || watched.isTerminal}>
							{t(ACTIONS_CANCEL_KEY)}
						</Button>
					)}
					<Button type="primary" onClick={onClose}>{t('common.close', { defaultValue: 'Close' })}</Button>
				</Space>
			}
		>
			<BacktestRunDrawerContent
				watched={watched}
				statusText={statusText}
				trades={visibleTrades}
				summary={summary}
				tradesLoading={tradesLoading}
				tradesError={visibleTradesError}
				columns={columns}
			/>
		</Modal>
	);
}

export default BacktestRunDrawer;
