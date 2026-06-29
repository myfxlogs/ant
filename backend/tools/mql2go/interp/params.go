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

// DeserializeParams decodes the binary format back to ParamDecl slice.
func DeserializeParams(raw []byte) []ParamDecl {
	if len(raw) == 0 {
		return nil
	}
	pos := 0
	count := int(raw[pos])
	pos++
	params := make([]ParamDecl, 0, count)
	for i := 0; i < count; i++ {
		if pos >= len(raw) {
			break
		}
		name, n := readString(raw[pos:])
		pos += n
		typ, n := readString(raw[pos:])
		pos += n
		var def *Expr
		if pos < len(raw) && raw[pos] == 1 {
			pos++
			defVal, n := readString(raw[pos:])
			pos += n
			def = &Expr{Kind: ExprLiteral, Val: StringVal(defVal)}
		} else {
			pos++
		}
		params = append(params, ParamDecl{Name: name, Type: typ, Default: def})
	}
	return params
}

// ParamDefaultsToMap extracts parameter name → default value as string.
// Parameters without defaults are omitted.
func ParamDefaultsToMap(raw []byte) map[string]string {
	params := DeserializeParams(raw)
	if len(params) == 0 {
		return nil
	}
	m := make(map[string]string, len(params))
	for _, p := range params {
		if p.Default != nil {
			m[p.Name] = EvalExprLiteral(p.Default)
		}
	}
	return m
}

func appendString(buf []byte, s string) []byte {
	buf = append(buf, byte(len(s)))
	buf = append(buf, s...)
	return buf
}

func readString(buf []byte) (string, int) {
	if len(buf) == 0 {
		return "", 0
	}
	n := int(buf[0])
	if 1+n > len(buf) {
		return "", 1
	}
	return string(buf[1 : 1+n]), 1 + n
}
