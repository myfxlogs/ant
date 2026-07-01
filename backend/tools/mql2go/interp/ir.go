package interp

// Pure Go IR types for the MQL interpreter.
// No tree-sitter dependency.

// Expr is a pure Go expression tree node.
type Expr struct {
	Kind     ExprKind
	Val      Value   // ExprLiteral: the literal value
	Name     string  // ExprVar / ExprConst: variable or constant name
	Op       string  // ExprBinary / ExprUnary: operator
	Args     []Expr  // ExprBinary: [left, right]; ExprCall: arguments; ExprField: [obj, args...]
	Index    *Expr   // ExprSubscript: index expression
	Cond     *Expr   // ExprTernary: condition
	ThenExpr *Expr   // ExprTernary: then branch
	ElseExpr *Expr   // ExprTernary: else branch
	IsAssign bool    // ExprField: true = field assignment (obj.field = value)
}

// ExprKind enumerates expression node types.
type ExprKind uint8

const (
	ExprLiteral       ExprKind = iota // Val
	ExprVar                         // Name
	ExprConst                       // Name (OP_BUY, PRICE_CLOSE, ...)
	ExprBinary                      // Op + Args[0], Args[1]
	ExprUnary                       // Op + Args[0]
	ExprCall                        // Name + Args
	ExprSubscript                   // Name + Index → Close[1], High[shift]
	ExprField                       // Args[0].Name.Args[1:] → obj.method or obj.field
	ExprTernary                     // Cond ? ThenExpr : ElseExpr
	ExprUpdate                      // Name + Op: i++, i--
	ExprAssignment                  // Name + Args[0]: a = b
	ExprDecl                        // Name + Args[0]: name := value (declaration)
	ExprCompoundAssign              // Name + Op + Args[0]: a += b
)

// Statement is a pure Go statement tree node.
type Statement struct {
	Kind     StatementKind
	Expr     *Expr        // StmtExpr / StmtReturn / StmtSwitch (switch expression)
	Cond     *Expr        // StmtIf / StmtWhile condition
	Init     *Statement   // StmtFor init
	Update   *Statement   // StmtFor update
	Body     []Statement  // child statements (StmtBlock, StmtFor, StmtIf body)
	ElseBody []Statement  // StmtIf else branch
	Cases    []SwitchCase // StmtSwitch case list (includes default)
}

// SwitchCase represents one case in a switch statement.
type SwitchCase struct {
	Expr *Expr       // case value (nil = default)
	Body []Statement // case body
}

// StatementKind enumerates statement node types.
type StatementKind uint8

const (
	StmtExpr    StatementKind = iota // expression statement
	StmtIf                           // if / else
	StmtFor                          // for(init; cond; update)
	StmtWhile                        // while(cond)
	StmtDoWhile                      // do { } while(cond)
	StmtReturn                       // return expr
	StmtBlock                        // { ... }
	StmtSwitch                       // switch / case / default
	StmtBreak                        // break
	StmtContinue                     // continue
)

// IR is the compiled representation of an MQL EA.
type IR struct {
	Version           string              // "mql4" or "mql5"
	OnInit            []Statement         // OnInit body (variable initialization + EventSetTimer)
	OnBar             []Statement         // OnBar body
	OnTick            []Statement         // OnTick body
	OnTimer           []Statement         // OnTimer body
	OnTrade           []Statement         // OnTrade body (MQL5, no args)
	OnTradeTransaction []Statement        // OnTradeTransaction body (MQL5, receives MqlTradeTransaction)
	OnDeinit          []Statement         // OnDeinit body
	Globals           []GlobalVar         // global variable declarations
	Params            []ParamDecl         // extern/input parameter declarations
	Funcs             map[string]*FuncDef // user-defined functions
	Enums             map[string]int32    // enum constants → int value
	EnumTypes         map[string]bool     // enum type names (e.g. "BuyOrSell0")
}

// GlobalVar represents a global variable declaration.
type GlobalVar struct {
	Name      string
	Type      string // "int", "double", "string", "bool", "datetime"
	InitVal   *Expr  // initializer expression (nil = zero value)
	IsArray   bool   // true if declared as array (e.g. double arr[30])
	ArraySize int    // array dimension size (0 = unknown/uninitialized)
}

// ParamDecl represents an extern/input parameter.
type ParamDecl struct {
	Name    string
	Type    string
	Default *Expr // default value
}

// FuncDef represents a user-defined function.
type FuncDef struct {
	Name   string      // function name
	Params []ParamDecl // parameter declarations
	Body   []Statement // function body statements
}
