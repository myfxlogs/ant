// Package knowledgebase implements the K1 compatibility knowledge base.
//
// It consolidates MQL compatibility facts (constants, functions, indicators)
// and L0 deterministic fixes (aliases, renames) from scattered Go maps
// (constants.go, api_registry.go, compat_fixes.go) into a unified PG-backed
// store with an in-memory cache for zero-latency lookups on the hot path.
//
// Architecture:
//   - Start(): full load from PG → in-memory cache; LISTEN for invalidation
//   - Lookup*(): read from cache only (0 PG queries per constant during compile)
//   - Record*(): INSERT to PG + pg_notify → LISTEN → cache refresh (push-first)
//   - interp.SetKB*Hooks(): wired at startup so compiler/VM use KB-first lookup
//
// C1 deterministic compound interest: new compat knowledge recorded via
// RecordFact/RecordFix is immediately available to the compiler without
// restart, recompilation, or LLM calls — 0 tokens, 0 drift.
package knowledgebase

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"alphaforge/internal/pglisten"
	"alphaforge/tools/mql2go/interp"
)

// notifyChannel is the PG LISTEN/NOTIFY channel for KB cache invalidation.
const notifyChannel = "kb_compat_update"

// FactRecord represents a compatibility fact entry for writing.
type FactRecord struct {
	Identifier    string
	Kind          string // "constant" | "function" | "indicator"
	Status        string // "supported" | "unsupported"
	Severity      string // "fatal" | "warning" | "info"
	ValueText     *string
	ValueNumeric  *int32
	MappingTarget *string
	Source        string
}

// FixRecord represents a compatibility fix entry for writing.
type FixRecord struct {
	Pattern          string
	FixType          string // "alias" | "rename" | "normalize"
	ResolutionTarget string
	Source           string
}

type funcInfo struct {
	supported bool
	severity  string
}

// Service is the KB compatibility knowledge service.
// It maintains an in-memory cache loaded from PG at startup,
// invalidated via LISTEN/NOTIFY when new facts/fixes are recorded.
type Service struct {
	pool     *pgxpool.Pool
	pgListen *pglisten.Listener
	log      *zap.Logger

	mu        sync.RWMutex
	constants map[string]interp.Value // identifier → resolved value
	fixes     map[string]string       // pattern → canonical name
	functions map[string]funcInfo     // name → status/severity

	// loadFromDB is overrideable for testing.
	loadFromDB func(ctx context.Context) error
}

// New creates a KB Service backed by the given PG pool and pglisten.
func New(pool *pgxpool.Pool, pgListen *pglisten.Listener, log *zap.Logger) *Service {
	s := &Service{
		pool:      pool,
		pgListen:  pgListen,
		log:       log,
		constants: make(map[string]interp.Value),
		fixes:     make(map[string]string),
		functions: make(map[string]funcInfo),
	}
	s.loadFromDB = s.loadFromDBImpl
	return s
}

// Start loads the cache from PG and begins listening for invalidation.
// If the tables are empty, runs Seed first.
// Must be called before any compilation occurs.
func (s *Service) Start(ctx context.Context) error {
	if err := s.loadFromDB(ctx); err != nil {
		return err
	}

	// Auto-seed if tables are empty.
	if len(s.constants) == 0 && len(s.functions) == 0 {
		s.log.Info("kb: tables empty, running seed")
		if err := s.Seed(ctx); err != nil {
			return err
		}
		if err := s.loadFromDB(ctx); err != nil {
			return err
		}
	}

	s.log.Info("kb: cache loaded",
		zap.Int("constants", len(s.constants)),
		zap.Int("fixes", len(s.fixes)),
		zap.Int("functions", len(s.functions)),
	)

	// Wire hooks so compiler/VM use KB-first lookup.
	interp.SetKBConstantLookup(s.LookupConstant)
	interp.SetKBFixLookup(s.LookupFix)
	interp.SetKBFunctionLookup(s.LookupFunction)

	// Start LISTEN for push-first cache invalidation.
	// Register LISTEN synchronously so that RecordFact/RecordFix calls
	// immediately after Start() returns will not race with LISTEN registration
	// (pg_notify before LISTEN = dropped notification = stale cache).
	notifCh, listenCancel, err := s.pgListen.Listen(ctx, notifyChannel)
	if err != nil {
		return fmt.Errorf("kb: LISTEN %s: %w", notifyChannel, err)
	}
	go s.listenLoop(ctx, notifCh, listenCancel)

	return nil
}

