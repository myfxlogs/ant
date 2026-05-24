// Package chmigrate provides embedded ClickHouse schema migrations.
// SQL files (*.sql) are embedded in the binary and executed in filename order.
// Versions are tracked in _schema_migrations for idempotent re-runs.
package chmigrate

import (
	"context"
	"crypto/sha256"
	"embed"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/ClickHouse/clickhouse-go/v2"
	"go.uber.org/zap"
)

//go:embed *.sql
var migrations embed.FS

// Run executes pending embedded .sql migrations against conn.
// Idempotent: skips versions already recorded in _schema_migrations
// with matching checksums; errors on checksum mismatch (schema drift).
func Run(ctx context.Context, conn clickhouse.Conn, log *zap.Logger) error {
	// Ensure tracking table exists first.
	if err := conn.Exec(ctx, `CREATE TABLE IF NOT EXISTS _schema_migrations (
		version    UInt32,
		name       String,
		applied_at DateTime64(3, 'UTC') DEFAULT now64(3),
		checksum   String
	) ENGINE = ReplacingMergeTree(applied_at) ORDER BY version`); err != nil {
		return fmt.Errorf("chmigrate: ensure _schema_migrations: %w", err)
	}

	entries, err := migrations.ReadDir(".")
	if err != nil {
		return fmt.Errorf("chmigrate: read embedded: %w", err)
	}

	// Filter and sort .sql files (excluding _schema_migrations DDL itself if embedded).
	var files []string
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".sql") || name == "_schema_migrations.sql" {
			continue
		}
		files = append(files, name)
	}
	sort.Strings(files)

	for _, name := range files {
		data, err := migrations.ReadFile(name)
		if err != nil {
			return fmt.Errorf("chmigrate: read %s: %w", name, err)
		}
		sql := strings.TrimSpace(string(data))
		if sql == "" {
			continue
		}

		// Extract version number from filename (e.g. "001_md_ticks.sql" → 1).
		parts := strings.SplitN(name, "_", 2)
		version, err := strconv.Atoi(parts[0])
		if err != nil {
			log.Warn("chmigrate: cannot parse version from filename", zap.String("file", name))
			continue
		}

		checksum := fmt.Sprintf("%x", sha256.Sum256(data))

		// Check if already applied.
		var count uint64
		row := conn.QueryRow(ctx,
			"SELECT count() FROM _schema_migrations WHERE version = @v AND checksum = @c",
			clickhouse.Named("v", version), clickhouse.Named("c", checksum))
		if err := row.Scan(&count); err != nil {
			log.Warn("chmigrate: version check failed", zap.Int("version", version), zap.Error(err))
			continue
		}
		if count > 0 {
			log.Debug("chmigrate: already applied", zap.Int("version", version), zap.String("file", name))
			continue
		}

		// Check for schema drift (same version, different checksum).
		row = conn.QueryRow(ctx,
			"SELECT count() FROM _schema_migrations WHERE version = @v AND checksum != @c",
			clickhouse.Named("v", version), clickhouse.Named("c", checksum))
		if err := row.Scan(&count); err == nil && count > 0 {
			return fmt.Errorf("chmigrate: schema drift on version %d (%s)", version, name)
		}

		// Apply migration (split on ; for multi-statement files like EXCHANGE TABLES sequences).
		for i, stmt := range splitSQL(sql) {
			if err := conn.Exec(ctx, stmt); err != nil {
				return fmt.Errorf("chmigrate: exec %s (stmt %d): %w", name, i+1, err)
			}
		}

		// Record version.
		if err := conn.Exec(ctx,
			"INSERT INTO _schema_migrations (version, name, checksum) VALUES (@v, @n, @c)",
			clickhouse.Named("v", version),
			clickhouse.Named("n", name),
			clickhouse.Named("c", checksum),
		); err != nil {
			return fmt.Errorf("chmigrate: record version %d: %w", version, err)
		}

		log.Info("chmigrate: applied", zap.Int("version", version), zap.String("file", name))
	}
	return nil
}

// splitSQL splits a multi-statement SQL string on ; delimiters,
// skipping empty statements and trimming whitespace.
func splitSQL(s string) []string {
	var stmts []string
	for _, part := range strings.Split(s, ";") {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			stmts = append(stmts, trimmed)
		}
	}
	return stmts
}

// MustRun panics on error.
func MustRun(ctx context.Context, conn clickhouse.Conn, log *zap.Logger) {
	if err := Run(ctx, conn, log); err != nil {
		log.Fatal("chmigrate: fatal", zap.Error(err))
	}
}
