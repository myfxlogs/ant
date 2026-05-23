package connection

import (
	"context"
	"fmt"
	"math"
	"os"
	"time"

	"github.com/google/uuid"

	"anttrader/internal/model"
	mt4pb "anttrader/mt4"
	mt5pb "anttrader/mt5"
	"anttrader/pkg/logger"

	"go.uber.org/zap"
)

func (m *ConnectionManager) AutoSyncOrderHistoryOnConnect(accountID uuid.UUID, mtType string) {
	enabled := os.Getenv("ANTRADER_AUTO_SYNC_HISTORY_ON_CONNECT")
	if enabled == "0" || enabled == "false" {
		return
	}

	if m.tradeRecordRepo == nil {
		return
	}

	const cooldown = 10 * time.Minute
	now := time.Now()

	m.historySyncMu.Lock()
	last, ok := m.lastAutoSyncAt[accountID]
	if ok && now.Sub(last) < cooldown {
		m.historySyncMu.Unlock()
		return
	}
	m.lastAutoSyncAt[accountID] = now
	m.historySyncMu.Unlock()

	m.SyncOrderHistory(accountID, mtType)
}

func (m *ConnectionManager) SyncOrderHistory(accountID uuid.UUID, mtType string) {
	go m.syncHistoryAsync(accountID, mtType)
}

