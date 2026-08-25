// decay_detector_batch.go — Batch detection and metrics helpers extracted from decay_detector.go.
package marketplace

import (
	"context"
	"fmt"
	"sync"

	"golang.org/x/sync/errgroup"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

func computePeriodMetrics(rows []dailyRow) periodMetrics {
	var m periodMetrics
	if len(rows) == 0 {
		return m
	}

	// Total return = sum of daily returns.
	for _, r := range rows {
		m.totalReturn = m.totalReturn.Add(r.dailyReturn)
	}
	m.avgDailyReturn = m.totalReturn.Div(decimal.NewFromInt(int64(len(rows))))

	// Win rate.
	var totalTrades, winningTrades int32
	for _, r := range rows {
		totalTrades += r.totalTrades
		winningTrades += r.winningTrades
	}
	if totalTrades > 0 {
		wr := decimal.NewFromInt(int64(winningTrades)).Div(decimal.NewFromInt(int64(totalTrades)))
		m.winRate = &wr
	}

	// Sharpe ratio (simplified): mean(dailyReturn) / stddev(dailyReturn) * sqrt(252).
	if len(rows) > 5 {
		mean := m.avgDailyReturn
		var sumSqDiff decimal.Decimal
		for _, r := range rows {
			diff := r.dailyReturn.Sub(mean)
			sumSqDiff = sumSqDiff.Add(diff.Mul(diff))
		}
		variance := sumSqDiff.Div(decimal.NewFromInt(int64(len(rows))))
		if variance.GreaterThan(decimal.Zero) {
			stddev := decimalSqrt(variance)
			if stddev.GreaterThan(decimal.Zero) {
				sr := mean.Div(stddev).Mul(decimal.NewFromFloat(15.8745)) // sqrt(252)
				m.sharpe = &sr
			}
		}
	}

	return m
}

func formatDecayReason(r *DecayResult) string {
	if !r.IsDecaying {
		return "no significant decay detected"
	}
	reasons := ""
	if r.SharpeSignal {
		reasons += "sharpe ratio declined significantly; "
	}
	if r.WinRateSignal {
		reasons += "win rate dropped; "
	}
	if r.ReturnSignal {
		reasons += "recent returns below expected; "
	}
	if reasons == "" {
		reasons = "multiple decay signals detected"
	}
	return reasons
}

// DetectDecayBatch runs decay detection for all published strategies with
// linked live accounts. Returns only strategies flagged as decaying.
// Called from the optimization trigger (push-first, not a cron).
func (s *Service) DetectDecayBatch(ctx context.Context) ([]*DecayResult, error) {
	rows, err := s.pg.Query(ctx,
		`SELECT strategy_id::text FROM marketplace_strategies
		 WHERE status = 'published' AND linked_account_id IS NOT NULL`)
	if err != nil {
		return nil, fmt.Errorf("marketplace: detect decay batch: %w", err)
	}
	defer rows.Close()

	var strategyIDs []string
	for rows.Next() {
		var sid string
		if err := rows.Scan(&sid); err != nil {
			continue
		}
		strategyIDs = append(strategyIDs, sid)
	}
	rows.Close()

	// Load thresholds once for all strategies (avoids N config queries).
	th := s.loadDecayThresholds(ctx)

	var mu sync.Mutex
	var results []*DecayResult
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(8) // limit concurrency to avoid overwhelming the pool

	for _, sid := range strategyIDs {
		sid := sid
		g.Go(func() error {
			sidUUID, err := uuid.Parse(sid)
			if err != nil {
				return nil
			}
			r, err := s.detectDecayWithThresholds(gctx, sidUUID, th)
			if err != nil {
				s.log.Warn("detect decay: strategy failed", zap.String("strategy_id", sid), zap.Error(err))
				return nil
			}
			if r.IsDecaying {
				mu.Lock()
				results = append(results, r)
				mu.Unlock()
			}
			return nil
		})
	}
	// All goroutines log errors internally and return nil (partial failures
	// don't abort the batch). g.Wait() respects errgroup context cancellation.
	if err := g.Wait(); err != nil {
		s.log.Warn("detect decay batch: errgroup", zap.Error(err))
	}
	return results, nil
}
