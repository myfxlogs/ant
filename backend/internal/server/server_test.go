package server

import (
	"testing"
)

func TestPackageCompiles(t *testing.T) {
	t.Parallel()
	// Verify the server package compiles and Run function signature is correct.
	_ = Run
}
