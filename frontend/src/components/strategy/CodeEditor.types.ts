export interface StrategyTemplate {
  name: string;
  description: string;
  code: string;
}

export interface BacktestMetrics {
  total_return: number;
  annual_return: number;
  max_drawdown: number;
  sharpeRatio: number;
  winRate: number;
  profitFactor: number;
  totalTrades: number;
  winning_trades: number;
  losing_trades: number;
}

export interface PreviewResult {
  success: boolean;
  signal?: {
    signal: string;
    symbol: string;
    price?: number;
    confidence: number;
    reason?: string;
    risk_level: string;
  };
  error?: string;
  logs?: string[];
}

export interface BacktestResult {
  success: boolean;
  metrics?: BacktestMetrics;
  equity_curve?: number[];
  error?: string;
}

export interface Account {
  id: string;
  login: string;
  mtType: string;
  isDisabled?: boolean;
}

export interface CodeEditorProps {
  code?: string;
  onCodeChange?: (code: string) => void;
  initialCode?: string;
}
