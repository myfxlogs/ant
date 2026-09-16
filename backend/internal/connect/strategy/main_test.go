package strategy

import (
	"testing"

	"go.uber.org/goleak"
)

// QS-2.2: package-level goroutine leak gate.
//
// Exempted (third-party resident goroutines, proven via stack + go.mod:87
// indirect dep — this repo has no direct rjeczalik/notify import):
//   - notify.(*nonrecursiveTree).dispatch / .internal: rjeczalik/notify@v0.9.3
//     spawns a process-lifetime event tree on first Watch; created transitively
//     by an indirect dependency during earlier tests, not by this package.
var goleakIgnoreThirdParty = []goleak.Option{
	goleak.IgnoreTopFunction("github.com/rjeczalik/notify.(*nonrecursiveTree).dispatch"),
	goleak.IgnoreTopFunction("github.com/rjeczalik/notify.(*nonrecursiveTree).internal"),
}

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m, goleakIgnoreThirdParty...)
}
