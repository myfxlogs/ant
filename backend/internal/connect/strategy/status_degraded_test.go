package strategy

import (
	"context"
	"testing"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/repository"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"go.uber.org/zap"
)

func TestHasInvariantBlindSpot(t *testing.T) {
	tests := []struct {
		name   string
		spots  []*antv1.BlindSpot
		expect bool
	}{
		{
			name:   "nil blind spots",
			spots:  nil,
			expect: false,
		},
		{
			name:   "empty blind spots",
			spots:  []*antv1.BlindSpot{},
			expect: false,
		},
		{
			name: "non-invariant blind spot only",
			spots: []*antv1.BlindSpot{
				{Id: "some_other_issue", Category: "coverage"},
			},
			expect: false,
		},
		{
			name: "single invariant blind spot zero_volume_trade",
			spots: []*antv1.BlindSpot{
				{Id: "zero_volume_trade", Category: "invariant"},
			},
			expect: true,
		},
		{
			name: "single invariant blind spot capital_not_conserved",
			spots: []*antv1.BlindSpot{
				{Id: "capital_not_conserved", Category: "invariant"},
			},
			expect: true,
		},
		{
			name: "single invariant blind spot non_positive_price",
			spots: []*antv1.BlindSpot{
				{Id: "non_positive_price", Category: "invariant"},
			},
			expect: true,
		},
		{
			name: "single invariant blind spot invalid_side",
			spots: []*antv1.BlindSpot{
				{Id: "invalid_side", Category: "invariant"},
			},
			expect: true,
		},
		{
			name: "single invariant blind spot time_order_violation",
			spots: []*antv1.BlindSpot{
				{Id: "time_order_violation", Category: "invariant"},
			},
			expect: true,
		},
		{
			name: "invariant mixed with non-invariant",
			spots: []*antv1.BlindSpot{
				{Id: "some_other_issue", Category: "coverage"},
				{Id: "zero_volume_trade", Category: "invariant"},
				{Id: "another_coverage_gap", Category: "coverage"},
			},
			expect: true,
		},
		{
			name: "all non-invariant",
			spots: []*antv1.BlindSpot{
				{Id: "missing_indicator", Category: "coverage"},
				{Id: "unsupported_function", Category: "coverage"},
			},
			expect: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &antv1.ExecuteBacktestResponse{BlindSpots: tt.spots}
			got := hasInvariantBlindSpot(resp)
			if got != tt.expect {
				t.Errorf("hasInvariantBlindSpot() = %v, want %v", got, tt.expect)
			}
		})
	}
}

func TestHasInvariantBlindSpot_AllInvariantIDs(t *testing.T) {
	for _, id := range invariantBlindSpotIDs {
		resp := &antv1.ExecuteBacktestResponse{
			BlindSpots: []*antv1.BlindSpot{{Id: id, Category: "invariant"}},
		}
		if !hasInvariantBlindSpot(resp) {
			t.Errorf("expected hasInvariantBlindSpot=true for id=%s, got false", id)
		}
	}
}

func TestSaveBacktestResult_StatusDegraded(t *testing.T) {
	srv := &StrategyExecutionServer{
		backtestRepo: &repository.BacktestRunRepository{},
		log:          zap.NewNop(),
	}
	run := &repository.BacktestRun{
		ID:     uuid.New(),
		UserID: uuid.New(),
	}

	before := testutil.ToFloat64(BacktestRunsTotal.WithLabelValues(StatusDegraded))
	beforeSucc := testutil.ToFloat64(BacktestRunsTotal.WithLabelValues(StatusSucceeded))

	result := &antv1.ExecuteBacktestResponse{
		Success: true,
		BlindSpots: []*antv1.BlindSpot{
			{Id: "zero_volume_trade", Category: "invariant"},
		},
		Metrics: &antv1.ExecuteBacktestMetrics{},
	}

	srv.saveBacktestResult(context.Background(), run, result)

	after := testutil.ToFloat64(BacktestRunsTotal.WithLabelValues(StatusDegraded))
	afterSucc := testutil.ToFloat64(BacktestRunsTotal.WithLabelValues(StatusSucceeded))

	if after != before+1 {
		t.Errorf("expected DEGRADED metric to increment by 1: before=%v after=%v", before, after)
	}
	if afterSucc != beforeSucc {
		t.Errorf("expected SUCCEEDED metric to NOT change: before=%v after=%v", beforeSucc, afterSucc)
	}
}

func TestSaveBacktestResult_StatusSucceeded(t *testing.T) {
	srv := &StrategyExecutionServer{
		backtestRepo: &repository.BacktestRunRepository{},
		log:          zap.NewNop(),
	}
	run := &repository.BacktestRun{
		ID:     uuid.New(),
		UserID: uuid.New(),
	}

	beforeDeg := testutil.ToFloat64(BacktestRunsTotal.WithLabelValues(StatusDegraded))
	beforeSucc := testutil.ToFloat64(BacktestRunsTotal.WithLabelValues(StatusSucceeded))

	result := &antv1.ExecuteBacktestResponse{
		Success: true,
		BlindSpots: []*antv1.BlindSpot{
			{Id: "some_coverage_gap", Category: "coverage"},
		},
		Metrics: &antv1.ExecuteBacktestMetrics{},
	}

	srv.saveBacktestResult(context.Background(), run, result)

	afterDeg := testutil.ToFloat64(BacktestRunsTotal.WithLabelValues(StatusDegraded))
	afterSucc := testutil.ToFloat64(BacktestRunsTotal.WithLabelValues(StatusSucceeded))

	if afterDeg != beforeDeg {
		t.Errorf("expected DEGRADED metric to NOT change: before=%v after=%v", beforeDeg, afterDeg)
	}
	if afterSucc != beforeSucc+1 {
		t.Errorf("expected SUCCEEDED metric to increment by 1: before=%v after=%v", beforeSucc, afterSucc)
	}
}
