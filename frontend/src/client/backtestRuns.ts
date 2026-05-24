import { backtestTradesClient } from './connect';

export interface BacktestTrade {
	ticket: number;
	side: string;
	volume: number;
	open_ts: number;
	open_price: number;
	close_ts: number;
	close_price: number;
	pnl: number;
	commission: number;
	reason: string;
}

export interface BacktestTradeSummary {
	count: number;
	wins: number;
	losses: number;
	netPnl: number;
}

export interface BacktestTradesResult {
	trades: BacktestTrade[];
	summary: BacktestTradeSummary;
}

export const backtestRunsApi = {
	getTrades: async (runId: string): Promise<BacktestTradesResult> => {
		const data = await backtestTradesClient.listBacktestRunTrades({ runId });
		const trades = (data.trades || []).map((t) => ({
			ticket: Number(t.ticket),
			side: t.side,
			volume: t.volume,
			open_ts: Number(t.openTs),
			open_price: t.openPrice,
			close_ts: Number(t.closeTs),
			close_price: t.closePrice,
			pnl: t.pnl,
			commission: t.commission,
			reason: t.reason,
		}));
		return {
			trades,
			summary: {
				count: data.summary?.count || 0,
				wins: data.summary?.wins || 0,
				losses: data.summary?.losses || 0,
				netPnl: data.summary?.netPnl || 0,
			},
		};
	},
};
