package mql2go

import (
	"fmt"

	"alphaforge/tools/mql2go/interp"
)

// BuiltinFunc is a VM builtin function handler.
// It takes the VM and a slice of arguments, returns a result Value and error.
type BuiltinFunc func(vm *VM, args []interp.Value) (interp.Value, error)

// builtinEntry maps a builtin name to its handler function.
type builtinEntry struct {
	name string
	fn   BuiltinFunc
}

// builtinRegistry defines all available builtin functions.
// The order determines the BuiltinID (index in this slice).


// registerBuiltins populates the Bytecode's builtin map.
func (c *astCompiler) registerBuiltins() {
	for i, entry := range builtinRegistry {
		c.bc.Builtins[entry.name] = BuiltinID(i)
	}
}

// registerMethodBuiltin registers a method call (obj.method) as a builtin
// and returns its ID. The method name is prefixed with the object type.
// Uses the bytecode's own builtin map for dynamic names; the global registry
// is pre-populated with all known method names so no append is needed.
func (c *astCompiler) registerMethodBuiltin(methodName string, _ int) BuiltinID {
	if bid, ok := c.bc.Builtins[methodName]; ok {
		return bid
	}
	// Unknown method — reject compilation instead of dispatching to an
	// unrelated builtin ID and silently producing a wrong value.
	c.bc.Coverage.AddBlindSpot("unknown method: " + methodName)
	if c.err == nil {
		c.err = fmt.Errorf("unsupported method: %s", methodName)
	}
	return 0
}
