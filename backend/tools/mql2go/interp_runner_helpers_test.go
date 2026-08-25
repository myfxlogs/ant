package mql2go

// setCompilePythonWithCoverageFn sets the coverage compiler function for testing.
// Returns a restore function. Test-only — not for production use.
func setCompilePythonWithCoverageFn(fn func(string) (*VMRunner, *CoverageResult, error)) func() {
	compilePythonWithCoverageMu.Lock()
	orig := compilePythonWithCoverageFn
	compilePythonWithCoverageFn = fn
	compilePythonWithCoverageMu.Unlock()
	return func() {
		compilePythonWithCoverageMu.Lock()
		compilePythonWithCoverageFn = orig
		compilePythonWithCoverageMu.Unlock()
	}
}