// listenLoop reads notifications from the registered LISTEN channel and
// refreshes the cache on each notification. Push-first: no polling.
// LISTEN registration is done synchronously in Start() before launching
// this goroutine, so there is no race between LISTEN and pg_notify.
func (s *Service) listenLoop(ctx context.Context, notifCh <-chan string, listenCancel context.CancelFunc) {
	defer listenCancel()

	for {
		select {
		case <-ctx.Done():
			return
		case _, ok := <-notifCh:
			if !ok {
				return
			}
			if err := s.loadFromDB(ctx); err != nil {
				s.log.Warn("kb: cache refresh failed", zap.Error(err))
			}
		}
	}
}

// LookupConstant returns the resolved Value for a constant identifier.
// Reads from in-memory cache only — 0 PG queries on the hot path.
func (s *Service) LookupConstant(name string) (interp.Value, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if v, ok := s.constants[name]; ok {
		return v, true
	}
	return interp.NoneVal(), false
}

// LookupFix returns the canonical name for an alias identifier.
// Reads from in-memory cache only.
func (s *Service) LookupFix(name string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	canonical, ok := s.fixes[name]
	return canonical, ok
}

// LookupFunction returns (supported, severity) for a function/indicator name.
// Reads from in-memory cache only.
func (s *Service) LookupFunction(name string) (bool, string) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	info, ok := s.functions[name]
	if !ok {
		return false, ""
	}
	return info.supported, info.severity
}

// RecordFact inserts a new compatibility fact into PG and sends a NOTIFY
// to refresh all caches. This is the only entry point for new compat knowledge.
func (s *Service) RecordFact(ctx context.Context, r FactRecord) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO kb_compat_fact (identifier, kind, status, severity, value_text, value_numeric, mapping_target, source)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 ON CONFLICT (identifier, kind) DO UPDATE SET
		   status = EXCLUDED.status, severity = EXCLUDED.severity,
		   value_text = EXCLUDED.value_text, value_numeric = EXCLUDED.value_numeric,
		   mapping_target = EXCLUDED.mapping_target, source = EXCLUDED.source,
		   verified_at = now()`,
		r.Identifier, r.Kind, r.Status, r.Severity,
		r.ValueText, r.ValueNumeric, r.MappingTarget, r.Source,
	)
	if err != nil {
		return err
	}
	pglisten.Notify(ctx, s.pool, notifyChannel, "fact:"+r.Identifier)
	return nil
}

// RecordFix inserts a new deterministic fix rule into PG and sends a NOTIFY.
func (s *Service) RecordFix(ctx context.Context, r FixRecord) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO kb_compat_fix (pattern, fix_type, resolution_target, source)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (pattern) DO UPDATE SET
		   fix_type = EXCLUDED.fix_type, resolution_target = EXCLUDED.resolution_target,
		   source = EXCLUDED.source`,
		r.Pattern, r.FixType, r.ResolutionTarget, r.Source,
	)
	if err != nil {
		return err
	}
	pglisten.Notify(ctx, s.pool, notifyChannel, "fix:"+r.Pattern)
	return nil
}

