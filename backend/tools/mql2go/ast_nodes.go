package mql2go

// AST node types for MQL — mirrors the Python ast_nodes.py.
// Produced by the tree-sitter CST→AST bridge.

// Node is the base interface for all AST nodes.
type Node interface{ node() }

// ── Top-level ─────────────────────────────────────────────────────

type SourceFile struct{ Declarations []Node }

// ── Declarations ──────────────────────────────────────────────────

type VarDecl struct {
	Name     string
	VarType  string
	Value    Node
	IsExtern bool
	IsInput  bool
}

type FuncDef struct {
	Name       string
	ReturnType string
	Params     []string
	Body       *CompoundStmt
}

type CompoundStmt struct{ Statements []Node }
type ExpressionStmt struct{ Expr Node }
type ReturnStmt struct{ Value Node }

// ── Control flow ──────────────────────────────────────────────────

type IfStmt struct {
	Condition  Node
	ThenBranch Node
	ElseBranch Node
}

type ForStmt struct {
	Init      Node
	Condition Node
	Update    Node
	Body      Node
}

type WhileStmt struct {
	Condition Node
	Body      Node
}

// ── Expressions ───────────────────────────────────────────────────

type BinaryOp struct {
	Left  Node
	Op    string
	Right Node
}

type UnaryOp struct {
	Op      string
	Operand Node
}

type CallExpr struct {
	Name string
	Args []Node
}

type Identifier struct{ Name string }
type NumberLiteral struct{ Value string }
type StringLiteral struct{ Value string }
type SubscriptExpr struct {
	Name  string
	Index Node
}
type AssignmentExpr struct {
	LHS string
	RHS Node
}

// Marker methods.
func (*SourceFile) node()      {}
func (*VarDecl) node()         {}
func (*FuncDef) node()         {}
func (*CompoundStmt) node()    {}
func (*ExpressionStmt) node()  {}
func (*ReturnStmt) node()      {}
func (*IfStmt) node()          {}
func (*ForStmt) node()         {}
func (*WhileStmt) node()       {}
func (*BinaryOp) node()        {}
func (*UnaryOp) node()         {}
func (*CallExpr) node()        {}
func (*Identifier) node()      {}
func (*NumberLiteral) node()   {}
func (*StringLiteral) node()   {}
func (*SubscriptExpr) node()   {}
func (*AssignmentExpr) node()  {}
