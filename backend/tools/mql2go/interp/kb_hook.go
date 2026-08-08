package interp

// KB hooks — set by the KB Service (internal/knowledgebase) at server startup.
// When nil (tests, CLI tools), LookupMQLConstant/LookupCompatFix/LookupAPI
// fall back to built-in maps (MQLConstants, CompatFixes, registryMap).
// This guarantees zero regression during migration: KB is additive, not replacement.

var (
	// kbConstantLookup returns a resolved Value for a constant identifier.
	// The KB Service implements this by checking its in-memory cache
	// (which includes both direct constants and alias resolutions).
	kbConstantLookup func(name string) (Value, bool)

	// kbFixLookup returns the canonical name for an alias identifier.
	// The KB Service implements this by checking its in-memory fix cache.
	kbFixLookup func(name string) (string, bool)

	// kbFunctionLookup returns function/indicator status info from KB.
	// Returns (supported, severity) where severity is "fatal"/"warning"/"info".
	kbFunctionLookup func(name string) (bool, string)
)

// SetKBConstantLookup sets the KB constant lookup hook.
// Called by KB Service at startup. Pass nil to disable (revert to built-in).
func SetKBConstantLookup(f func(name string) (Value, bool)) {
	kbConstantLookup = f
}

// SetKBFixLookup sets the KB compat-fix lookup hook.
func SetKBFixLookup(f func(name string) (string, bool)) {
	kbFixLookup = f
}

// SetKBFunctionLookup sets the KB function status lookup hook.
func SetKBFunctionLookup(f func(name string) (bool, string)) {
	kbFunctionLookup = f
}

// KBConstantLookupActive returns true if the KB constant hook is set.
// Used by tests to verify KB wiring.
func KBConstantLookupActive() bool {
	return kbConstantLookup != nil
}
