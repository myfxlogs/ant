package admin

import (
	"context"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"

	antv1 "alphaforge/gen/proto/ant/v1"
	antv1c "alphaforge/gen/proto/ant/v1/antv1connect"
)

// PlatformHealthServer implements ADR-0028 §5.3 Admin "Platform Health Center".
// Provides root cause reports clustering backtest failure signatures and
// health alert streaming via SSE (push-first on backtest status notifications).
type PlatformHealthServer struct {
	pool *pgxpool.Pool
	log  *zap.Logger
}

var _ antv1c.PlatformHealthServiceHandler = (*PlatformHealthServer)(nil)

func NewPlatformHealthServer(pool *pgxpool.Pool, log *zap.Logger) *PlatformHealthServer {
	return &PlatformHealthServer{pool: pool, log: log}
}

func (s *PlatformHealthServer) GetRootCauseReport(
	ctx context.Context,
	req *connect.Request[antv1.GetRootCauseReportRequest],
) (*connect.Response[antv1.RootCauseReport], error) {
	lookbackDays := int(req.Msg.GetLookbackDays())
	if lookbackDays <= 0 {
		lookbackDays = 14
	}

	now := time.Now().UTC()
	periodStart := now.AddDate(0, 0, -lookbackDays)
	prevStart := periodStart.AddDate(0, 0, -lookbackDays)

	// Summary counts for current and previous period.
	var totalRuns, totalDegraded, totalFailed int
	var prevDegraded, prevFailed int

	err := s.pool.QueryRow(ctx, `
		SELECT
			COUNT(*),
			COUNT(*) FILTER (WHERE status = 'DEGRADED'),
			COUNT(*) FILTER (WHERE status = 'FAILED')
		FROM backtest_runs
		WHERE created_at >= $1
	`, periodStart).Scan(&totalRuns, &totalDegraded, &totalFailed)
	if err != nil {
		return nil, fmt.Errorf("query current period: %w", err)
	}

	err = s.pool.QueryRow(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE status = 'DEGRADED'),
			COUNT(*) FILTER (WHERE status = 'FAILED')
		FROM backtest_runs
		WHERE created_at >= $1 AND created_at < $2
	`, prevStart, periodStart).Scan(&prevDegraded, &prevFailed)
	if err != nil {
		return nil, fmt.Errorf("query previous period: %w", err)
	}

	// Failure signature clusters.
	rows, err := s.pool.Query(ctx, `
		SELECT
			signature,
			severity,
			category,
			MAX(description) AS description,
			COUNT(DISTINCT backtest_run_id) AS affected_runs,
			COUNT(DISTINCT strategy_id) AS affected_strategies
		FROM backtest_failure_signatures
		WHERE created_at >= $1
		GROUP BY signature, severity, category
		ORDER BY affected_runs DESC
	`, periodStart)
	if err != nil {
		return nil, fmt.Errorf("query failure clusters: %w", err)
	}
	defer rows.Close()

	var clusters []*antv1.FailureCluster
	totalFailures := totalDegraded + totalFailed
	for rows.Next() {
		var sig, severity, category, desc string
		var affectedRuns, affectedStrategies int
		if err := rows.Scan(&sig, &severity, &category, &desc, &affectedRuns, &affectedStrategies); err != nil {
			return nil, fmt.Errorf("scan cluster row: %w", err)
		}
		impactPct := 0.0
		if totalFailures > 0 {
			impactPct = float64(affectedRuns) / float64(totalFailures) * 100
		}
		clusters = append(clusters, &antv1.FailureCluster{
			Signature:          sig,
			Severity:           severity,
			Description:        desc,
			AffectedRuns:       int32(affectedRuns),
			AffectedStrategies: int32(affectedStrategies),
			ImpactPct:          impactPct,
			Priority:           int32(len(clusters) + 1),
		})
	}

	// Recurrence rate: percentage of signatures that appeared in both periods.
	var prevSignatures int
	err = s.pool.QueryRow(ctx, `
		SELECT COUNT(DISTINCT signature)
		FROM backtest_failure_signatures
		WHERE created_at >= $1 AND created_at < $2
	`, prevStart, periodStart).Scan(&prevSignatures)
	if err != nil {
		s.log.Warn("failed to query previous signatures", zap.Error(err))
	}

	recurrenceRate := 0.0
	if len(clusters) > 0 && prevSignatures > 0 {
		// Count signatures present in both periods.
		var recurring int
		err = s.pool.QueryRow(ctx, `
			SELECT COUNT(DISTINCT a.signature)
			FROM backtest_failure_signatures a
			WHERE a.created_at >= $1
			AND EXISTS (
				SELECT 1 FROM backtest_failure_signatures b
				WHERE b.signature = a.signature
				AND b.created_at >= $2 AND b.created_at < $1
			)
		`, periodStart, prevStart).Scan(&recurring)
		if err != nil {
			s.log.Warn("failed to query recurring signatures", zap.Error(err))
		}
		recurrenceRate = float64(recurring) / float64(len(clusters)) * 100
	}

	degradedRate := 0.0
	failedRate := 0.0
	if totalRuns > 0 {
		degradedRate = float64(totalDegraded) / float64(totalRuns) * 100
		failedRate = float64(totalFailed) / float64(totalRuns) * 100
	}

	return connect.NewResponse(&antv1.RootCauseReport{
		GeneratedAt:       timestamppb.Now(),
		LookbackDays:      int32(lookbackDays),
		TotalBacktestRuns: int32(totalRuns),
		TotalDegraded:     int32(totalDegraded),
		TotalFailed:       int32(totalFailed),
		DegradedRatePct:   degradedRate,
		FailedRatePct:     failedRate,
		Clusters:          clusters,
		PrevTotalDegraded: int32(prevDegraded),
		PrevTotalFailed:   int32(prevFailed),
		RecurrenceRatePct: recurrenceRate,
	}), nil
}

func (s *PlatformHealthServer) WatchHealthAlerts(
	ctx context.Context,
	_ *connect.Request[antv1.WatchHealthAlertsRequest],
	stream *connect.ServerStream[antv1.HealthAlert],
) error {
	// Push-first: listen to backtest_status NOTIFY and push alerts on DEGRADED/FAILED.
	conn, err := s.pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire conn for LISTEN: %w", err)
	}
	defer conn.Release()

	if _, err := conn.Exec(ctx, "LISTEN backtest_status"); err != nil {
		return fmt.Errorf("LISTEN backtest_status: %w", err)
	}

	// Send initial snapshot of recent alerts.
	if err := s.sendRecentAlerts(ctx, stream); err != nil {
		s.log.Warn("failed to send recent alerts", zap.Error(err))
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			notif, err := conn.Conn().WaitForNotification(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return nil
				}
				return fmt.Errorf("wait for notification: %w", err)
			}
			alert := parseBacktestNotification(notif.Payload)
			if alert == nil {
				continue
			}
			if err := stream.Send(alert); err != nil {
				return fmt.Errorf("send health alert: %w", err)
			}
		}
	}
}

func (s *PlatformHealthServer) sendRecentAlerts(ctx context.Context, stream *connect.ServerStream[antv1.HealthAlert]) error {
	rows, err := s.pool.Query(ctx, `
		SELECT signature, severity, description, COUNT(*) AS cnt
		FROM backtest_failure_signatures
		WHERE created_at >= now() - INTERVAL '24 hours'
		GROUP BY signature, severity, description
		ORDER BY cnt DESC
		LIMIT 10
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var sig, severity, desc string
		var cnt int
		if err := rows.Scan(&sig, &severity, &desc, &cnt); err != nil {
			return err
		}
		alertType := "statistical"
		if severity == "致命" {
			alertType = "invariant"
		}
		if err := stream.Send(&antv1.HealthAlert{
			Timestamp:     timestamppb.Now(),
			AlertType:     alertType,
			Signature:     sig,
			Description:   desc,
			AffectedCount: int32(cnt),
		}); err != nil {
			return err
		}
	}
	return nil
}

// parseBacktestNotification extracts a HealthAlert from a PG NOTIFY payload.
// Returns nil for non-failure notifications (SUCCEEDED, etc.).
func parseBacktestNotification(payload string) *antv1.HealthAlert {
	if payload == "" {
		return nil
	}
	alertType := "degraded"
	if strings.Contains(payload, "FAILED") {
		alertType = "failed"
	} else if !strings.Contains(payload, "DEGRADED") {
		return nil
	}
	return &antv1.HealthAlert{
		Timestamp:   timestamppb.Now(),
		AlertType:   alertType,
		Description: payload,
	}
}
