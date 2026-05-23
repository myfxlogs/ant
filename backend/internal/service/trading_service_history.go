package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"anttrader/internal/model"
	"anttrader/pkg/faultisol"
)

type HistoryOrderResponse struct {
	Ticket     int64   `json:"ticket"`
	Symbol     string  `json:"symbol"`
	Type       string  `json:"type"`
	Volume     float64 `json:"volume"`
	OpenPrice  float64 `json:"open_price"`
	ClosePrice float64 `json:"close_price"`
	Profit     float64 `json:"profit"`
	Swap       float64 `json:"swap"`
	Commission float64 `json:"commission"`
	OpenTime   string  `json:"open_time"`
	CloseTime  string  `json:"close_time"`
	StopLoss   float64 `json:"stop_loss"`
	TakeProfit float64 `json:"take_profit"`
	Comment    string  `json:"comment"`
	Magic      int64   `json:"magic_number"`
}

func (s *TradingService) GetOrderHistory(ctx context.Context, userID, accountID uuid.UUID, from, to time.Time) ([]*HistoryOrderResponse, error) {
	var result []*HistoryOrderResponse

	err := s.executor.Execute(ctx, func(ctx context.Context) error {
		account, err := s.getAccountAndVerify(ctx, userID, accountID)
		if err != nil {
			return err
		}

		if account.MTType == "MT4" {
			result, err = s.getOrderHistoryMT4(ctx, accountID, from, to)
		} else {
			result, err = s.getOrderHistoryMT5(ctx, accountID, from, to)
		}

		return err
	}, faultisol.WithOperation("get_order_history"), faultisol.WithTimeout(60*time.Second))

	return result, err
}

func (s *TradingService) getOrderHistoryMT4(ctx context.Context, accountID uuid.UUID, from, to time.Time) ([]*HistoryOrderResponse, error) {
	conn, err := s.getMT4Connection(accountID)
	if err != nil {
		return nil, ErrAccountNotConnected
	}

	fromStr := from.Format("2006-01-02T15:04:05")
	toStr := to.Format("2006-01-02T15:04:05")

	orders, err := conn.OrderHistory(ctx, fromStr, toStr)
	if err != nil {
		return nil, err
	}

	var history []*HistoryOrderResponse
	for _, order := range orders {
		openTime := ""
		closeTime := ""
		if order.GetOpenTime() != nil {
			openTime = order.GetOpenTime().AsTime().Format("2006-01-02T15:04:05Z")
		}
		if order.GetCloseTime() != nil {
			closeTime = order.GetCloseTime().AsTime().Format("2006-01-02T15:04:05Z")
		}

		history = append(history, &HistoryOrderResponse{
			Ticket:     int64(order.GetTicket()),
			Symbol:     order.GetSymbol(),
			Type:       OrderTypeToString(int32(order.GetType())),
			Volume:     order.GetLots(),
			OpenPrice:  order.GetOpenPrice(),
			ClosePrice: order.GetClosePrice(),
			Profit:     order.GetProfit(),
			Swap:       order.GetSwap(),
			Commission: order.GetCommission(),
			OpenTime:   openTime,
			CloseTime:  closeTime,
			StopLoss:   order.GetStopLoss(),
			TakeProfit: order.GetTakeProfit(),
			Comment:    order.GetComment(),
			Magic:      int64(order.GetMagicNumber()),
		})
	}

	return history, nil
}

func (s *TradingService) getOrderHistoryMT5(ctx context.Context, accountID uuid.UUID, from, to time.Time) ([]*HistoryOrderResponse, error) {
	conn, err := s.getMT5Connection(accountID)
	if err != nil {
		return nil, ErrAccountNotConnected
	}

	fromStr := from.Format("2006-01-02T15:04:05")
	toStr := to.Format("2006-01-02T15:04:05")

	orders, err := conn.OrderHistory(ctx, fromStr, toStr)
	if err != nil {
		return nil, err
	}

	var history []*HistoryOrderResponse
	for _, order := range orders {
		orderType := OrderTypeToString(int32(order.GetOrderType()))
		dealType := order.GetDealType()
		if dealType >= 2 {
			orderType = OrderTypeToString(int32(dealType))
		}

		openTime := ""
		closeTime := ""
		if order.GetOpenTime() != nil {
			openTime = order.GetOpenTime().AsTime().Format("2006-01-02T15:04:05Z")
		}
		if order.GetCloseTime() != nil {
			closeTime = order.GetCloseTime().AsTime().Format("2006-01-02T15:04:05Z")
		}

		history = append(history, &HistoryOrderResponse{
			Ticket:     order.GetTicket(),
			Symbol:     order.GetSymbol(),
			Type:       orderType,
			Volume:     order.GetLots(),
			OpenPrice:  order.GetOpenPrice(),
			ClosePrice: order.GetClosePrice(),
			Profit:     order.GetProfit(),
			Swap:       order.GetSwap(),
			Commission: order.GetCommission(),
			OpenTime:   openTime,
			CloseTime:  closeTime,
			StopLoss:   order.GetStopLoss(),
			TakeProfit: order.GetTakeProfit(),
			Comment:    order.GetComment(),
			Magic:      order.GetExpertId(),
		})
	}

	return history, nil
}

func (s *TradingService) SyncOrderHistory(ctx context.Context, userID, accountID uuid.UUID, from, to time.Time) (int, error) {
	var account *model.MTAccount
	var err error

	if userID != uuid.Nil {
		account, err = s.getAccountAndVerify(ctx, userID, accountID)
		if err != nil {
			return 0, err
		}
	} else {
		account, err = s.accountRepo.GetByID(ctx, accountID)
		if err != nil {
			return 0, err
		}
	}

	var history []*HistoryOrderResponse
	if account.MTType == "MT4" {
		history, err = s.getOrderHistoryMT4(ctx, accountID, from, to)
	} else {
		history, err = s.getOrderHistoryMT5(ctx, accountID, from, to)
	}
	if err != nil {
		return 0, err
	}

	if len(history) == 0 {
		return 0, nil
	}

	var records []*model.TradeRecord
	for _, h := range history {
		openTime, _ := time.Parse("2006-01-02T15:04:05Z", h.OpenTime)
		closeTime, _ := time.Parse("2006-01-02T15:04:05Z", h.CloseTime)

		records = append(records, &model.TradeRecord{
			AccountID:    accountID,
			Ticket:       h.Ticket,
			Symbol:       h.Symbol,
			OrderType:    h.Type,
			Volume:       h.Volume,
			OpenPrice:    h.OpenPrice,
			ClosePrice:   h.ClosePrice,
			Profit:       h.Profit,
			Swap:         h.Swap,
			Commission:   h.Commission,
			OpenTime:     openTime,
			CloseTime:    closeTime,
			StopLoss:     h.StopLoss,
			TakeProfit:   h.TakeProfit,
			OrderComment: h.Comment,
			MagicNumber:  int(h.Magic),
			Platform:     account.MTType,
		})
	}

	if s.tradeRecordRepo != nil {
		if err := s.tradeRecordRepo.BatchCreate(ctx, records); err != nil {
			return 0, fmt.Errorf("failed to save trade records: %w", err)
		}
	}

	return len(records), nil
}
