package strategy

import "alphaforge/tools/mql2go"

// compileForLive compiles source for live execution with bytecode cache.
// isPython selects the Python vs MQL compiler front-end.
// Returns (runner, bytecodeData, error). bytecodeData is non-nil on both
// cache hit (returns cachedBytecode input) and cold compile (fresh marshal).
//
// VM-AUDIT-2026-08-27-6: unified helper so all 4 live paths (executeVMLive,
// executePythonVMLive, NewVMLiveSessionCached, NewPythonVMLiveSessionCached)
// share the same cache verification logic. Prevents future live paths from
// bypassing SourceHash verification (the root cause of BUG-1).
func compileForLive(source string, cachedBytecode []byte, isPython bool) (*mql2go.VMRunner, []byte, error) {
	if isPython {
		return mql2go.CompilePythonCached(source, cachedBytecode)
	}
	return mql2go.CompileMQLCached(source, cachedBytecode)
}
