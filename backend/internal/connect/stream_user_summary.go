package connect

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "anttrader/gen/proto"
	"anttrader/internal/model"
)

func (s *StreamService) SubscribeUserSummary(ctx context.Context, _ *connect.Request[emptypb.Empty], stream *connect.ServerStream[v1.UserSummaryEvent]) error {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return err
	}

	accounts, err := s.accountRepo.GetByUserID(ctx, userID)
	if err != nil {
		return connect.NewError(connect.CodeInternal, fmt.Errorf("获取账户列表失败: %w", err))
	}

	// S1: only enabled accounts are eligible for summary subscription
	enabled := make([]*model.MTAccount, 0, len(accounts))
	for _, a := range accounts {
		if a == nil {
			continue
		}
		if a.IsDisabled {
			continue
		}
		enabled = append(enabled, a)
	}

	type snap struct {
		balance float64
		equity  float64
		profit  float64
	}

	latest := make(map[string]snap, len(enabled))
	var latestMu sync.Mutex
	updates := make(chan struct{}, 1)
	statsUpdates := make(chan struct{}, 1)

	var statsMu sync.Mutex
	var stats userSummaryStats

	unsubscribeFns := make([]func(), 0, len(enabled))
	defer func() {
		for _, fn := range unsubscribeFns {
			if fn != nil {
				fn()
			}
		}
	}()

	for _, account := range enabled {
		accountIDStr := account.ID.String()

		// Ownership guard (defense in depth)
		if account.UserID != userID {
			return connect.NewError(connect.CodePermissionDenied, errors.New("无权限访问该账户"))
		}

		latestMu.Lock()
		latest[accountIDStr] = snap{
			balance: account.Balance,
			equity:  account.Equity,
			profit:  account.Equity - account.Balance - account.Credit,
		}
		latestMu.Unlock()

		err = s.ensureAccountStream(ctx, account)
		if err != nil {
			return err
		}

		// MT-official-like: do NOT register an internal subscriber per account for summary.
		// Instead, consume internal profit snapshot notifications emitted by the supervisor workers.
		_, err = s.goroutineMgr.Spawn("user-summary-snapshot-"+accountIDStr, func(goroutineCtx context.Context) error {
			// Small fallback tick in case notifications are missed during races.
			fallback := time.NewTicker(5 * time.Second)
			defer fallback.Stop()
			for {
				select {
				case <-goroutineCtx.Done():
					return goroutineCtx.Err()
				case <-ctx.Done():
					return ctx.Err()
				case <-fallback.C:
					// refresh
				case <-func() <-chan struct{} {
					accountStream, ok := s.getAccountStream(accountIDStr)
					if !ok || accountStream == nil {
						return nil
					}
					return accountStream.profitNotifyCh()
				}():
					// notified
				}

				accountStream, ok := s.getAccountStream(accountIDStr)
				if !ok || accountStream == nil {
					continue
				}
				p := accountStream.getProfitSnapshot()
				if p == nil {
					continue
				}
				latestMu.Lock()
				latest[accountIDStr] = snap{balance: p.Balance, equity: p.Equity, profit: p.Profit}
				latestMu.Unlock()
				select {
				case updates <- struct{}{}:
				default:
				}
			}
		})
		if err != nil {
			return connect.NewError(connect.CodeInternal, fmt.Errorf("failed to start user summary worker: %w", err))
		}
	}

	if s.analyticsRepo != nil {
		_, err = s.goroutineMgr.Spawn("user-summary-stats", func(goroutineCtx context.Context) error {
			tick := time.NewTicker(30 * time.Second)
			defer tick.Stop()

			compute := func(at time.Time) {
				p, err := s.computeUserSummaryStats(goroutineCtx, userID, at)
				if err != nil {
					return
				}
				statsMu.Lock()
				stats = p
				statsMu.Unlock()
				select {
				case statsUpdates <- struct{}{}:
				default:
				}
			}

			compute(time.Now())
			for {
				select {
				case <-goroutineCtx.Done():
					return goroutineCtx.Err()
				case <-ctx.Done():
					return ctx.Err()
				case t := <-tick.C:
					compute(t)
				}
			}
		})
		if err != nil {
			return connect.NewError(connect.CodeInternal, fmt.Errorf("failed to start user summary stats worker: %w", err))
		}
	}

	// Send initial snapshot (even if we have no profit data yet)
	select {
	case updates <- struct{}{}:
	default:
	}

	keepAlive := time.NewTicker(20 * time.Second)
	defer keepAlive.Stop()

	// Aggregation and streaming loop
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-updates:
		case <-statsUpdates:
		case <-keepAlive.C:
		}

		var totalBalance float64
		var totalEquity float64
		var totalProfit float64
		connectedCount := int32(0)
		latestMu.Lock()
		connectedCount = int32(len(latest))
		for _, s0 := range latest {
			totalBalance += s0.balance
			totalEquity += s0.equity
			totalProfit += s0.profit
		}
		latestMu.Unlock()

		statsMu.Lock()
		st := stats
		statsMu.Unlock()

		e := &v1.UserSummaryEvent{
			TotalBalance:         totalBalance,
			TotalEquity:          totalEquity,
			TotalProfit:          totalProfit,
			AccountCount:         int32(len(enabled)),
			ConnectedCount:       connectedCount,
			UpdatedAt:            timestamppb.Now(),
			PnlToday:             st.pnlToday,
			PnlWeek:              st.pnlWeek,
			PnlMonth:             st.pnlMonth,
			TradesToday:          st.tradesToday,
			TradesWeek:           st.tradesWeek,
			TradesMonth:          st.tradesMonth,
			WinRate:              st.winRate,
			ProfitFactor:         st.profitFactor,
			MaxDrawdownPercent:   st.maxDrawdownPercent,
			MaxConsecutiveWins:   st.maxConsecWins,
			MaxConsecutiveLosses: st.maxConsecLosses,
		}
		if err := stream.Send(e); err != nil {
			return err
		}
	}
}