func (m *ConnectionManager) syncHistoryAsync(accountID uuid.UUID, mtType string) {
	if m.tradeRecordRepo == nil {
		logger.Warn("TradeRecordRepo not set, skipping history sync")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	var lastSyncTime *time.Time
	lastSyncTime, err := m.tradeRecordRepo.GetLastSyncTime(ctx, accountID)
	if err != nil {
		lastSyncTime = nil
	}

	// 从上次同步时间开始，但最多回溯 3 个月（经纪商通常只保留 3 个月数据）
	from := time.Now().AddDate(0, -3, 0)
	if lastSyncTime != nil && lastSyncTime.After(from) {
		from = *lastSyncTime
	}
	to := time.Now()

	fromStr := from.Format("2006-01-02T15:04:05")
	toStr := to.Format("2006-01-02T15:04:05")

	m.connectionsMu.RLock()
	conn, exists := m.connections[accountID]
	m.connectionsMu.RUnlock()

	if !exists || conn == nil {
		// Best-effort reconnect to recover MT history sync after process restart or stale connection map.
		acc, accErr := m.accountRepo.GetByID(ctx, accountID)
		if accErr != nil || acc == nil {
			logger.Warn("Connection not found for history sync",
				zap.String("account_id", accountID.String()))
			return
		}
		connectCtx, cancelConnect := context.WithTimeout(context.Background(), 45*time.Second)
		connectErr := m.Connect(connectCtx, acc)
		cancelConnect()
		if connectErr != nil {
			logger.Warn("History sync reconnect failed",
				zap.String("account_id", accountID.String()),
				zap.Error(connectErr))
			return
		}
		m.connectionsMu.RLock()
		conn = m.connections[accountID]
		m.connectionsMu.RUnlock()
		if conn == nil {
			logger.Warn("Connection still missing after reconnect for history sync",
				zap.String("account_id", accountID.String()))
			return
		}
	}

	var orders interface{}
	var orderErr error

	if mtType == "MT4" {
		mt4Conn := conn.GetMT4Connection()
		if mt4Conn == nil {
			acc, accErr := m.accountRepo.GetByID(ctx, accountID)
			if accErr == nil && acc != nil {
				connectCtx, cancelConnect := context.WithTimeout(context.Background(), 45*time.Second)
				_ = m.Connect(connectCtx, acc)
				cancelConnect()
				m.connectionsMu.RLock()
				conn = m.connections[accountID]
				m.connectionsMu.RUnlock()
				if conn != nil {
					mt4Conn = conn.GetMT4Connection()
				}
			}
			if mt4Conn == nil {
				logger.Warn("MT4 connection not available for history sync")
				return
			}
		}
		orders, orderErr = mt4Conn.OrderHistory(ctx, fromStr, toStr)
	} else {
		mt5Conn := conn.GetMT5Connection()
		if mt5Conn == nil {
			acc, accErr := m.accountRepo.GetByID(ctx, accountID)
			if accErr == nil && acc != nil {
				connectCtx, cancelConnect := context.WithTimeout(context.Background(), 45*time.Second)
				_ = m.Connect(connectCtx, acc)
				cancelConnect()
				m.connectionsMu.RLock()
				conn = m.connections[accountID]
				m.connectionsMu.RUnlock()
				if conn != nil {
					mt5Conn = conn.GetMT5Connection()
				}
			}
			if mt5Conn == nil {
				logger.Warn("MT5 connection not available for history sync")
				return
			}
		}
		orders, orderErr = mt5Conn.OrderHistory(ctx, fromStr, toStr)
	}

	if orderErr != nil {
		logger.Error("Failed to get order history",
			zap.String("account_id", accountID.String()),
			zap.Error(orderErr))
		return
	}

	var records []*model.TradeRecord

	if mtType == "MT4" {
		if mt4Orders, ok := orders.([]*mt4pb.Order); ok {
			records = convertMT4OrdersToRecords(accountID, mt4Orders)
		}
	} else {
		if mt5Orders, ok := orders.([]*mt5pb.Order); ok {
			records = convertMT5OrdersToRecords(accountID, mt5Orders)
		}
	}

	if len(records) > 0 {
		if err := m.tradeRecordRepo.BatchCreate(ctx, records); err != nil {
			logger.Error("Failed to save order history",
				zap.String("account_id", accountID.String()),
				zap.Error(err))
			return
		}
		logger.Info("History sync persisted records",
			zap.String("account_id", accountID.String()),
			zap.String("mt_type", mtType),
			zap.Int("records", len(records)))
	}
}

func convertMT4OrdersToRecords(accountID uuid.UUID, orders []*mt4pb.Order) []*model.TradeRecord {
	records := make([]*model.TradeRecord, 0, len(orders))
	for _, o := range orders {
		openTime := time.Time{}
		if t := o.GetOpenTime(); t != nil {
			openTime = t.AsTime()
		}
		closeTime := time.Time{}
		if t := o.GetCloseTime(); t != nil {
			closeTime = t.AsTime()
		}
		records = append(records, &model.TradeRecord{
			AccountID:   accountID,
			Ticket:      int64(o.GetTicket()),
			Symbol:      o.GetSymbol(),
			OrderType:   formatOrderType(int32(o.GetType())),
			Volume:      truncateVolume(o.GetLots()),
			OpenPrice:   truncatePrice(o.GetOpenPrice()),
			ClosePrice:  truncatePrice(o.GetClosePrice()),
			Profit:      truncateProfit(o.GetProfit()),
			Swap:        truncateProfit(o.GetSwap()),
			Commission:  truncateProfit(o.GetCommission()),
			OpenTime:    openTime,
			CloseTime:   closeTime,
			StopLoss:    truncatePrice(o.GetStopLoss()),
			TakeProfit:  truncatePrice(o.GetTakeProfit()),
			MagicNumber: int(o.GetMagicNumber()),
			Platform:    "MT4",
		})
	}
	return records
}

func convertMT5OrdersToRecords(accountID uuid.UUID, orders []*mt5pb.Order) []*model.TradeRecord {
	records := make([]*model.TradeRecord, 0, len(orders))
	for _, o := range orders {
		symbol := o.GetSymbol()
		if symbol == "" {
			symbol = "UNKNOWN"
		}

		orderType := formatOrderType(int32(o.GetOrderType()))
		dealType := o.GetDealType()
		if dealType >= 2 {
			orderType = formatDealType(int32(dealType))
			if symbol == "UNKNOWN" {
				symbol = ""
			}
		}

		openTime := time.Time{}
		if t := o.GetOpenTime(); t != nil {
			openTime = t.AsTime()
		}
		closeTime := time.Time{}
		if t := o.GetCloseTime(); t != nil {
			closeTime = t.AsTime()
		}
		records = append(records, &model.TradeRecord{
			AccountID:   accountID,
			Ticket:      o.GetTicket(),
			Symbol:      symbol,
			OrderType:   orderType,
			Volume:      truncateVolume(o.GetLots()),
			OpenPrice:   truncatePrice(o.GetOpenPrice()),
			ClosePrice:  truncatePrice(o.GetClosePrice()),
			Profit:      truncateProfit(o.GetProfit()),
			Swap:        truncateProfit(o.GetSwap()),
			Commission:  truncateProfit(o.GetCommission()),
			OpenTime:    openTime,
			CloseTime:   closeTime,
			StopLoss:    truncatePrice(o.GetStopLoss()),
			TakeProfit:  truncatePrice(o.GetTakeProfit()),
			MagicNumber: int(o.GetExpertId()),
			Platform:    "MT5",
		})
	}
	return records
}

func formatDealType(dealType int32) string {
	switch dealType {
	case 0:
		return "BUY"
	case 1:
		return "SELL"
	case 2:
		return "BALANCE"
	case 3:
		return "CREDIT"
	case 4:
		return "CHARGE"
	case 5:
		return "CORRECTION"
	case 6:
		return "BONUS"
	case 7:
		return "COMMISSION"
	default:
		return fmt.Sprintf("UNKNOWN_DEAL_%d", dealType)
	}
}

func formatDealTypeString(dealType string) string {
	switch dealType {
	case "0", "Buy", "BUY":
		return "BUY"
	case "1", "Sell", "SELL":
		return "SELL"
	case "2":
		return "BALANCE"
	case "3":
		return "CREDIT"
	case "4":
		return "CHARGE"
	case "5":
		return "CORRECTION"
	case "6":
		return "BONUS"
	case "7":
		return "COMMISSION"
	default:
		return fmt.Sprintf("UNKNOWN_DEAL_%s", dealType)
	}
}

func formatOrderType(orderType int32) string {
	switch orderType {
	case 0:
		return "BUY"
	case 1:
		return "SELL"
	case 2:
		return "BUY_LIMIT"
	case 3:
		return "SELL_LIMIT"
	case 4:
		return "BUY_STOP"
	case 5:
		return "SELL_STOP"
	case 6:
		return "BALANCE"
	case 7:
		return "CREDIT"
	case 100:
		return "BALANCE"
	case 101:
		return "CREDIT"
	default:
		return fmt.Sprintf("UNKNOWN_%d", orderType)
	}
}

func truncateFloat(val float64, precision int) float64 {
	multiplier := math.Pow(10, float64(precision))
	return math.Trunc(val*multiplier) / multiplier
}

func truncatePrice(val float64) float64 {
	return truncateFloat(val, 5)
}

func truncateVolume(val float64) float64 {
	return truncateFloat(val, 4)
}

func truncateProfit(val float64) float64 {
	return truncateFloat(val, 2)
}
