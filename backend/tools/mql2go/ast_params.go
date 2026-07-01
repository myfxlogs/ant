package mql2go

import (
	"anttrader/tools/mql2go/interp"
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
func ExtractParamInfos(bc *Bytecode) []ParamInfo {
	out := make([]ParamInfo, 0, len(bc.Params))
	for _, p := range bc.Params {
		pi := ParamInfo{Name: p.Name, Type: p.Type}
		if p.Default != nil {
			pi.Default = interp.EvalExprLiteral(p.Default)
		}
		out = append(out, pi)
	}
	return out
}

// SerializeParams delegates to interp.SerializeParams for DB storage.
func SerializeParams(bc *Bytecode) []byte {
	return interp.SerializeParams(bc.Params)
}
