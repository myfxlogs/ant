// service_loader.go — DB loading and demand signal methods extracted from service.go.
package knowledgebase

import (
	"context"
	"time"

	"github.com/google/uuid"

	"alphaforge/internal/pglisten"
	"alphaforge/tools/mql2go/interp"
)

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
