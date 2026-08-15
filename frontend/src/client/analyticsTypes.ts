// analyticsTypes.ts — shared types for analytics data.

// ── Core analytics types ──────────────────────────────────────────────────────

export interface TradeStats {
  totalTrades: number;
  winRate: number;
  profitFactor: number;
  averageProfit: number;
  averageLoss: number;
  largestWin: number;
  largestLoss: number;
  maxConsecutiveWins: number;
  maxConsecutiveLosses: number;
  averageHoldingTime: string;
  netProfit: number;
  totalDeposit: number;
  totalWithdrawal: number;
  netDeposit: number;
}

export interface RiskMetrics {
  maxDrawdownPercent: number;
  sharpeRatio: number;
  sortinoRatio: number;
  calmarRatio: number;
  volatility: number;
  averageDailyReturn: number;
}

export interface SymbolStat {
  symbol: string;
  profit: number;
  tradeSharePercent: number;
}

export interface EquityPoint {
  date: string;
  equity: number;
  balance: number;
  profit: number;
}

export interface DailyPnL {
  day: string;
  date: string;
  pnl: number;
  trades: number;
  lots: number;
  balance: number;
  profitFactor: number;
  maxFloatingLossAmount: number;
  maxFloatingLossRatio: number;
  maxFloatingProfitAmount: number;
  maxFloatingProfitRatio: number;
}

export interface HourlyStat {
  hour: number;
  lots: number;
  balance: number;
  profitFactor: number;
  maxFloatingLossAmount: number;
  maxFloatingLossRatio: number;
  maxFloatingProfitAmount: number;
  maxFloatingProfitRatio: number;
}

export interface AccountAnalyticsData {
  tradeStats: TradeStats;
  riskMetrics: RiskMetrics;
  symbolStats: SymbolStat[];
  equityCurve: EquityPoint[];
  dailyPnl: DailyPnL[];
  hourlyStats: HourlyStat[];
}

export interface TradeRecordItem {
  ticket: number;
  symbol: string;
  type: string;
  volume: number;
  openPrice: number;
  closePrice: number;
  profit: number;
  openTime: string;
  closeTime: string;
  swap: number;
  commission: number;
  comment: string;
  magicNumber: number;
}

export interface RecentTradesData {
  trades: TradeRecordItem[];
  total: number;
}

export interface MonthlyPnLData {
  monthlyPnl: Array<{
    month: number;
    profit: number;
    trades: number;
  }>;
}

export interface MonthlyAnalysisData {
  years: number[];
  data: unknown;
}

// ── Attribution / Rolling / Monthly Detail ────────────────────────────────────

export interface SymbolPnLData {
  symbol: string;
  netProfit: number;
  totalTrades: number;
  winRate: number;
  profitFactor: number;
  tradeSharePercent: number;
}

export interface DirectionBreakdown {
  longProfit: number;
  longTrades: number;
  longWinRate: number;
  shortProfit: number;
  shortTrades: number;
  shortWinRate: number;
}

export interface TradeBucket {
  label: string;
  minValue: number;
  maxValue: number;
  count: number;
}

export interface AttributionAnalysisData {
  symbolPnls: SymbolPnLData[];
  direction: DirectionBreakdown;
  tradeDistribution: {
    profitBuckets: TradeBucket[];
  };
  hourlyPnl: Array<{ hour: number; profit: number; trades: number; winRate: number }>;
}

export interface RollingMetricsData {
  rollingSharpe: Array<{ date: string; value: number }>;
  drawdownEvents: Array<{
    startDate: string;
    endDate: string;
    durationDays: number;
    depthPercent: number;
    recoveryDate: string;
  }>;
  monthlyWinRates: Array<{ month: string; winRate: number; totalTrades: number }>;
  equityCurve: EquityPoint[];
  drawdownCurve: Array<{ date: string; drawdownPercent: number }>;
}

export interface MonthlyDetailData {
  metrics: {
    netReturn: number;
    returnPercent: number;
    totalTrades: number;
    winRate: number;
    profitFactor: number;
    bestTrade: number;
    worstTrade: number;
  };
  symbolPnls: Array<{
    symbol: string;
    netProfit: number;
    trades: number;
    winRate: number;
  }>;
  holdingStats: {
    averageHours: number;
    medianHours: number;
    maxHours: number;
    minHours: number;
  };
  bonus?: {
    riskRatio: number;
    symbolPopularity: Array<{ symbol: string; trades: number; sharePercent: number }>;
    symbolRisks: Array<{ symbol: string; riskRatio: number }>;
    symbolHoldingSplit: Array<{ symbol: string; bullsSeconds: number; shortTermSeconds: number }>;
  };
}

export interface ReportCallbacks {
  onPhase: (phase: string) => void;
  onDelta: (delta: string) => void;
  onSection: (section: string) => void;
  onSummary: (text: string) => void;
  onFindings: (text: string) => void;
  onRecommendations: (text: string) => void;
  onError: (error: string) => void;
  onDone: () => void;
}
