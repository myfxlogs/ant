package mql2go

import (
	"alphaforge/tools/mql2go/interp"
	"fmt"

	sitter "github.com/smacker/go-tree-sitter"
)

// This file contains declaration and enum compilation helpers
// extracted from compile_interp.go for file-size compliance.

func (c *compiler) compileDeclaration(n *sitter.Node) *interp.Statement {
	// Variable declaration as a statement: int x = 5; or int x = 5, y = 6;
	// Also handles uninitialized declarations: int local_value;
	// Rejects local array declarations: double arr[2]; (not supported).
	var decls []interp.Expr
	for i := 0; i < int(n.NamedChildCount()); i++ {
		child := n.NamedChild(i)
		if child.Type() == "array_declarator" {
			c.err = fmt.Errorf("local arrays are not supported: %s", c.text(n))
			return nil
		}
		if child.Type() == "init_declarator" {
			name := c.findIdent(child)
			if name == "" {
				continue
			}
			valExpr := c.findInitValue(child, name)
			if valExpr != nil {
				decls = append(decls, interp.Expr{
					Kind: interp.ExprDecl,
					Name: name,
					Args: []interp.Expr{*c.compileExpr(valExpr)},
				})
			} else {
				// Uninitialized declaration: int x; → ExprDecl with zero value.
				decls = append(decls, interp.Expr{
					Kind: interp.ExprDecl,
					Name: name,
					Args: []interp.Expr{{Kind: interp.ExprLiteral, Val: interp.IntVal(0)}},
				})
			}
		} else if child.Type() == "declarator" || child.Type() == nodeIdentifier {
			// Uninitialized declaration: int x; (identifier directly, no
			// init_declarator or declarator wrapper).
			name := c.findIdent(child)
			if name == "" {
				name = c.text(child)
			}
			if name == "" {
				continue
			}
			decls = append(decls, interp.Expr{
				Kind: interp.ExprDecl,
				Name: name,
				Args: []interp.Expr{{Kind: interp.ExprLiteral, Val: interp.IntVal(0)}},
			})
		}
	}
	if len(decls) == 0 {
		return nil
	}
	if len(decls) == 1 {
		return &interp.Statement{Kind: interp.StmtExpr, Expr: &decls[0]}
	}
	// Multi-variable declaration: ExprSeq with multiple ExprDecl.
	return &interp.Statement{
		Kind: interp.StmtExpr,
		Expr: &interp.Expr{Kind: interp.ExprSeq, Args: decls},
	}
}

// collectFuncParams extracts parameter declarations from a function_definition node.
func (c *compiler) collectFuncParams(n *sitter.Node) []interp.ParamDecl {
	var params []interp.ParamDecl
	for i := 0; i < int(n.NamedChildCount()); i++ {
		child := n.NamedChild(i)
		if child.Type() == "function_declarator" {
			for j := 0; j < int(child.NamedChildCount()); j++ {
				fc := child.NamedChild(j)
				if fc.Type() == "parameter_list" {
					for k := 0; k < int(fc.NamedChildCount()); k++ {
						pd := fc.NamedChild(k)
						if pd.Type() == "parameter_declaration" {
							pName := c.findIdent(pd)
							pType := c.findType(pd)
							if pName != "" {
								params = append(params, interp.ParamDecl{
									Name: pName,
									Type: pType,
								})
							}
						}
					}
				}
			}
		}
	}
	return params
}

// collectEnum processes enum declarations and registers constants.
func (c *compiler) collectEnum(ir *interp.IR, n *sitter.Node) {
	var enumName string
	for i := 0; i < int(n.NamedChildCount()); i++ {
		child := n.NamedChild(i)
		if child.Type() == "type_identifier" {
			enumName = c.text(child)
			if ir.EnumTypes == nil {
				ir.EnumTypes = make(map[string]bool)
			}
			ir.EnumTypes[enumName] = true
		}
		if child.Type() == "enumerator_list" {
			c.processEnumeratorList(ir, child, enumName)
		}
	}
}

func (c *compiler) processEnumeratorList(ir *interp.IR, list *sitter.Node, enumName string) {
	counter := int32(0)
	for j := 0; j < int(list.NamedChildCount()); j++ {
		ec := list.NamedChild(j)
		if ec.Type() != "enumerator" {
			continue
		}
		name, val := c.parseEnumerator(ec)
		if name == "" {
			continue
		}
		if val != nil {
			counter = *val
		}
		ir.Enums[name] = counter
		if enumName != "" {
			ir.Enums[enumName+"::"+name] = counter
		}
		counter++
	}
}

func (c *compiler) parseEnumerator(ec *sitter.Node) (name string, val *int32) {
	for k := 0; k < int(ec.NamedChildCount()); k++ {
		ecChild := ec.NamedChild(k)
		if ecChild.Type() == nodeIdentifier {
			name = c.text(ecChild)
		} else if ecChild.Type() == "number_literal" {
			nVal := interp.ParseNumberLiteral(c.text(ecChild))
			v := nVal.ToInt()
			val = &v
		}
	}
	return
}

// compileDoWhile compiles a do { } while(cond) statement.
// Also handles single-statement do-while: do stmt; while(cond);
func (c *compiler) compileDoWhile(n *sitter.Node) *interp.Statement {
	stmt := &interp.Statement{Kind: interp.StmtDoWhile}
	for i := 0; i < int(n.NamedChildCount()); i++ {
		child := n.NamedChild(i)
		if child.Type() == nodeCompoundStatement {
			stmt.Body = c.compileBlock(child)
		} else if child.Type() == nodeParenExpr {
			stmt.Cond = c.compileExpr(child)
		} else if stmt.Body == nil {
			// Single-statement body: do g_value++; while(cond);
			if s := c.compileStmt(child); s != nil {
				stmt.Body = []interp.Statement{*s}
			}
		}
	}
	return stmt
}
