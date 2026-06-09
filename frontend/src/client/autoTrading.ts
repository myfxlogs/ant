// autoTrading.ts — AutoTradingService client wrapper.
// Covers global settings, risk config, risk checks, position sizing, and trading logs.

import { autoTradingClient } from './connect';
import type {
  GlobalSettings,
  RiskConfig,
  CheckRiskLimitsResponse,
  CalculatePositionSizeResponse,
  AutoTradingStatus,
} from '../gen/ant/v1/auto_trading_pb';

export type {
  GlobalSettings,
  RiskConfig,
  CheckRiskLimitsResponse,
  CalculatePositionSizeResponse,
  AutoTradingStatus,
} from '../gen/ant/v1/auto_trading_pb';

export const autoTradingApi = {
  // ── Global Settings ──
  getGlobalSettings: async (): Promise<GlobalSettings> => {
    return autoTradingClient.getGlobalSettings({});
  },

  updateGlobalSettings: async (req: {
    maxPositions?: number;
    maxLeverage?: number;
    defaultVolume?: number;
    stopLossPercent?: number;
    takeProfitPercent?: number;
    maxDailyLoss?: number;
    maxConsecutiveLosses?: number;
    enabledPairs?: string[];
    excludedPairs?: string[];
    tradingHoursStart?: string;
    tradingHoursEnd?: string;
  }): Promise<GlobalSettings> => {
    return autoTradingClient.updateGlobalSettings(req);
  },

  toggleAutoTrade: async (enabled: boolean): Promise<{ success: boolean; message: string }> => {
    return autoTradingClient.toggleAutoTrade({ enabled });
  },

  // ── Risk Config ──
  getRiskConfig: async (accountId: string): Promise<RiskConfig> => {
    return autoTradingClient.getRiskConfig({ accountId });
  },

  updateRiskConfig: async (req: {
    accountId: string;
    maxPositionSize?: number;
    maxOpenPositions?: number;
    maxDailyTrades?: number;
    maxLossPerTrade?: number;
    maxDailyLoss?: number;
    riskPerTradePercent?: number;
    maxSlippageBps?: number;
    enableCircuitBreaker?: boolean;
    circuitBreakerLossThreshold?: number;
    circuitBreakerDurationMinutes?: number;
  }): Promise<RiskConfig> => {
    return autoTradingClient.updateRiskConfig(req);
  },

  // ── Risk Checks ──
  checkRiskLimits: async (req: {
    accountId: string;
    symbol: string;
    volume: number;
    stopLoss?: number;
    takeProfit?: number;
  }): Promise<CheckRiskLimitsResponse> => {
    return autoTradingClient.checkRiskLimits(req);
  },

  calculatePositionSize: async (req: {
    accountId: string;
    symbol: string;
    stopLossPrice?: number;
    riskPercent?: number;
  }): Promise<CalculatePositionSizeResponse> => {
    return autoTradingClient.calculatePositionSize(req);
  },

  // ── Status & Logs ──
  getAutoTradingStatus: async (accountId: string): Promise<AutoTradingStatus> => {
    return autoTradingClient.getAutoTradingStatus({ accountId });
  },

  getTradingLogs: async (req: {
    accountId: string;
    from?: string;
    to?: string;
    page?: number;
    pageSize?: number;
  }) => {
    return autoTradingClient.getTradingLogs(req);
  },

  getRecentTradingLogs: async (req: { accountId: string; limit?: number }) => {
    return autoTradingClient.getRecentTradingLogs(req);
  },
};
