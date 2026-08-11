// autoTrading.ts — AutoTradingService client wrapper.
// Covers global settings, risk config, risk checks, position sizing, and trading logs.

import { autoTradingClient } from './connect';
import { create } from '@bufbuild/protobuf';
import { UpdateGlobalSettingsRequestSchema, GetAutoTradingStatusRequestSchema, type GlobalSettings, type AutoTradingStatus } from '../gen/ant/v1/auto_trading_settings_pb';
import { UpdateRiskConfigRequestSchema, type RiskConfig } from '../gen/ant/v1/auto_trading_risk_config_pb';
import { CheckRiskLimitsRequestSchema, type CheckRiskLimitsResponse } from '../gen/ant/v1/auto_trading_risk_check_pb';
import { CalculatePositionSizeRequestSchema, type CalculatePositionSizeResponse } from '../gen/ant/v1/auto_trading_position_size_pb';

export type { GlobalSettings, AutoTradingStatus } from '../gen/ant/v1/auto_trading_settings_pb';
export type { RiskConfig } from '../gen/ant/v1/auto_trading_risk_config_pb';
export type { CheckRiskLimitsResponse } from '../gen/ant/v1/auto_trading_risk_check_pb';
export type { CalculatePositionSizeResponse } from '../gen/ant/v1/auto_trading_position_size_pb';

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
    return autoTradingClient.updateGlobalSettings(create(UpdateGlobalSettingsRequestSchema, req as Record<string, unknown>));
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
    return autoTradingClient.updateRiskConfig(create(UpdateRiskConfigRequestSchema, req as Record<string, unknown>));
  },

  // ── Risk Checks ──
  checkRiskLimits: async (req: {
    accountId: string;
    symbol: string;
    volume: number;
    stopLoss?: number;
    takeProfit?: number;
  }): Promise<CheckRiskLimitsResponse> => {
    return autoTradingClient.checkRiskLimits(create(CheckRiskLimitsRequestSchema, req as Record<string, unknown>));
  },

  calculatePositionSize: async (req: {
    accountId: string;
    symbol: string;
    stopLossPrice?: number;
    riskPercent?: number;
  }): Promise<CalculatePositionSizeResponse> => {
    return autoTradingClient.calculatePositionSize(create(CalculatePositionSizeRequestSchema, req as Record<string, unknown>));
  },

  // ── Status & Logs ──
  getAutoTradingStatus: async (): Promise<AutoTradingStatus> => {
    return autoTradingClient.getAutoTradingStatus(create(GetAutoTradingStatusRequestSchema));
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

  getRecentTradingLogs: async (req: { userId: string; limit?: number }) => {
    return autoTradingClient.getRecentTradingLogs({ userId: req.userId, limit: req.limit ?? 20 });
  },
};