// loadFromDBImpl loads all facts and fixes from PG into the in-memory cache.
// Atomically swaps all maps to avoid partial state during refresh.
func (s *Service) loadFromDBImpl(ctx context.Context) error {
	newConstants := make(map[string]interp.Value)
	newFixes := make(map[string]string)
	newFunctions := make(map[string]funcInfo)

	// 1. Load facts.
	rows, err := s.pool.Query(ctx,
		`SELECT identifier, kind, status, severity, value_text, value_numeric, mapping_target
		 FROM kb_compat_fact`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var identifier, kind, status, severity string
		var valueText *string
		var valueNumeric *int32
		var mappingTarget *string
		if err := rows.Scan(&identifier, &kind, &status, &severity, &valueText, &valueNumeric, &mappingTarget); err != nil {
			rows.Close()
			return err
		}

		switch kind {
		case "constant":
			if mappingTarget != nil && *mappingTarget != "" {
				newFixes[identifier] = *mappingTarget
				continue
			}
			if valueNumeric != nil {
				newConstants[identifier] = interp.IntVal(*valueNumeric)
			} else if valueText != nil {
				switch *valueText {
				case "true":
					newConstants[identifier] = interp.BoolVal(true)
				case "false":
					newConstants[identifier] = interp.BoolVal(false)
				default:
					newConstants[identifier] = interp.StringVal(*valueText)
				}
			}
		case "function", "indicator":
			newFunctions[identifier] = funcInfo{
				supported: status == "supported",
				severity:  severity,
			}
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	// 2. Load fixes.
	fixRows, err := s.pool.Query(ctx,
		`SELECT pattern, resolution_target FROM kb_compat_fix`)
	if err != nil {
		return err
	}
	for fixRows.Next() {
		var pattern, target string
		if err := fixRows.Scan(&pattern, &target); err != nil {
			fixRows.Close()
			return err
		}
		newFixes[pattern] = target
	}
	fixRows.Close()
	if err := fixRows.Err(); err != nil {
		return err
	}

	// 3. Resolve aliases: for each fix, if the canonical name is in constants,
	//    add the alias → resolved value directly to constants for O(1) lookup.
	for alias, canonical := range newFixes {
		if v, ok := newConstants[canonical]; ok {
			newConstants[alias] = v
		}
	}

	// 4. Atomic swap.
	s.mu.Lock()
	s.constants = newConstants
	s.fixes = newFixes
	s.functions = newFunctions
	s.mu.Unlock()

	return nil
}

// ── K3: Demand Signal Capture ───────────────────────────────────────

// demandNotifyChannel is the PG LISTEN/NOTIFY channel for demand signal updates.
const demandNotifyChannel = "kb_demand_update"

// DemandSummaryRow is one row of the demand summary (per builtin).
type DemandSummaryRow struct {
	BuiltinName string
	TotalHits   int
	UniqueUsers int
	LastHitAt   time.Time
}

// RecordDemandSignal records that a user hit an unsupported builtin.
// Upserts hit_count++ and sends pg_notify for admin dashboard refresh.
// K3: captures demand signals so admins can prioritize which builtins to support.
func (s *Service) RecordDemandSignal(ctx context.Context, builtinName string, userID uuid.UUID) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO kb_demand_signal (builtin_name, user_id, hit_count, last_hit_at)
		 VALUES ($1, $2, 1, now())
		 ON CONFLICT (builtin_name, user_id) DO UPDATE SET
		   hit_count = kb_demand_signal.hit_count + 1,
		   last_hit_at = now()`,
		builtinName, userID,
	)
	if err != nil {
		return err
	}
	pglisten.Notify(ctx, s.pool, demandNotifyChannel, "demand:"+builtinName)
	return nil
}

// GetDemandSummary returns the aggregate demand signals ordered by total hits descending.
// Used by admin roadmap view to prioritize which builtins to implement next.
func (s *Service) GetDemandSummary(ctx context.Context) ([]DemandSummaryRow, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT builtin_name, SUM(hit_count), COUNT(*), MAX(last_hit_at)
		 FROM kb_demand_signal
		 GROUP BY builtin_name
		 ORDER BY SUM(hit_count) DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []DemandSummaryRow
	for rows.Next() {
		var r DemandSummaryRow
		if err := rows.Scan(&r.BuiltinName, &r.TotalHits, &r.UniqueUsers, &r.LastHitAt); err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, rows.Err()
}
