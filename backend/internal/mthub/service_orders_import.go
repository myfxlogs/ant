package mthub

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"alphaforge/internal/model"
)

// ImportBrokerOrder writes a broker-side OrderRecord into the OMS orders table
// and (if closed) into trade_records. Used by reconciliation to converge ghost
// orders (broker has, ant missing) per ADR-0013 §2.3.
//
// Idempotent via ON CONFLICT (mt_account_id, ticket) DO NOTHING on orders
// (UK uk_order_ticket, 001_init.up.sql:111) — uses the real broker ticket,
// NOT the hashToNegative placeholder used by InsertOrder (which conflicts on
// id). trade_records write reuses TradeRecordRepository.Create (hash chain +
// ON CONFLICT (account_id, ticket, close_time) DO NOTHING).
func (s *MtHubService) ImportBrokerOrder(ctx context.Context, accountID string, br *OrderRecord) error {
	if s.omsWriter == nil || br == nil {
		return nil
	}
	pool := s.omsWriter.Pool()
	if pool == nil {
		return nil
	}

	// 1. Insert into orders table with real broker ticket + broker-mapped state.
	state := brokerOrderStateToOMS(br.State)
	if state == "" {
		state = OMSStateReconciling
	}
	plat := platform(accountID, s.hub)
	orderID := uuid.New().String()
	_, err := pool.Exec(ctx, `
		INSERT INTO orders (id, mt_account_id, platform, ticket, symbol, order_type, volume, price, stop_loss, take_profit, state, magic_number)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (mt_account_id, ticket) DO NOTHING
	`, orderID, accountID, plat, br.Ticket, br.SymbolRaw, int16(br.OrderType),
		br.Volume, br.OpenPrice, br.StopLoss, br.TakeProfit, string(state), br.Magic)
	if err != nil {
		return fmt.Errorf("import broker order: insert orders: %w", err)
	}

	// 2. If closed (has CloseTime + ClosePrice), also write to trade_records
	//    via TradeRecordRepository.Create (hash chain + idempotent UK).
	if !br.CloseTime.IsZero() && br.ClosePrice.GreaterThan(decimal.Zero) {
		var userID uuid.UUID
		if err := pool.QueryRow(ctx,
			`SELECT user_id FROM mt_accounts WHERE id = $1::uuid`, accountID).Scan(&userID); err != nil {
			return fmt.Errorf("import broker order: lookup user_id: %w", err)
		}
		rec := &model.TradeRecord{
			UserID:      userID,
			AccountID:   uuid.MustParse(accountID),
			Ticket:      br.Ticket,
			Symbol:      br.SymbolRaw,
			OrderType:   br.OrderTypeString(),
			Volume:      br.Volume,
			OpenPrice:   br.OpenPrice,
			ClosePrice:  br.ClosePrice,
			Profit:      br.Profit,
			Swap:        br.Swap,
			Commission:  br.Commission,
			OpenTime:    br.OpenTime,
			CloseTime:   br.CloseTime,
			StopLoss:    br.StopLoss,
			TakeProfit:  br.TakeProfit,
			MagicNumber: int(br.Magic),
			Platform:    plat,
		}
		if s.tradeRecordRepo != nil {
			if err := s.tradeRecordRepo.Create(ctx, rec); err != nil {
				return fmt.Errorf("import broker order: insert trade_record: %w", err)
			}
		}
	}
	return nil
}
