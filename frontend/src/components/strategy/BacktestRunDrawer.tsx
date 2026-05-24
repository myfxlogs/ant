import React, { useEffect, useMemo, useState } from 'react';
import { Alert, Button, Descriptions, Modal, Space, Table, Tag, Typography } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';
import { useTranslation } from 'react-i18next';

import { useWatchBacktestRun } from '@/hooks/useWatchBacktestRun';
import { backtestRunsApi, type BacktestTrade, type BacktestTradeSummary } from '@/client/backtestRuns';
import { isSucceededRun } from '@/pages/strategy/StrategyTemplatePage.utils';

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
			.catch((e: any) => {
				if (cancelled) return;
				setTradesError(e?.message || String(e));
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
				return t('strategy.backtestRun.status.queued');
			case 2:
				return t('strategy.backtestRun.status.running');
			case 3:
				return t('strategy.backtestRun.status.completed');
			case 4:
				return t('strategy.backtestRun.status.failed');
			case 5:
				return t('strategy.backtestRun.status.canceling');
			case 6:
				return t('strategy.backtestRun.status.canceled');
			default:
				return watched.run?.status != null ? String(watched.run.status) : '-';
		}
	})();

	const summary = useMemo(() => {
		if (!visibleTradeSummary || !visibleTradeSummary.count) return null;
		return t('strategy.backtestRun.trades.summary', {
			count: visibleTradeSummary.count,
			wins: visibleTradeSummary.wins,
			losses: visibleTradeSummary.losses,
			pnl: visibleTradeSummary.netPnl.toFixed(2),
		});
	}, [visibleTradeSummary, t]);

	const columns = useMemo<ColumnsType<BacktestTrade>>(
		() => [
			{ title: t('strategy.backtestRun.trades.ticket'), dataIndex: 'ticket', key: 'ticket', width: 70 },
			{
				title: t('strategy.backtestRun.trades.side'),
				dataIndex: 'side',
				key: 'side',
				width: 70,
				render: (v: string) => {
					const isBuy = String(v).toLowerCase() === 'buy';
					return (
						<Tag color={isBuy ? 'green' : 'red'}>
							{isBuy ? t('strategy.backtestRun.trades.sideBuy') : t('strategy.backtestRun.trades.sideSell')}
						</Tag>
					);
				},
			},
			{
				title: t('strategy.backtestRun.trades.volume'),
				dataIndex: 'volume',
				key: 'volume',
				width: 80,
				render: (v: number) => fmt(v, 2),
			},
			{
				title: t('strategy.backtestRun.trades.openTime'),
				dataIndex: 'open_ts',
				key: 'open_ts',
				render: (v: number) => fmtTs(v),
			},
			{
				title: t('strategy.backtestRun.trades.openPrice'),
				dataIndex: 'open_price',
				key: 'open_price',
				width: 100,
				render: (v: number) => fmt(v, 5),
			},
			{
				title: t('strategy.backtestRun.trades.closeTime'),
				dataIndex: 'close_ts',
				key: 'close_ts',
				render: (v: number) => fmtTs(v),
			},
			{
				title: t('strategy.backtestRun.trades.closePrice'),
				dataIndex: 'close_price',
				key: 'close_price',
				width: 100,
				render: (v: number) => fmt(v, 5),
			},
			{
				title: t('strategy.backtestRun.trades.pnl'),
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
				title: t('strategy.backtestRun.trades.commission'),
				dataIndex: 'commission',
				key: 'commission',
				width: 100,
				render: (v: number) => fmt(v, 2),
			},
			{
				title: t('strategy.backtestRun.trades.reason'),
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
			title={t('strategy.backtestRun.title')}
			open={open}
			onCancel={onClose}
			destroyOnClose
			width={1100}
			styles={{ body: { maxHeight: 'calc(100vh - 200px)', overflowY: 'auto' } }}
			footer={
				<Space>
					{watched.isTerminal ? (
						<Button disabled>
							{statusText || t('strategy.backtestRun.status.ended')}
						</Button>
					) : (
						<Button onClick={onCancel} loading={!!canceling} disabled={!runId || watched.isTerminal}>
							{t('strategy.backtestRun.actions.cancel')}
						</Button>
					)}
					<Button type="primary" onClick={onClose}>
						{t('common.close', { defaultValue: 'Close' })}
					</Button>
				</Space>
			}
		>
			{watched.loading ? (
				<Alert type="info" title={t('common.loading')} />
			) : watched.error ? (
				<Alert type="error" title={watched.error} />
			) : (
				<>
					{watched.run?.status === 1 ? (
						<Alert type="info" title={t('strategy.backtestRun.hints.queued')} />
					) : watched.run?.status === 2 ? (
						<Alert type="info" title={t('strategy.backtestRun.hints.running')} />
					) : watched.run?.status === 5 ? (
						<Alert type="warning" title={t('strategy.backtestRun.hints.canceling')} />
					) : null}
					<Descriptions size="small" column={1} bordered>
						<Descriptions.Item label={t('strategy.backtestRun.fields.status')}>{statusText}</Descriptions.Item>
						<Descriptions.Item label={t('strategy.backtestRun.fields.error')}>{watched.run?.error || '-'}</Descriptions.Item>
					</Descriptions>
					<div className="mt-4" />
					<Descriptions size="small" column={2} bordered>
						<Descriptions.Item label={t('strategy.backtestRun.metrics.totalReturn')}>
							{isCompleted ? watched.metrics?.totalReturn ?? '-' : '-'}
						</Descriptions.Item>
						<Descriptions.Item label={t('strategy.backtestRun.metrics.annualReturn')}>
							{isCompleted ? watched.metrics?.annualReturn ?? '-' : '-'}
						</Descriptions.Item>
						<Descriptions.Item label={t('strategy.backtestRun.metrics.maxDrawdown')}>
							{isCompleted ? watched.metrics?.maxDrawdown ?? '-' : '-'}
						</Descriptions.Item>
						<Descriptions.Item label={t('strategy.backtestRun.metrics.sharpe')}>
							{isCompleted ? watched.metrics?.sharpeRatio ?? '-' : '-'}
						</Descriptions.Item>
						<Descriptions.Item label={t('strategy.backtestRun.metrics.winRate')}>
							{isCompleted ? watched.metrics?.winRate ?? '-' : '-'}
						</Descriptions.Item>
						<Descriptions.Item label={t('strategy.backtestRun.metrics.totalTrades')}>
							{isCompleted ? watched.metrics?.totalTrades ?? '-' : '-'}
						</Descriptions.Item>
					</Descriptions>
					<div className="mt-4" />
					<Descriptions size="small" column={1} bordered>
						<Descriptions.Item label={t('strategy.backtestRun.metrics.equityCurvePoints')}>
							{isCompleted && Array.isArray(watched.equityCurve) ? watched.equityCurve.length : 0}
						</Descriptions.Item>
					</Descriptions>

					{isCompleted && (
						<>
							<div className="mt-4" />
							<Typography.Title level={5} style={{ marginBottom: 8 }}>
								{t('strategy.backtestRun.trades.title')}
							</Typography.Title>
							{visibleTradesError ? (
								<Alert type="error" message={t('strategy.backtestRun.trades.loadFailed')} description={visibleTradesError} />
							) : (
								<>
									{summary && (
										<Typography.Paragraph type="secondary" style={{ marginBottom: 8 }}>
											{summary}
										</Typography.Paragraph>
									)}
									<Table<BacktestTrade>
										size="small"
										rowKey="ticket"
										loading={tradesLoading}
										columns={columns}
										dataSource={visibleTrades}
										locale={{ emptyText: t('strategy.backtestRun.trades.empty') }}
										pagination={{ pageSize: 20, showSizeChanger: true, pageSizeOptions: ['10', '20', '50', '100'] }}
										scroll={{ x: 'max-content' }}
									/>
								</>
							)}
						</>
					)}
				</>
			)}
		</Modal>
	);
};

export default BacktestRunDrawer;
