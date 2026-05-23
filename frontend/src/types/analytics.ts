export interface TradeStats {
  total_trades: number;
  winning_trades: number;
  losing_trades: number;
  win_rate: number;
  total_profit: number;
  total_loss: number;
  net_profit: number;
  profit_factor: number;
  average_profit: number;
  average_loss: number;
  average_trade: number;
  largest_win: number;
  largest_loss: number;
  max_consecutive_wins: number;
  max_consecutive_losses: number;
  average_holding_time: string;
  total_deposit: number;
  total_withdrawal: number;
  net_deposit: number;
}

export interface RiskMetrics {
  max_drawdown: number;
  max_drawdown_percent: number;
  sharpe_ratio: number;
  sortino_ratio: number;
  calmar_ratio: number;
  volatility: number;
  value_at_risk_95: number;
  expected_shortfall: number;
  max_daily_loss: number;
  max_weekly_loss: number;
  average_daily_return: number;
  return_std_dev: number;
}

export interface SymbolStats {
  symbol: string;
  total_trades: number;
  winning_trades: number;
  losing_trades: number;
  win_rate: number;
  total_profit: number;
  total_loss: number;
  net_profit: number;
  profit_factor: number;
  average_profit: number;
  total_volume: number;
  average_volume: number;
  largest_win: number;
  largest_loss: number;
  average_holding_time: string;
}

export interface DailyEquity {
  date: string;
  equity: number;
  balance: number;
  profit: number;
  drawdown: number;
}

export interface TradeReport {
  account_id: string;
  start_date: string;
  end_date: string;
  trade_stats: TradeStats;
  risk_metrics: RiskMetrics;
  symbol_stats: SymbolStats[];
  daily_equity: DailyEquity[];
  equity_curve: number[];
  drawdown_curve: number[];
}

export interface TradeRecord {
  id: string;
  account_id: string;
  ticket: number;
  symbol: string;
  order_type: string;
  volume: number;
  open_price: number;
  close_price: number;
  profit: number;
  swap: number;
  commission: number;
  open_time: string;
  close_time: string;
  stop_loss: number;
  take_profit: number;
  order_comment: string;
  magic_number: number;
}

export interface MonthlyPnL {
  month: string;
  month_num: number;
  profit: number;
  trades: number;
  win_trades: number;
  loss_trades: number;
}

export interface DailyPnL {
  day: string;
  day_num: number;
  pnl: number;
  trades: number;
}

export interface HourlyStats {
  hour: string;
  hour_start: number;
  trades: number;
  profit: number;
  win_rate: number;
  avg_pnl: number;
}

export interface EquityPoint {
  date: string;
  equity: number;
  balance: number;
  profit: number;
}

export interface AccountAnalytics {
  trade_stats: TradeStats | null;
  risk_metrics: RiskMetrics | null;
  symbol_stats: SymbolStats[];
  monthly_pnl: MonthlyPnL[];
  daily_pnl: DailyPnL[];
  hourly_stats: HourlyStats[];
  equity_curve: EquityPoint[];
  recent_trades: TradeRecord[];
}
