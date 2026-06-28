package mql2go

import (
	sitter "github.com/smacker/go-tree-sitter"
)

// ── Indicators ──────────────────────────────────────────────────────

func extractIndicatorsCST(root *sitter.Node, version string) []IndicatorSpec {
	var specs []IndicatorSpec
	seen := make(map[string]bool)
	walkCST(root, func(n *sitter.Node) bool {
		if n.Type() != "call_expression" {
			return true
		}
		name := callFuncName(n)
		method := indicatorMethodCST(name, version)
		if method == "" {
			return true
		}
		key := name + ":" + nodeText("", n)
		if seen[key] {
			return true
		}
		seen[key] = true

		params := make(map[string]string)
		spec := IndicatorSpec{SDKMethod: method, Params: params}
		args := childByType("", n, "argument_list")
		if args != nil {
			named := getNamedChildren(args)
			extractIndicatorParams(name, version, named, params)
		}
		spec.ResultVar = findResultVarForNode(root, n)
		specs = append(specs, spec)
		return true
	})
	return specs
}

func findResultVarForNode(root *sitter.Node, callNode *sitter.Node) string {
	callStart := callNode.StartByte()
	for _, fn := range findFunctions(root) {
		body := funcBody(fn)
		if body == nil {
			continue
		}
		for i := 0; i < int(body.ChildCount()); i++ {
			c := body.Child(i)
			if c.Type() == "declaration" {
				init := childByType("", c, "init_declarator")
				if init == nil {
					continue
				}
				call := childByType("", init, "call_expression")
				if call == nil {
					continue
				}
				if call.StartByte() == callStart {
					id := childByType("", init, "identifier")
					if id != nil {
						return nodeText("", id)
					}
				}
			}
			if c.Type() == "expression_statement" {
				assign := childByType("", c, "assignment_expression")
				if assign == nil {
					continue
				}
				call := childByType("", assign, "call_expression")
				if call == nil {
					continue
				}
				if call.StartByte() == callStart {
					named := getNamedChildren(assign)
					if len(named) > 0 {
						lhs := named[0]
						if lhs.Type() == "identifier" || lhs.Type() == "field_identifier" {
							return nodeText("", lhs)
						}
						id := childByType("", lhs, "identifier")
						if id == nil {
							id = childByType("", lhs, "field_identifier")
						}
						if id != nil {
							return nodeText("", id)
						}
					}
				}
			}
		}
	}
	return ""
}

func findResultVar(root *sitter.Node, indicatorName string) string {
	for _, fn := range findFunctions(root) {
		body := funcBody(fn)
		if body == nil {
			continue
		}
		for i := 0; i < int(body.ChildCount()); i++ {
			c := body.Child(i)
			if c.Type() != "declaration" {
				continue
			}
			init := childByType("", c, "init_declarator")
			if init == nil {
				continue
			}
			call := childByType("", init, "call_expression")
			if call == nil {
				continue
			}
			if callFuncName(call) == indicatorName {
				id := childByType("", init, "identifier")
				if id != nil {
					return nodeText("", id)
				}
			}
		}
	}
	return ""
}
