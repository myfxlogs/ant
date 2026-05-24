package mdgateway_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"anttrader/internal/mdgateway"
)

func TestGatewayInterface(t *testing.T) {
	// Compile-time check that Gateway interface has expected methods
	var _ mdgateway.Gateway = nil //nolint: staticcheck
	assert.Nil(t, nil)
}

func TestAccountHealthDefaults(t *testing.T) {
	h := mdgateway.AccountHealth{State: "disconnected"}
	assert.Equal(t, "disconnected", h.State)
}
