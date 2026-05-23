package connect

import (
	"context"
	"math"
	"strings"
	"time"

	v1 "anttrader/gen/proto"
	"anttrader/internal/model"
	pb "anttrader/mt4"
	mt5pb "anttrader/mt5"
	"anttrader/pkg/logger"

	"go.uber.org/zap"
)

// marginLevelFromFields returns margin level percent when gateway sends 0.
func marginLevelFromFields(ml, margin, equity float64) float64 {
	if ml <= 0 && margin > 0 {
		computed := (equity / margin) * 100
		if !math.IsNaN(computed) && !math.IsInf(computed, 0) {
			return computed
		}
	}
	return ml
}

// mt5StreamProfitFallback uses OnOrderProfit frame only when AccountSummary is unavailable.
// Official MT5 account floating P/L is ACCOUNT_PROFIT (see MQL5 AccountInfoDouble / ENUM_ACCOUNT_INFO_DOUBLE).
func mt5StreamProfitFallback(latest *mt5pb.ProfitUpdate) float64 {
	if latest == nil {
		return 0
	}
	b, e := latest.GetBalance(), latest.GetEquity()
	credit := 0.0
	if latest.Credit != nil {
		credit = *latest.Credit
		if math.IsNaN(credit) || math.IsInf(credit, 0) {
			credit = 0
		}
	}
	d := e - b - credit
	if !math.IsNaN(d) && !math.IsInf(d, 0) {
		return d
	}
	p := latest.GetProfit()
	if !math.IsNaN(p) && !math.IsInf(p, 0) {
		return p
	}
	return 0
}

func (s *StreamService) startProfitStream(accountStream *AccountStream, account *model.MTAccount) {
	s.startProfitStreamWithCtx(accountStream.Ctx, accountStream, account)
}

