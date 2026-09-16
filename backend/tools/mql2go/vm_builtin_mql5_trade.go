package mql2go

import (
	"alphaforge/tools/mql2go/interp"
)

// MQL5 trade helpers — PositionSelect only.
// VM-API-TRUTH-1: MQL5 order/deal/history stubs removed (reclassified
// StatusUnsupported in interp/api_registry.go; the compiler now rejects
// them instead of letting strategies run on fake data).

func builtinPositionSelect(vm *VM, args []interp.Value) (interp.Value, error) {
	// Delegate to PositionSelectByTicket
	return builtinPositionSelectByTicket(vm, args)
}
