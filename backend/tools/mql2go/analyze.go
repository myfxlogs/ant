package mql2go

import "strings"

// Analyze parses MQL source with tree-sitter and extracts a full StrategyIntent.
func Analyze(source string) (*StrategyIntent, error) {
	ast, err := ParseMQL(source)
	if err != nil {
		return analyzeFallback(source), nil
	}

	rec := NewRecognizer(ast)

	intent := &StrategyIntent{
		Meta: StrategyMeta{
			MQLVersion: detectMQLVersion(source),
		},
		Params:    rec.ExtractParams(ast),
		State:     rec.ExtractState(ast),
		Entry:     rec.ExtractEntries(ast),
		Exit:      rec.ExtractExits(ast),
		Execution: rec.DetectExecution(ast),
		Timer:     rec.DetectTimer(ast),
	}
	intent.Sizing = rec.DetectSizing(intent)
	intent.Indicators = extractIndicators(ast)
	return intent, nil
}

// Stub for tree-sitter parsing — will be implemented when grammar is compiled to Go.
func ParseMQL(source string) (*SourceFile, error) {
	return nil, nil
}

func analyzeFallback(source string) *StrategyIntent {
	return &StrategyIntent{
		Meta:      StrategyMeta{MQLVersion: detectMQLVersion(source)},
		Execution: ExecutionModel{Kind: ExecOnBar},
		Params:    extractParams(source),
		Entry:     detectEntries(source),
		Exit:      detectExits(source),
	}
}

func extractIndicators(ast *SourceFile) []IndicatorSpec {
	var specs []IndicatorSpec
	rec := &Recognizer{}
	rec.walkFunctions(ast, func(fn *FuncDef) {
		if fn.Body == nil {
			return
		}
		rec.walkNodes(fn.Body, func(n Node) bool {
			call, ok := n.(*CallExpr)
			if !ok {
				return true
			}
			method := indicatorMethod(call.Name)
			if method == "" {
				return true
			}
			resultVar := ""
			if fn.Body != nil {
				for _, stmt := range fn.Body.Statements {
					if vd, ok := stmt.(*VarDecl); ok && vd.Value == call {
						resultVar = vd.Name
						break
					}
				}
			}
			params := make(map[string]string)
			switch call.Name {
			case "iMA":
				if len(call.Args) > 2 {
					params["period"] = nodeToString(call.Args[2])
				}
				if len(call.Args) > 3 {
					params["shift"] = nodeToString(call.Args[3])
				}
			case "iRSI":
				if len(call.Args) > 2 {
					params["period"] = nodeToString(call.Args[2])
				}
				if len(call.Args) > 4 {
					params["shift"] = nodeToString(call.Args[4])
				}
			case "iATR":
				if len(call.Args) > 2 {
					params["period"] = nodeToString(call.Args[2])
				}
				if len(call.Args) > 3 {
					params["shift"] = nodeToString(call.Args[3])
				}
			}
			specs = append(specs, IndicatorSpec{
				SDKMethod: method,
				ResultVar: resultVar,
				Params:    params,
			})
			return true
		})
	})
	return specs
}

func indicatorMethod(mqlName string) string {
	switch mqlName {
	case "iMA":
		return "ema"
	case "iRSI":
		return "rsi"
	case "iATR":
		return "atr"
	case "iBands":
		return "bands"
	case "iMACD":
		return "macd"
	}
	return ""
}

func detectMQLVersion(source string) string {
	if strings.Contains(source, "class ") {
		return "mql5"
	}
	return "mql4"
}

func extractParams(source string) []ParamSpec { return nil }
func detectEntries(source string) []EntryRule { return nil }
func detectExits(source string) []ExitRule    { return nil }
