package interp

import "errors"

// callUserFunc invokes a user-defined MQL function.
// Creates a new scope, binds parameters, executes the body, and returns the return value.
func (it *Interpreter) callUserFunc(fn *FuncDef, args []Expr) Value {
	// Save current scope stack and create a fresh one for the function
	savedScopes := it.scopes
	it.scopes = []map[string]Value{make(map[string]Value)}

	// Bind parameters
	for i, p := range fn.Params {
		if i < len(args) {
			it.scopes[0][p.Name] = it.evalExpr(&args[i])
		} else if p.Default != nil {
			it.scopes[0][p.Name] = it.evalExpr(p.Default)
		} else {
			it.scopes[0][p.Name] = zeroValue(p.Type)
		}
	}

	// Execute function body
	it.retVal = NoneVal()
	err := it.execBlock(fn.Body)

	// Restore scope stack
	it.scopes = savedScopes

	if err != nil && !errors.Is(err, errReturn) {
		// Log error but don't propagate
		if it.ctx != nil {
			it.ctx.Log("MQL interpreter: error in user function " + fn.Name)
		}
	}
	return it.retVal
}
