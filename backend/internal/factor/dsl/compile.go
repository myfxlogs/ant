package dsl

import (
	"fmt"
)

// Compiler compiles a DSL expression AST into an evaluable Op.
type Compiler struct {
	fields  map[string]int // field name → FieldIndex
	factors map[string]Op
}

// FieldIndex maps a field name to an index in the []float64 field array.
type FieldIndex struct {
	Fields map[string]int
}

// NewCompiler creates a Compiler with the given field mapping.
func NewCompiler(fields FieldIndex, factors map[string]Op) *Compiler {
	return &Compiler{fields: fields.Fields, factors: factors}
}

// Compile parses and compiles an expression into an evaluable Op.
func (c *Compiler) Compile(expr string) (Op, error) {
	node, err := Parse(expr)
	if err != nil {
		return nil, fmt.Errorf("compile: %w", err)
	}
	return c.compileNode(node)
}

// CompileWithFactors compiles an expression with runtime factor Ops.
func (c *Compiler) CompileWithFactors(expr string, factors map[string]Op) (Op, error) {
	saved := c.factors
	c.factors = factors
	defer func() { c.factors = saved }()

	node, err := Parse(expr)
	if err != nil {
		return nil, fmt.Errorf("compile: %w", err)
	}
	return c.compileNode(node)
}

func (c *Compiler) compileNode(node Node) (Op, error) {
	switch n := node.(type) {
	case *NumberLit:
		return &constOp{val: n.Value}, nil
	case *FieldRef:
		idx, ok := c.fields[n.Name]
		if !ok {
			return nil, fmt.Errorf("compile: unknown field $%s", n.Name)
		}
		return &fieldOp{idx: idx}, nil
	case *FactorRef:
		op, ok := c.factors[n.Name]
		if !ok {
			return nil, fmt.Errorf("compile: unknown factor %q", n.Name)
		}
		return op, nil
	case *BinaryExpr:
		return c.compileBinary(n)
	case *UnaryExpr:
		return c.compileUnary(n)
	case *TernaryExpr:
		return c.compileTernary(n)
	case *CallExpr:
		return c.compileCall(n)
	default:
		return nil, fmt.Errorf("compile: unsupported node type %T", n)
	}
}

func (c *Compiler) compileBinary(n *BinaryExpr) (Op, error) {
	left, err := c.compileNode(n.Left)
	if err != nil {
		return nil, err
	}
	right, err := c.compileNode(n.Right)
	if err != nil {
		return nil, err
	}
	return &binaryOp{op: n.Op, left: left, right: right}, nil
}

func (c *Compiler) compileUnary(n *UnaryExpr) (Op, error) {
	inner, err := c.compileNode(n.Expr)
	if err != nil {
		return nil, err
	}
	return &unaryOp{op: n.Op, inner: inner}, nil
}

func (c *Compiler) compileTernary(n *TernaryExpr) (Op, error) {
	cond, err := c.compileNode(n.Cond)
	if err != nil {
		return nil, err
	}
	t, err := c.compileNode(n.True)
	if err != nil {
		return nil, err
	}
	f, err := c.compileNode(n.False)
	if err != nil {
		return nil, err
	}
	return &ternaryOp{cond: cond, t: t, f: f}, nil
}