type dailyPnLPoint struct {
	Date time.Time
	PnL  float64
}

func computeMaxDrawdownPercentFromDailyPnL(points []dailyPnLPoint) float64 {
	running := 0.0
	runningMax := 0.0
	maxDrawdown := 0.0
	for _, p := range points {
		running += p.PnL
		if running > runningMax {
			runningMax = running
		}
		drawdown := running - runningMax
		if drawdown < maxDrawdown {
			maxDrawdown = drawdown
		}
	}
	if runningMax <= 0 {
		return 0
	}
	return (maxDrawdown / runningMax) * 100
}

func (s *StreamService) computeUserSummaryStats(ctx context.Context, userID uuid.UUID, at time.Time) (userSummaryStats, error) {
	loc := at.Location()
	todayStart := time.Date(at.Year(), at.Month(), at.Day(), 0, 0, 0, 0, loc)
	weekStart := at.AddDate(0, 0, -7)
	monthStart := at.AddDate(0, 0, -30)
	end := at

	pnlToday, tradesToday, _, _, _, _, err := s.analyticsRepo.GetUserPnLTradesWinLoss(ctx, userID, todayStart, end, true)
	if err != nil {
		return userSummaryStats{}, err
	}
	pnlWeek, tradesWeek, _, _, _, _, err := s.analyticsRepo.GetUserPnLTradesWinLoss(ctx, userID, weekStart, end, true)
	if err != nil {
		return userSummaryStats{}, err
	}
	pnlMonth, tradesMonth, winTradesMonth, lossTradesMonth, sumProfitPos, sumLossAbs, err := s.analyticsRepo.GetUserPnLTradesWinLoss(ctx, userID, monthStart, end, true)
	if err != nil {
		return userSummaryStats{}, err
	}

	dailyRepo, err := s.analyticsRepo.GetUserDailyPnL(ctx, userID, monthStart, end, true)
	if err != nil {
		return userSummaryStats{}, err
	}
	daily := make([]dailyPnLPoint, 0, len(dailyRepo))
	for _, dp := range dailyRepo {
		daily = append(daily, dailyPnLPoint{Date: dp.Date, PnL: dp.PnL})
	}

	maxWins, maxLosses, err := s.analyticsRepo.GetUserConsecutiveStats(ctx, userID, monthStart, end, true)
	if err != nil {
		return userSummaryStats{}, err
	}

	winRate := 0.0
	if (winTradesMonth + lossTradesMonth) > 0 {
		winRate = float64(winTradesMonth) / float64(winTradesMonth+lossTradesMonth) * 100
	}

	profitFactor := 0.0
	if sumLossAbs > 0 {
		profitFactor = sumProfitPos / sumLossAbs
	}

	return userSummaryStats{
		pnlToday:           pnlToday,
		pnlWeek:            pnlWeek,
		pnlMonth:           pnlMonth,
		tradesToday:        int32(tradesToday),
		tradesWeek:         int32(tradesWeek),
		tradesMonth:        int32(tradesMonth),
		winRate:            winRate,
		profitFactor:       profitFactor,
		maxDrawdownPercent: computeMaxDrawdownPercentFromDailyPnL(daily),
		maxConsecWins:      int32(maxWins),
		maxConsecLosses:    int32(maxLosses),
	}, nil
}
