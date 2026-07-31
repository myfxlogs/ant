// partition_mgr.go — ensures PG partitioned tables have monthly partitions.
// Called at startup to create future partitions and drop expired ones.

package repository

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

var validPartitionName = regexp.MustCompile(`^[a-z_][a-z0-9_]*$`)

const (
	mdBarsRetentionDays = 730 // 2 years
	mdBarsLookAhead     = 6
)

// EnsureMarketDataPartitions creates missing monthly partitions for md_bars,
// and detaches partitions older than the retention period.
// ADR-0012: md_ticks partitions removed — table dropped.
func EnsureMarketDataPartitions(ctx context.Context, pool *pgxpool.Pool, log *zap.Logger) {
	now := time.Now().UTC()

	// md_bars: current month + 5 future months (bars are sparse, pre-create more)
	ensurePartitionsForTable(ctx, pool, log, "md_bars", "close_ts_unix_ms", now, mdBarsLookAhead, mdBarsRetentionDays)
}

func ensurePartitionsForTable(ctx context.Context, pool *pgxpool.Pool, log *zap.Logger,
	table, rangeCol string, now time.Time, lookAheadMonths, retentionDays int,
) {
	// Create past partitions (within retention window).
	// Without these, backfill of larger periods (1d/1w) fails because
	// returned bars fall before the earliest partition.
	pastMonths := (retentionDays / 30) + 1
	for i := -pastMonths; i < 0; i++ {
		monthStart := now.AddDate(0, i, 0)
		monthStart = time.Date(monthStart.Year(), monthStart.Month(), 1, 0, 0, 0, 0, time.UTC)
		createPartitionIfNotExists(ctx, pool, log, table, rangeCol, monthStart)
	}

	// Create future partitions.
	for i := 0; i < lookAheadMonths; i++ {
		monthStart := now.AddDate(0, i, 0)
		monthStart = time.Date(monthStart.Year(), monthStart.Month(), 1, 0, 0, 0, 0, time.UTC)
		createPartitionIfNotExists(ctx, pool, log, table, rangeCol, monthStart)
	}

	// Prune expired partitions.
	if retentionDays > 0 {
		cutoff := now.AddDate(0, 0, -retentionDays)
		cutoffMonth := time.Date(cutoff.Year(), cutoff.Month(), 1, 0, 0, 0, 0, time.UTC)
		prunePartitions(ctx, pool, log, table, rangeCol, cutoffMonth)
	}
}

func createPartitionIfNotExists(ctx context.Context, pool *pgxpool.Pool, log *zap.Logger,
	table, rangeCol string, monthStart time.Time,
) {
	monthEnd := monthStart.AddDate(0, 1, 0)
	partName := fmt.Sprintf("%s_%s", table, monthStart.Format("200601"))
	startMs := monthStart.UnixMilli()
	endMs := monthEnd.UnixMilli()

	// Check if partition already exists.
	var exists bool
	err := pool.QueryRow(ctx,
		`SELECT EXISTS (
			SELECT 1 FROM pg_class WHERE relname = $1
		)`, partName,
	).Scan(&exists)
	if err != nil {
		log.Warn("partition_mgr: check partition failed", zap.String("table", table), zap.String("partition", partName), zap.Error(err))
		return
	}
	if exists {
		return
	}

	// Create the partition.
	sql := fmt.Sprintf(`CREATE TABLE %s PARTITION OF %s FOR VALUES FROM (%d) TO (%d)`,
		partName, table, startMs, endMs)
	if _, err := pool.Exec(ctx, sql); err != nil {
		log.Warn("partition_mgr: create partition failed", zap.String("table", table), zap.String("partition", partName), zap.Error(err))
		return
	}
	log.Info("partition_mgr: created partition", zap.String("table", table), zap.String("partition", partName))
}

func prunePartitions(ctx context.Context, pool *pgxpool.Pool, log *zap.Logger,
	table, rangeCol string, beforeMonth time.Time,
) {
	rows, err := pool.Query(ctx,
		`SELECT c.relname
		 FROM pg_class c
		 JOIN pg_inherits i ON i.inhrelid = c.oid
		 JOIN pg_class p ON i.inhparent = p.oid
		 WHERE p.relname = $1 AND c.relname < $2
		 ORDER BY c.relname`,
		table, fmt.Sprintf("%s_%s", table, beforeMonth.Format("200601")),
	)
	if err != nil {
		log.Warn("partition_mgr: list partitions for prune failed", zap.String("table", table), zap.Error(err))
		return
	}
	defer rows.Close()

	var toDrop []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			continue
		}
		toDrop = append(toDrop, name)
	}
	for _, name := range toDrop {
		if !validPartitionName.MatchString(name) {
			log.Warn("partition_mgr: skipping invalid partition name", zap.String("name", name))
			continue
		}
		sql := fmt.Sprintf("DROP TABLE IF EXISTS %s", name)
		if _, err := pool.Exec(ctx, sql); err != nil {
			log.Warn("partition_mgr: drop partition failed", zap.String("table", name), zap.Error(err))
			continue
		}
		log.Info("partition_mgr: dropped expired partition", zap.String("table", name))
	}
}
