package interp

// SerializeParams encodes parameter declarations to a compact binary format
// for database storage (BYTEA column in imported_strategies table).
//
// Format: u8 count + per-param: u8 nameLen + name + u8 typeLen + type + u8 hasDefault + defaultLen + default
func SerializeParams(params []ParamDecl) []byte {
	if len(params) == 0 {
		return nil
	}
	buf := make([]byte, 0, 64)
	buf = append(buf, byte(len(params)))
	for _, p := range params {
		buf = appendString(buf, p.Name)
		buf = appendString(buf, p.Type)
		if p.Default != nil {
			defVal := EvalExprLiteral(p.Default)
			buf = append(buf, 1)
			buf = appendString(buf, defVal)
		} else {
			buf = append(buf, 0)
		}
	}
	return buf
}

func appendString(buf []byte, s string) []byte {
	buf = append(buf, byte(len(s)))
	buf = append(buf, s...)
	return buf
}