func (s *StreamService) startProfitStreamWithCtx(ctx context.Context, accountStream *AccountStream, account *model.MTAccount) {
	switch strings.ToUpper(account.MTType) {
	case "MT4":
		conn, err := s.connManager.GetMT4Connection(account.ID)
		if err != nil {
			logger.Error("获取 MT4 连接失败", zap.String("account_id", account.ID.String()), zap.Error(err))
			return
		}

		mt4ProfitCh := conn.GetProfitChannel()
		if mt4ProfitCh == nil {
			logger.Error("MT4 利润通道为空", zap.String("account_id", account.ID.String()))
			return
		}

		flushTick := time.NewTicker(500 * time.Millisecond)
		defer flushTick.Stop()
		var latest *pb.ProfitUpdate
		dirty := false
		lastMismatchLog := time.Time{}
		for {
			select {
			case <-ctx.Done():
				return
			case upd, ok := <-mt4ProfitCh:
				if !ok {
					return
				}
				latest = upd
				dirty = true
			case <-flushTick.C:
				if !dirty {
					continue
				}
				dirty = false
				if ctx.Err() != nil {
					return
				}
				if latest == nil {
					continue
				}

				var orderProfits []*v1.OrderProfitItem
				for _, o := range latest.GetOrders() {
					orderProfits = append(orderProfits, &v1.OrderProfitItem{
						Ticket:       int64(o.GetTicket()),
						Symbol:       o.GetSymbol(),
						Profit:       o.GetProfit(),
						Volume:       o.GetLots(),
						CurrentPrice: o.GetClosePrice(),
					})
				}

				profitValue := latest.GetEquity() - latest.GetBalance() - latest.GetCredit()
				if math.IsNaN(profitValue) || math.IsInf(profitValue, 0) {
					profitValue = latest.GetProfit()
				}

				diff := math.Abs(profitValue - latest.GetProfit())
				if diff > 1 && time.Since(lastMismatchLog) > 10*time.Second {
					logger.Warn("MT4 profit mismatch",
						zap.String("account_id", account.ID.String()),
						zap.Float64("calc_profit", profitValue),
						zap.Float64("mt_profit", latest.GetProfit()),
						zap.Float64("balance", latest.GetBalance()),
						zap.Float64("equity", latest.GetEquity()),
						zap.Float64("credit", latest.GetCredit()))
					lastMismatchLog = time.Now()
				}

				profitEvent := &v1.ProfitUpdateEvent{
					AccountId:   accountStream.AccountID,
					Balance:     latest.GetBalance(),
					Credit:      latest.GetCredit(),
					Profit:      profitValue,
					Equity:      latest.GetEquity(),
					Margin:      latest.GetMargin(),
					FreeMargin:  latest.GetFreeMargin(),
					MarginLevel: marginLevelFromFields(latest.GetMarginLevel(), latest.GetMargin(), latest.GetEquity()),
					Orders:      orderProfits,
					ProfitPercent: func() float64 {
						if latest.GetBalance() <= 0 {
							return 0
						}
						return (profitValue / latest.GetBalance()) * 100
					}(),
				}

				s.publishProfitEvent(accountStream.AccountID, profitEvent)
				accountStream.setProfitSnapshot(profitEvent)
			}
		}

	case "MT5":
		conn, err := s.connManager.GetMT5Connection(account.ID)
		if err != nil {
			logger.Error("获取 MT5 连接失败", zap.String("account_id", account.ID.String()), zap.Error(err))
			return
		}

		mt5ProfitCh := conn.GetProfitChannel()
		if mt5ProfitCh == nil {
			logger.Error("MT5 利润通道为空", zap.String("account_id", account.ID.String()))
			return
		}

		// OnOrderProfit stream can expose a Profit field that does not match terminal ACCOUNT_PROFIT.
		// mtapi unary AccountSummary aligns with terminal account snapshot (same family as MQL5 AccountInfoDouble).
		// Each stream message triggers AccountSummary; on failure use this frame's OnOrderProfit for cards (avoid stale cache).

		for {
			select {
			case <-ctx.Done():
				return
			case upd, ok := <-mt5ProfitCh:
				if !ok {
					return
				}
				if ctx.Err() != nil {
					return
				}
				if upd == nil {
					continue
				}

				// Do not forward OnOrderProfit.orders: mtapi often sends a subset per tick, which makes UIs
				// treat the list as authoritative and flap vs full OpenedOrders/getPositions.
				// Open positions list is driven by order/position streams + Connect getPositions.

				var balance, equity, margin, freeMargin, marginLevel, profitValue, creditOut float64
				sctx, cancel := context.WithTimeout(ctx, 5*time.Second)
				asum, aerr := conn.AccountSummary(sctx)
				cancel()
				if aerr == nil && asum != nil {
					balance = asum.GetBalance()
					creditOut = asum.GetCredit()
					if math.IsNaN(creditOut) || math.IsInf(creditOut, 0) {
						creditOut = 0
					}
					equity = asum.GetEquity()
					margin = asum.GetMargin()
					freeMargin = asum.GetFreeMargin()
					marginLevel = marginLevelFromFields(asum.GetMarginLevel(), margin, equity)
					profitValue = equity - balance - creditOut
					if math.IsNaN(profitValue) || math.IsInf(profitValue, 0) {
						profitValue = asum.GetProfit()
						if math.IsNaN(profitValue) || math.IsInf(profitValue, 0) {
							profitValue = mt5StreamProfitFallback(upd)
						}
					}
				} else {
					if aerr != nil {
						logger.Debug("MT5 AccountSummary failed, account cards from OnOrderProfit frame",
							zap.String("account_id", account.ID.String()),
							zap.Error(aerr))
					}
					balance = upd.GetBalance()
					equity = upd.GetEquity()
					margin = upd.GetMargin()
					freeMargin = upd.GetFreeMargin()
					marginLevel = marginLevelFromFields(upd.GetMarginLevel(), margin, equity)
					profitValue = mt5StreamProfitFallback(upd)
					creditOut = 0
					if upd.Credit != nil {
						creditOut = *upd.Credit
						if math.IsNaN(creditOut) || math.IsInf(creditOut, 0) {
							creditOut = 0
						}
					}
				}

				profitEvent := &v1.ProfitUpdateEvent{
					AccountId:   accountStream.AccountID,
					Balance:     balance,
					Credit:      creditOut,
					Profit:      profitValue,
					Equity:      equity,
					Margin:      margin,
					FreeMargin:  freeMargin,
					MarginLevel: marginLevel,
					Orders:      nil,
					ProfitPercent: func() float64 {
						if balance <= 0 {
							return 0
						}
						return (profitValue / balance) * 100
					}(),
				}

				s.publishProfitEvent(accountStream.AccountID, profitEvent)
				accountStream.setProfitSnapshot(profitEvent)
			}
		}
	}
}
