package connect

import (
	"context"
	"fmt"
	"strings"
	"time"

	v1 "anttrader/gen/proto"
	"anttrader/internal/model"
	"anttrader/pkg/logger"

	"github.com/google/uuid"

	"go.uber.org/zap"
)

func parseScheduleIDFromOrderComment(comment string) *uuid.UUID {
	comment = strings.TrimSpace(comment)
	if comment == "" {
		return nil
	}
	// Expected prefix: schedule:<uuid>|...
	const prefix = "schedule:"
	idx := strings.Index(comment, prefix)
	if idx < 0 {
		return nil
	}
	s := comment[idx+len(prefix):]
	sep := strings.IndexAny(s, "| ")
	if sep >= 0 {
		s = s[:sep]
	}
	id, err := uuid.Parse(strings.TrimSpace(s))
	if err != nil {
		return nil
	}
	return &id
}

func (s *StreamService) startOrderStream(accountStream *AccountStream, account *model.MTAccount) {
	s.startOrderStreamWithCtx(accountStream.Ctx, accountStream, account)
}

func (s *StreamService) startOrderStreamWithCtx(ctx context.Context, accountStream *AccountStream, account *model.MTAccount) {
	switch strings.ToUpper(account.MTType) {
	case "MT4":
		conn, err := s.connManager.GetMT4Connection(account.ID)
		if err != nil {
			logger.Error("获取 MT4 连接失败", zap.String("account_id", account.ID.String()), zap.Error(err))
			return
		}

		mt4OrderCh := conn.GetOrderChannel()
		if mt4OrderCh == nil {
			logger.Error("MT4 订单通道为空", zap.String("account_id", account.ID.String()))
			return
		}

		prevBalance := float64(0)
		havePrevBalance := false
		prevCredit := float64(0)
		havePrevCredit := false

		for {
			select {
			case <-ctx.Done():
				return
			case mt4Order, ok := <-mt4OrderCh:
				if !ok {
					return
				}
				upd := mt4Order.GetUpdate()
				if upd == nil {
					continue
				}

				action := upd.GetAction()
				if (action.String() == "UpdateAction_Balance" || action.String() == "UpdateAction_Credit") && upd.GetOrder() == nil {
					bal := mt4Order.GetBalance()
					cred := mt4Order.GetCredit()
					if action.String() == "UpdateAction_Balance" {
						delta := float64(0)
						if havePrevBalance {
							delta = bal - prevBalance
						}
						prevBalance = bal
						havePrevBalance = true
						s.publishLedgerEntryEvent(accountStream.AccountID, &v1.LedgerEntryEvent{
							AccountId:     accountStream.AccountID,
							EntryType:     "BALANCE",
							Amount:        delta,
							Currency:      mt4Order.GetCurrency(),
							Time:          time.Now().Unix(),
							Comment:       fmt.Sprintf("balance=%.2f credit=%.2f", bal, cred),
							RelatedTicket: 0,
						})
						accountStream.SetLedgerSnapshot(&v1.LedgerEntryEvent{
							AccountId:     accountStream.AccountID,
							EntryType:     "BALANCE",
							Amount:        delta,
							Currency:      mt4Order.GetCurrency(),
							Time:          time.Now().Unix(),
							Comment:       fmt.Sprintf("balance=%.2f credit=%.2f", bal, cred),
							RelatedTicket: 0,
						})
					} else {
						delta := float64(0)
						if havePrevCredit {
							delta = cred - prevCredit
						}
						prevCredit = cred
						havePrevCredit = true
						s.publishLedgerEntryEvent(accountStream.AccountID, &v1.LedgerEntryEvent{
							AccountId:     accountStream.AccountID,
							EntryType:     "CREDIT",
							Amount:        delta,
							Currency:      mt4Order.GetCurrency(),
							Time:          time.Now().Unix(),
							Comment:       fmt.Sprintf("balance=%.2f credit=%.2f", bal, cred),
							RelatedTicket: 0,
						})
						accountStream.SetLedgerSnapshot(&v1.LedgerEntryEvent{
							AccountId:     accountStream.AccountID,
							EntryType:     "CREDIT",
							Amount:        delta,
							Currency:      mt4Order.GetCurrency(),
							Time:          time.Now().Unix(),
							Comment:       fmt.Sprintf("balance=%.2f credit=%.2f", bal, cred),
							RelatedTicket: 0,
						})
					}
					continue
				}

				order := upd.GetOrder()
				if order == nil {
					continue
				}

				orderEvent := &v1.OrderUpdateEvent{
					AccountId: accountStream.AccountID,
					Ticket:    int64(order.Ticket),
					Action:    action.String(),
					Symbol:    order.Symbol,
					Volume:    order.Lots,
					OpenPrice: order.OpenPrice,
				}

				s.publishOrderEvent(accountStream.AccountID, orderEvent)

				// v2 official-like: position updates (MT4: action indicates position lifecycle).
				openAt := int64(0)
				closeAt := int64(0)
				if order.OpenTime != nil {
					openAt = order.OpenTime.AsTime().Unix()
				}
				if order.CloseTime != nil {
					closeAt = order.CloseTime.AsTime().Unix()
				}
				posEv := &v1.PositionUpdateEvent{
					AccountId:       accountStream.AccountID,
					PositionTicket:  int64(order.Ticket),
					Symbol:          order.Symbol,
					Action:          action.String(),
					Volume:          order.Lots,
					OpenPrice:       order.OpenPrice,
					ClosePrice:      order.ClosePrice,
					Profit:          order.Profit,
					StopLoss:        order.StopLoss,
					TakeProfit:      order.TakeProfit,
					OpenTime:        openAt,
					CloseTime:       closeAt,
					Comment:         order.Comment,
				}
				s.publishPositionEvent(accountStream.AccountID, posEv)
				accountStream.UpsertPosition(posEv)

				if action.String() == "TradeActionClose" || action.String() == "TradeActionCloseBy" {
					if s.tradeRecordRepo != nil {
						ticket := int64(order.Ticket)
						go func() {
							for attempt := 0; attempt < 8; attempt++ {
								fromStr := time.Now().Add(-30 * 24 * time.Hour).Format("2006-01-02T15:04:05")
								toStr := time.Now().Add(10 * time.Minute).Format("2006-01-02T15:04:05")
								historyOrders, err := conn.OrderHistory(context.Background(), fromStr, toStr)
								if err != nil {
									logger.Warn("获取历史订单失败", zap.String("account_id", account.ID.String()), zap.Error(err))
								} else {
									for _, histOrder := range historyOrders {
										if int64(histOrder.Ticket) != ticket {
											continue
										}
										if histOrder.CloseTime == nil {
											continue
										}
										tradeRecord := &model.TradeRecord{
											ScheduleID:   parseScheduleIDFromOrderComment(histOrder.Comment),
											AccountID:    account.ID,
											Ticket:       int64(histOrder.Ticket),
											Symbol:       histOrder.Symbol,
											OrderType:    getOrderTypeString(int16(histOrder.Type)),
											Volume:       histOrder.Lots,
											OpenPrice:    histOrder.OpenPrice,
											ClosePrice:   histOrder.ClosePrice,
											Profit:       histOrder.Profit,
											Swap:         histOrder.Swap,
											Commission:   histOrder.Commission,
											OpenTime:     histOrder.OpenTime.AsTime(),
											CloseTime:    histOrder.CloseTime.AsTime(),
											StopLoss:     histOrder.StopLoss,
											TakeProfit:   histOrder.TakeProfit,
											OrderComment: histOrder.Comment,
											MagicNumber:  int(histOrder.MagicNumber),
											Platform:     "MT4",
										}
										if err := s.tradeRecordRepo.Create(context.Background(), tradeRecord); err != nil {
											logger.Warn("保存历史订单失败", zap.String("account_id", account.ID.String()), zap.Int64("ticket", int64(histOrder.Ticket)), zap.Error(err))
										} else {
											return
										}
									}
								}
								time.Sleep(2 * time.Second)
							}
						}()
					}
				}

				accountStream.upsertOpenedOrder(orderEvent)
			}
		}

	case "MT5":
		conn, err := s.connManager.GetMT5Connection(account.ID)
		if err != nil {
			logger.Error("获取 MT5 连接失败", zap.String("account_id", account.ID.String()), zap.Error(err))
			return
		}

		mt5OrderCh := conn.GetOrderChannel()
		if mt5OrderCh == nil {
			logger.Error("MT5 订单通道为空", zap.String("account_id", account.ID.String()))
			return
		}

		for {
			select {
			case <-ctx.Done():
				return
			case mt5Order, ok := <-mt5OrderCh:
				if !ok {
					return
				}
				upd := mt5Order.GetUpdate()
				if upd == nil || upd.GetOrder() == nil {
					continue
				}

				order := upd.GetOrder()
				action := upd.GetType()

				orderEvent := &v1.OrderUpdateEvent{
					AccountId: accountStream.AccountID,
					Ticket:    order.Ticket,
					Action:    action.String(),
					Symbol:    order.Symbol,
					Volume:    order.Lots,
					OpenPrice: float64(order.OpenPrice),
				}

				s.publishOrderEvent(accountStream.AccountID, orderEvent)

				// v2 official-like: deal + position updates (MT5 deals include PositionTicket).
				if deal := upd.GetDeal(); deal != nil {
					dealEv := &v1.DealUpdateEvent{
						AccountId:       accountStream.AccountID,
						DealTicket:      deal.TicketNumber,
						PositionTicket:  deal.PositionTicket,
						Symbol:          deal.Symbol,
						DealType:        deal.Type.String(),
						Direction:       deal.Direction.String(),
						Volume:          deal.Lots,
						Price:           deal.Price,
						Profit:          deal.Profit,
						Swap:            deal.Swap,
						Commission:      deal.Commission,
						Time:            deal.OpenTime,
						Comment:         deal.Comment,
						ExpertId:        deal.ExpertId,
					}
					s.publishDealEvent(accountStream.AccountID, dealEv)
					accountStream.UpsertDeal(dealEv)

					if ledEv := BuildMT5LedgerEntryEvent(accountStream.AccountID, deal, upd.GetTrans()); ledEv != nil {
						s.publishLedgerEntryEvent(accountStream.AccountID, ledEv)
						accountStream.SetLedgerSnapshot(ledEv)
					}

					posEv := &v1.PositionUpdateEvent{
						AccountId:       accountStream.AccountID,
						PositionTicket:  deal.PositionTicket,
						Symbol:          deal.Symbol,
						Action:          action.String(),
						Volume:          deal.Lots,
						OpenPrice:       deal.OpenPrice,
						ClosePrice:      deal.Price,
						Profit:          deal.Profit,
						StopLoss:        deal.StopLoss,
						TakeProfit:      deal.TakeProfit,
						OpenTime:        deal.OpenTime,
						CloseTime:       0,
						Comment:         deal.Comment,
					}
					s.publishPositionEvent(accountStream.AccountID, posEv)
					accountStream.UpsertPosition(posEv)
				}

				if action.String() == "OrderTypeDeal" || action.String() == "OrderTypeBalance" {
					if s.tradeRecordRepo != nil {
						ticket := order.Ticket
						go func() {
							for attempt := 0; attempt < 8; attempt++ {
								fromStr := time.Now().Add(-30 * 24 * time.Hour).Format("2006-01-02T15:04:05")
								toStr := time.Now().Add(10 * time.Minute).Format("2006-01-02T15:04:05")
								historyOrders, err := conn.OrderHistory(context.Background(), fromStr, toStr)
								if err != nil {
									logger.Warn("获取历史订单失败", zap.String("account_id", account.ID.String()), zap.Error(err))
								} else {
									for _, histOrder := range historyOrders {
										if histOrder.Ticket != ticket {
											continue
										}
										if histOrder.CloseTime == nil {
											continue
										}
										tradeRecord := &model.TradeRecord{
											ScheduleID:   parseScheduleIDFromOrderComment(histOrder.Comment),
											AccountID:    account.ID,
											Ticket:       histOrder.Ticket,
											Symbol:       histOrder.Symbol,
											OrderType:    getOrderTypeString(int16(histOrder.OrderType)),
											Volume:       histOrder.Lots,
											OpenPrice:    float64(histOrder.OpenPrice),
											ClosePrice:   float64(histOrder.ClosePrice),
											Profit:       float64(histOrder.Profit),
											Swap:         float64(histOrder.Swap),
											Commission:   float64(histOrder.Commission),
											OpenTime:     histOrder.OpenTime.AsTime(),
											CloseTime:    histOrder.CloseTime.AsTime(),
											StopLoss:     float64(histOrder.StopLoss),
											TakeProfit:   float64(histOrder.TakeProfit),
											OrderComment: histOrder.Comment,
											MagicNumber:  int(histOrder.ExpertId),
											Platform:     "MT5",
										}
										if err := s.tradeRecordRepo.Create(context.Background(), tradeRecord); err != nil {
											logger.Warn("保存历史订单失败", zap.String("account_id", account.ID.String()), zap.Int64("ticket", histOrder.Ticket), zap.Error(err))
										} else {
											return
										}
									}
								}
								time.Sleep(2 * time.Second)
							}
						}()
					}
				}

				accountStream.upsertOpenedOrder(orderEvent)
			}
		}
	}
}
