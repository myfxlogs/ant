package service

import (
	"math"
	"strings"

	"anttrader/internal/model"
)

func (s *AnalyticsService) calculateTradeStats(logs []*model.TradeLog) *model.TradeStats {
	stats := &model.TradeStats{}

	var totalProfit, totalLoss float64
	var profits, losses []float64

	for _, log := range logs {
		if log.Action != "close" && log.Action != "order_close" {
			continue
		}

		stats.TotalTrades++
		stats.TotalVolume += log.Volume

		orderType := strings.ToLower(strings.TrimSpace(log.OrderType))
		if strings.HasPrefix(orderType, "buy") {
			stats.BuyTrades++
		} else if strings.HasPrefix(orderType, "sell") {
			stats.SellTrades++
		}

		if log.Profit > 0 {
			stats.WinningTrades++
			totalProfit += log.Profit
			profits = append(profits, log.Profit)
			if log.Profit > stats.LargestWin {
				stats.LargestWin = log.Profit
			}
		} else if log.Profit < 0 {
			stats.LosingTrades++
			totalLoss += math.Abs(log.Profit)
			losses = append(losses, log.Profit)
			if math.Abs(log.Profit) > stats.LargestLoss {
				stats.LargestLoss = math.Abs(log.Profit)
			}
		}
	}

	stats.TotalProfit = totalProfit
	stats.TotalLoss = totalLoss
	stats.NetProfit = totalProfit - totalLoss

	if stats.TotalTrades > 0 {
		stats.WinRate = float64(stats.WinningTrades) / float64(stats.TotalTrades) * 100
	}

	if totalLoss > 0 {
		stats.ProfitFactor = totalProfit / totalLoss
	} else if totalProfit > 0 {
		stats.ProfitFactor = 999.99
	}

	if len(profits) > 0 {
		stats.AverageProfit = totalProfit / float64(len(profits))
	}
	if len(losses) > 0 {
		stats.AverageLoss = totalLoss / float64(len(losses))
	}
	if stats.TotalTrades > 0 {
		stats.AverageTrade = stats.NetProfit / float64(stats.TotalTrades)
		stats.AverageVolume = stats.TotalVolume / float64(stats.TotalTrades)
	}

	return stats
}

func (s *AnalyticsService) calculateTradeStatsFromRecords(records []*TradeRecord) *model.TradeStats {
	stats := &model.TradeStats{}

	var totalProfit, totalLoss float64
	var profits, losses []float64

	for _, record := range records {
		if isBalanceTradeRecord(record) {
			if record.Profit >= 0 {
				stats.TotalDeposit += record.Profit
			} else {
				stats.TotalWithdrawal += math.Abs(record.Profit)
			}
			continue
		}

		stats.TotalTrades++
		stats.TotalVolume += record.Volume

		orderType := strings.ToLower(strings.TrimSpace(record.OrderType))
		if strings.HasPrefix(orderType, "buy") {
			stats.BuyTrades++
		} else if strings.HasPrefix(orderType, "sell") {
			stats.SellTrades++
		}

		if record.Profit > 0 {
			stats.WinningTrades++
			totalProfit += record.Profit
			profits = append(profits, record.Profit)
			if record.Profit > stats.LargestWin {
				stats.LargestWin = record.Profit
			}
		} else if record.Profit < 0 {
			stats.LosingTrades++
			totalLoss += math.Abs(record.Profit)
			losses = append(losses, record.Profit)
			if math.Abs(record.Profit) > stats.LargestLoss {
				stats.LargestLoss = math.Abs(record.Profit)
			}
		}
	}

	stats.NetDeposit = stats.TotalDeposit - stats.TotalWithdrawal
	stats.TotalProfit = totalProfit
	stats.TotalLoss = totalLoss
	stats.NetProfit = totalProfit - totalLoss

	if stats.TotalTrades > 0 {
		stats.WinRate = float64(stats.WinningTrades) / float64(stats.TotalTrades) * 100
	}

	if totalLoss > 0 {
		stats.ProfitFactor = totalProfit / totalLoss
	} else if totalProfit > 0 {
		stats.ProfitFactor = 999.99
	}

	if len(profits) > 0 {
		stats.AverageProfit = totalProfit / float64(len(profits))
	}
	if len(losses) > 0 {
		stats.AverageLoss = totalLoss / float64(len(losses))
	}
	if stats.TotalTrades > 0 {
		stats.AverageTrade = stats.NetProfit / float64(stats.TotalTrades)
		stats.AverageVolume = stats.TotalVolume / float64(stats.TotalTrades)
	}

	return stats
}
