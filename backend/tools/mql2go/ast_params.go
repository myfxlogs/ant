package mql2go

import (
	"alphaforge/tools/mql2go/interp"
)

// ExtractParams returns the parameter declarations from a compiled Bytecode.
// These are the extern/input parameters from the MQL source.
func ExtractParams(bc *Bytecode) []interp.ParamDecl {
	return bc.Params
}

// ParamInfo is a simplified parameter representation for API responses.
type ParamInfo struct {
	Name    string
	Type    string
	Default string
}

// ExtractParamInfos returns parameter info as a simple slice.
// Filters out unreferenced extern string parameters — these are UI labels
// in the MT4 properties panel (e.g. multi-line descriptions), not trading parameters.
// A parameter is considered referenced if its global slot is read via OP_PUSH_GLOBAL
// anywhere in the bytecode.
func ExtractParamInfos(bc *Bytecode) []ParamInfo {
	referencedSlots := scanReferencedGlobalSlots(bc)

	out := make([]ParamInfo, 0, len(bc.Params))
	for _, p := range bc.Params {
		// Filter unreferenced extern string params (UI labels, not real parameters)
		if p.Type == "string" {
			if slot, ok := bc.GlobalSlots[p.Name]; ok {
				if !referencedSlots[int(slot)] {
					continue
				}
			}
		}
		pi := ParamInfo{Name: p.Name, Type: p.Type}
		if p.Default != nil {
			pi.Default = interp.EvalExprLiteral(p.Default)
		}
		out = append(out, pi)
	}
	return out
}

// scanReferencedGlobalSlots returns a set of global slot IDs that are read
// via OP_PUSH_GLOBAL anywhere in the bytecode instructions.
func scanReferencedGlobalSlots(bc *Bytecode) map[int]bool {
	refs := make(map[int]bool)
	for _, ins := range bc.Code {
		if ins.Op == OP_PUSH_GLOBAL {
			refs[int(ins.A)] = true
		}
	}
	return refs
}

// SerializeParams delegates to interp.SerializeParams for DB storage.
func SerializeParams(bc *Bytecode) []byte {
	return interp.SerializeParams(bc.Params)
}
