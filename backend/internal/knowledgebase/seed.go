package knowledgebase

import (
	"context"
	"fmt"

	"alphaforge/tools/mql2go/interp"
)

// Seed populates kb_compat_fact and kb_compat_fix from the existing Go maps:
//   - interp.MQLConstants → constants (kind="constant", status="supported")
//   - interp.AllImplementedFunctions() → functions (kind="function", status="supported")
//   - interp.AllUnsupportedFunctions() → functions (kind="function", status="unsupported")
//   - interp.CompatFixes → fixes (fix_type="alias")
//
// All entries are inserted with source="seed". Uses ON CONFLICT DO NOTHING
// so re-seeding is idempotent (won't overwrite manual/auto-verified entries).
func (s *Service) Seed(ctx context.Context) error {
	if err := s.seedConstants(ctx); err != nil {
		return fmt.Errorf("seed constants: %w", err)
	}
	if err := s.seedFunctions(ctx); err != nil {
		return fmt.Errorf("seed functions: %w", err)
	}
	return s.seedFixes(ctx)
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
			 ON CONFLICT (identifier, kind) DO NOTHING`,
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
			 ON CONFLICT (identifier, kind) DO NOTHING`,
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
			 ON CONFLICT (identifier, kind) DO NOTHING`,
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
			 ON CONFLICT (identifier, kind) DO NOTHING`,
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
			 ON CONFLICT (pattern) DO NOTHING`,
			pattern, target,
		); err != nil {
			return err
		}
	}
	return nil
}

// isIndicatorName returns true for iXxx pattern names (MQL indicator convention).
func isIndicatorName(name string) bool {
	return len(name) > 1 && name[0] == 'i' && name[1] >= 'A' && name[1] <= 'Z'
}
