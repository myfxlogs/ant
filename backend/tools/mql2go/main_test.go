package mql2go

import (
	"testing"

	"go.uber.org/goleak"
)

// QS-2.2: package-level goroutine leak gate.
func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}
