package mql2go

import (
	"fmt"

	"github.com/shopspring/decimal"

	"alphaforge/tools/mql2go/interp"
)

// Opcode is a single bytecode operation.
type Opcode uint8

const (
	// Stack operations
	OP_PUSH_CONST Opcode = iota
	OP_PUSH_VAR
	OP_PUSH_GLOBAL
	OP_STORE_VAR
	OP_STORE_GLOBAL
	OP_POP
	OP_DUP
	OP_SWAP

	// Arithmetic
	OP_ADD
	OP_SUB
	OP_MUL
	OP_DIV
	OP_MOD
	OP_FLOOR_DIV
	OP_NEG

	// Comparison
	OP_EQ
	OP_NE
	OP_LT
	OP_LE
	OP_GT
	OP_GE

	// Logical
	OP_AND
	OP_OR
	OP_NOT

	// Control flow
	OP_JMP
	OP_JMP_IF_FALSE
	OP_JMP_IF_TRUE

	// Function call
	OP_CALL_BUILTIN
	OP_CALL_USER
	OP_ENTER_FUNC
	OP_LEAVE_FUNC

	// Event entry points (markers — JMP targets)
	OP_ENTER_ONINIT
	OP_ENTER_ONBAR
	OP_ENTER_ONTICK
	OP_ENTER_ONTRADE
	OP_ENTER_ONTIMER
	OP_ENTER_ONDEINIT
	OP_ENTER_ONTRADETRANSACTION
	OP_ENTER_ONBOOKEVENT

	// Return
	OP_RETURN

	// Subscript (series access: Close[i], Open[i], etc.)
	OP_PUSH_SERIES

	// User array access (arr[i] read / arr[i] = val write)
	OP_PUSH_ARRAY
	OP_STORE_ARRAY

	// Field access (obj.field)
	OP_GET_FIELD
	OP_SET_FIELD

	// Halt
	OP_HALT
)

// ConstID is an index into the constant pool.
type ConstID uint16

// VarID is an index into the variable slot space.
type VarID uint16

// FuncID is an index into the function table.
type FuncID uint16

// BuiltinID is an index into the builtin function table.
type BuiltinID uint16

// Instruction is a single bytecode instruction.
type Instruction struct {
	Op   Opcode
	A    int32  // generic operand (const ID, var ID, func ID, jump target, etc.)
	B    int32  // second operand (for binary ops, arg count, etc.)
	Line uint32 // source line for debugging
}

// ConstValue is a constant pool entry.
type ConstValue struct {
	Kind interp.ValueKind
	Int  int32
	Dec  decimal.Decimal
	Str  string
	Bool bool
}

// Bytecode is a compiled MQL strategy ready for VM execution.
type Bytecode struct {
	// Constant pool
	Consts []ConstValue

	// Instructions (flat array, jump targets are indices)
	Code []Instruction

	// Variable name → slot ID mapping (globals)
	GlobalSlots map[string]VarID

	// Global variable declarations (for array initialization)
	GlobalDecls []interp.GlobalVar

	// Function name → entry point mapping (user-defined funcs)
	Funcs map[string]FuncEntry

	// Builtin name → ID mapping
	Builtins map[string]BuiltinID

	// Event entry points (instruction indices, -1 = not compiled)
	OnInit             int32
	OnBar              int32
	OnTick             int32
	OnTrade            int32
	OnTimer            int32
	OnDeinit           int32
	OnTradeTransaction int32
	OnBookEvent        int32

	// EventLocals tracks the number of local variable slots needed per event handler.
	// Key = entry PC, value = number of local slots.
	EventLocals map[int32]int

	// Parameters (extern/input declarations)
	Params []interp.ParamDecl

	// Version ("mql4" or "mql5")
	Version string

	// SourceHash is the SHA256 hex of the source code, for cache integrity.
	// VM-CACHE-INTEGRITY-1: CompileMQLCached verifies this on cache hit to
	// reject stale bytecode from a different source.
	SourceHash string

	// Enums (constant name → int value)
	Enums map[string]int32

	// Coverage report (populated during compilation)
	Coverage *CoverageReport
}

// FuncEntry describes a user-defined function in the bytecode.
type FuncEntry struct {
	Name      string
	EntryPC   int32    // instruction index of function body
	NumParams int      // number of parameters
	NumLocals int      // total local slots (params + locals)
	ParamName []string // parameter names (for binding)
}

// CoverageReport tracks what MQL features were encountered during compilation.
type CoverageReport struct {
	SupportedNodes   []string // CST node types successfully compiled
	UnsupportedNodes []string // CST node types that couldn't be compiled
	BlindSpots       []string // functions called that have no implementation
}

func (r *CoverageReport) AddSupported(nodeType string) {
	r.SupportedNodes = append(r.SupportedNodes, nodeType)
}

func (r *CoverageReport) AddUnsupported(nodeType string) {
	r.UnsupportedNodes = append(r.UnsupportedNodes, nodeType)
}

func (r *CoverageReport) AddBlindSpot(name string) {
	r.BlindSpots = append(r.BlindSpots, name)
}

// constFromValue converts an interp.Value to a ConstValue.
func constFromValue(v interp.Value) ConstValue {
	return ConstValue{
		Kind: v.Kind,
		Int:  v.Int,
		Dec:  v.Decimal,
		Str:  v.Str,
		Bool: v.Bool,
	}
}

// constToValue converts a ConstValue back to an interp.Value.
func constToValue(c ConstValue) interp.Value {
	return interp.Value{
		Kind:    c.Kind,
		Int:     c.Int,
		Decimal: c.Dec,
		Str:     c.Str,
		Bool:    c.Bool,
	}
}

// String returns a human-readable representation of an instruction.
func (ins Instruction) String() string {
	return fmt.Sprintf("%s %d %d (line %d)", opName(ins.Op), ins.A, ins.B, ins.Line)
}

func opName(op Opcode) string {
	names := []string{
		"PUSH_CONST", "PUSH_VAR", "PUSH_GLOBAL", "STORE_VAR", "STORE_GLOBAL",
		"POP", "DUP", "SWAP",
		"ADD", "SUB", "MUL", "DIV", "MOD", "FLOOR_DIV", "NEG",
		"EQ", "NE", "LT", "LE", "GT", "GE",
		"AND", "OR", "NOT",
		"JMP", "JMP_IF_FALSE", "JMP_IF_TRUE",
		"CALL_BUILTIN", "CALL_USER", "ENTER_FUNC", "LEAVE_FUNC",
		"ENTER_ONINIT", "ENTER_ONBAR", "ENTER_ONTICK", "ENTER_ONTRADE", "ENTER_ONTIMER", "ENTER_ONDEINIT",
		"ENTER_ONTRADETRANSACTION", "ENTER_ONBOOKEVENT",
		"RETURN",
		"PUSH_SERIES",
		"PUSH_ARRAY", "STORE_ARRAY",
		"GET_FIELD", "SET_FIELD",
		"HALT",
	}
	if int(op) < len(names) {
		return names[op]
	}
	return fmt.Sprintf("OP_%d", op)
}
