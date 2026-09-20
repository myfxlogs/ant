package knowledgebase

import (
	"context"
	"fmt"

	"alphaforge/tools/mql2go/interp"
)

// Seed reconciles kb_compat_fact and kb_compat_fix with the Go registries:
//   - interp.MQLConstants → constants (kind="constant", status="supported")
//   - interp.AllImplementedFunctions() → functions (kind="function", status="supported")
//   - interp.AllUnsupportedFunctions() → functions (kind="function", status="unsupported")
//   - interp.CompatFixes → fixes (fix_type="alias")
//
// Seed-sourced rows are a mirror of the built-in registries: the upsert
// corrects drifted values and pruneSeedRows removes rows whose identifier was
// deleted from the registries (stale seeds shadow the built-in table via the
// KB-first lookup — e.g. enum renumbering must propagate). Rows with
// source='manual'/'auto-verified' are deliberate ops overrides and are never
// touched by the upsert (WHERE source='seed') nor by pruning.
func (s *Service) Seed(ctx context.Context) error {
	if err := s.seedConstants(ctx); err != nil {
		return fmt.Errorf("seed constants: %w", err)
	}
	if err := s.seedFunctions(ctx); err != nil {
		return fmt.Errorf("seed functions: %w", err)
	}
	if err := s.seedFixes(ctx); err != nil {
		return fmt.Errorf("seed fixes: %w", err)
	}
	return s.pruneSeedRows(ctx)
}

func (s *Service) seedConstants(ctx context.Context) error {
	for name, v := range interp.MQLConstants {
		var valText *string
		var valNum *int32

		switch v.Kind {
		case interp.ValInt:
			n := v.Int
			valNum = &n
		case interp.ValBool:
			if v.Bool {
				t := "true"
				valText = &t
			} else {
				f := "false"
				valText = &f
			}
		default:
			continue
		}

		if _, err := s.pool.Exec(ctx,
			`INSERT INTO kb_compat_fact (identifier, kind, status, severity, value_text, value_numeric, source)
			 VALUES ($1, 'constant', 'supported', 'info', $2, $3, 'seed')
			 ON CONFLICT (identifier, kind) DO UPDATE
			   SET value_text = EXCLUDED.value_text, value_numeric = EXCLUDED.value_numeric
			   WHERE kb_compat_fact.source = 'seed'`,
			name, valText, valNum,
		); err != nil {
			return err
		}
	}

	// Seed alias entries from CompatFixes (mapping_target set, no direct value).
	for alias, canonical := range interp.CompatFixes {
		target := canonical
		if _, err := s.pool.Exec(ctx,
			`INSERT INTO kb_compat_fact (identifier, kind, status, severity, mapping_target, source)
			 VALUES ($1, 'constant', 'supported', 'info', $2, 'seed')
			 ON CONFLICT (identifier, kind) DO UPDATE
			   SET mapping_target = EXCLUDED.mapping_target
			   WHERE kb_compat_fact.source = 'seed'`,
			alias, &target,
		); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) seedFunctions(ctx context.Context) error {
	// Implemented functions.
	for _, name := range interp.AllImplementedFunctions() {
		kind := "function"
		if isIndicatorName(name) {
			kind = "indicator"
		}
		if _, err := s.pool.Exec(ctx,
			`INSERT INTO kb_compat_fact (identifier, kind, status, severity, source)
			 VALUES ($1, $2, 'supported', 'info', 'seed')
			 ON CONFLICT (identifier, kind) DO UPDATE
			   SET status = 'supported', severity = 'info'
			   WHERE kb_compat_fact.source = 'seed'`,
			name, kind,
		); err != nil {
			return err
		}
	}

	// Unsupported functions.
	for _, name := range interp.AllUnsupportedFunctions() {
		severity := interp.SeverityForBuiltin(name)
		kind := "function"
		if isIndicatorName(name) {
			kind = "indicator"
		}
		if _, err := s.pool.Exec(ctx,
			`INSERT INTO kb_compat_fact (identifier, kind, status, severity, source)
			 VALUES ($1, $2, 'unsupported', $3, 'seed')
			 ON CONFLICT (identifier, kind) DO UPDATE
			   SET status = 'unsupported', severity = EXCLUDED.severity
			   WHERE kb_compat_fact.source = 'seed'`,
			name, kind, severity,
		); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) seedFixes(ctx context.Context) error {
	for pattern, target := range interp.CompatFixes {
		if _, err := s.pool.Exec(ctx,
			`INSERT INTO kb_compat_fix (pattern, fix_type, resolution_target, source)
			 VALUES ($1, 'alias', $2, 'seed')
			 ON CONFLICT (pattern) DO UPDATE
			   SET resolution_target = EXCLUDED.resolution_target
			   WHERE kb_compat_fix.source = 'seed'`,
			pattern, target,
		); err != nil {
			return err
		}
	}
	return nil
}

// pruneSeedRows deletes seed-sourced rows whose identifier no longer exists in
// the built-in registries. Without pruning, a removed constant/function stays
// resolvable forever (seed rows are never refreshed by ON CONFLICT paths) and
// shadows the registries via the KB-first lookups — resurrecting deliberately
// deleted names and leaking stale enum values into compiled bytecode.
func (s *Service) pruneSeedRows(ctx context.Context) error {
	validConstants := make([]string, 0, len(interp.MQLConstants)+len(interp.CompatFixes))
	for name := range interp.MQLConstants {
		validConstants = append(validConstants, name)
	}
	for alias := range interp.CompatFixes {
		validConstants = append(validConstants, alias)
	}
	if _, err := s.pool.Exec(ctx,
		`DELETE FROM kb_compat_fact
		  WHERE source = 'seed' AND kind = 'constant' AND identifier <> ALL($1)`,
		validConstants,
	); err != nil {
		return err
	}

	validFuncs := interp.AllImplementedFunctions()
	validFuncs = append(validFuncs, interp.AllUnsupportedFunctions()...)
	if _, err := s.pool.Exec(ctx,
		`DELETE FROM kb_compat_fact
		  WHERE source = 'seed' AND kind IN ('function', 'indicator') AND identifier <> ALL($1)`,
		validFuncs,
	); err != nil {
		return err
	}

	validFixes := make([]string, 0, len(interp.CompatFixes))
	for pattern := range interp.CompatFixes {
		validFixes = append(validFixes, pattern)
	}
	_, err := s.pool.Exec(ctx,
		`DELETE FROM kb_compat_fix
		  WHERE source = 'seed' AND pattern <> ALL($1)`,
		validFixes,
	)
	return err
}

// isIndicatorName returns true for iXxx pattern names (MQL indicator convention).
func isIndicatorName(name string) bool {
	return len(name) > 1 && name[0] == 'i' && name[1] >= 'A' && name[1] <= 'Z'
}
