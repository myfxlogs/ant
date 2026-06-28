package interp

import (
	"encoding/binary"
	"io"

	"github.com/shopspring/decimal"
)

// ── IR Binary Serialization ─────────────────────────────────────────
//
// Hand-rolled binary format for transferring IR from host to WASM module.
// Not proto (IR is not a proto message) and not JSON (project rule).
// Format is length-prefixed fields, little-endian uint32 for lengths.
//
// Layout:
//   u8  version (1)
//   u32 len(version_str) + version_str
//   u32 num_globals + [GlobalVar]...
//   u32 num_params + [ParamDecl]...
//   u32 num_funcs + [FuncDef]...
//   u32 num_enums + [enum entries]...
//   u32 len(onInit) + [Statement]...
//   u32 len(onBar) + [Statement]...
//   u32 len(onTick) + [Statement]...
//   u32 len(onTimer) + [Statement]...
//   u32 len(onDeinit) + [Statement]...

const irFormatVersion byte = 1

// SerializeIR encodes an IR to a binary byte slice for WASM transfer.
func SerializeIR(ir *IR) []byte {
	buf := &writeBuf{}
	buf.writeU8(irFormatVersion)
	buf.writeString(ir.Version)
	buf.writeGlobals(ir.Globals)
	buf.writeParams(ir.Params)
	buf.writeFuncs(ir.Funcs)
	buf.writeEnums(ir.Enums)
	buf.writeStatements(ir.OnInit)
	buf.writeStatements(ir.OnBar)
	buf.writeStatements(ir.OnTick)
	buf.writeStatements(ir.OnTimer)
	buf.writeStatements(ir.OnDeinit)
	return buf.bytes()
}

// DeserializeIR reconstructs an IR from a binary byte slice.
func DeserializeIR(data []byte) *IR {
	if len(data) == 0 {
		return nil
	}
	buf := &readBuf{data: data}
	ver := buf.readU8()
	if ver != irFormatVersion {
		return nil
	}
	ir := &IR{
		Version:  buf.readString(),
		Globals:  buf.readGlobals(),
		Params:   buf.readParams(),
		Funcs:    buf.readFuncs(),
		Enums:    buf.readEnums(),
		OnInit:   buf.readStatements(),
		OnBar:    buf.readStatements(),
		OnTick:   buf.readStatements(),
		OnTimer:  buf.readStatements(),
		OnDeinit: buf.readStatements(),
	}
	if buf.err != nil {
		return nil
	}
	return ir
}

// ── Write buffer ────────────────────────────────────────────────────

type writeBuf struct {
	data []byte
}

func (w *writeBuf) writeU8(v byte) {
	w.data = append(w.data, v)
}

func (w *writeBuf) writeU32(v uint32) {
	var b [4]byte
	binary.LittleEndian.PutUint32(b[:], v)
	w.data = append(w.data, b[:]...)
}

func (w *writeBuf) writeString(s string) {
	w.writeU32(uint32(len(s)))
	w.data = append(w.data, []byte(s)...)
}

func (w *writeBuf) writeBool(b bool) {
	if b {
		w.writeU8(1)
	} else {
		w.writeU8(0)
	}
}

func (w *writeBuf) bytes() []byte { return w.data }

func (w *writeBuf) writeGlobals(globals []GlobalVar) {
	w.writeU32(uint32(len(globals)))
	for _, g := range globals {
		w.writeString(g.Name)
		w.writeString(g.Type)
		w.writeBool(g.InitVal != nil)
		if g.InitVal != nil {
			w.writeExpr(g.InitVal)
		}
	}
}

func (w *writeBuf) writeParams(params []ParamDecl) {
	w.writeU32(uint32(len(params)))
	for _, p := range params {
		w.writeString(p.Name)
		w.writeString(p.Type)
		w.writeBool(p.Default != nil)
		if p.Default != nil {
			w.writeExpr(p.Default)
		}
	}
}

func (w *writeBuf) writeFuncs(funcs map[string]*FuncDef) {
	w.writeU32(uint32(len(funcs)))
	for name, fn := range funcs {
		w.writeString(name)
		w.writeParams(fn.Params)
		w.writeStatements(fn.Body)
	}
}

func (w *writeBuf) writeEnums(enums map[string]int32) {
	w.writeU32(uint32(len(enums)))
	for name, val := range enums {
		w.writeString(name)
		w.writeU32(uint32(val))
	}
}

func (w *writeBuf) writeStatements(stmts []Statement) {
	w.writeU32(uint32(len(stmts)))
	for i := range stmts {
		w.writeStatement(&stmts[i])
	}
}

func (w *writeBuf) writeStatement(s *Statement) {
	w.writeU8(byte(s.Kind))
	// Expr (nullable)
	w.writeBool(s.Expr != nil)
	if s.Expr != nil {
		w.writeExpr(s.Expr)
	}
	// Cond (nullable)
	w.writeBool(s.Cond != nil)
	if s.Cond != nil {
		w.writeExpr(s.Cond)
	}
	// Init (nullable)
	w.writeBool(s.Init != nil)
	if s.Init != nil {
		w.writeStatement(s.Init)
	}
	// Update (nullable)
	w.writeBool(s.Update != nil)
	if s.Update != nil {
		w.writeStatement(s.Update)
	}
	// Body
	w.writeStatements(s.Body)
	// ElseBody
	w.writeStatements(s.ElseBody)
	// Cases
	w.writeU32(uint32(len(s.Cases)))
	for _, c := range s.Cases {
		w.writeBool(c.Expr != nil)
		if c.Expr != nil {
			w.writeExpr(c.Expr)
		}
		w.writeStatements(c.Body)
	}
}

func (w *writeBuf) writeExpr(e *Expr) {
	w.writeU8(byte(e.Kind))
	w.writeValue(e.Val)
	w.writeString(e.Name)
	w.writeString(e.Op)
	w.writeBool(e.IsAssign)
	// Args
	w.writeU32(uint32(len(e.Args)))
	for i := range e.Args {
		w.writeExpr(&e.Args[i])
	}
	// Index (nullable)
	w.writeBool(e.Index != nil)
	if e.Index != nil {
		w.writeExpr(e.Index)
	}
	// Cond (nullable)
	w.writeBool(e.Cond != nil)
	if e.Cond != nil {
		w.writeExpr(e.Cond)
	}
	// ThenExpr (nullable)
	w.writeBool(e.ThenExpr != nil)
	if e.ThenExpr != nil {
		w.writeExpr(e.ThenExpr)
	}
	// ElseExpr (nullable)
	w.writeBool(e.ElseExpr != nil)
	if e.ElseExpr != nil {
		w.writeExpr(e.ElseExpr)
	}
}

func (w *writeBuf) writeValue(v Value) {
	w.writeU8(byte(v.Kind))
	w.writeU32(uint32(v.Int))
	// Decimal as string
	w.writeString(v.Decimal.String())
	w.writeString(v.Str)
	w.writeBool(v.Bool)
	w.writeU64(uint64(v.Datetime))
	// Array
	w.writeU32(uint32(len(v.Array)))
	for i := range v.Array {
		w.writeValue(v.Array[i])
	}
	// Class (nullable)
	w.writeBool(v.Class != nil)
	if v.Class != nil {
		w.writeString(v.Class.Name)
		w.writeU32(uint32(len(v.Class.Fields)))
		for name, val := range v.Class.Fields {
			w.writeString(name)
			w.writeValue(val)
		}
	}
}

func (w *writeBuf) writeU64(v uint64) {
	var b [8]byte
	binary.LittleEndian.PutUint64(b[:], v)
	w.data = append(w.data, b[:]...)
}

// ── Read buffer ─────────────────────────────────────────────────────

type readBuf struct {
	data []byte
	pos  int
	err  error
}

func (r *readBuf) readU8() byte {
	if r.err != nil || r.pos+1 > len(r.data) {
		r.err = io.ErrUnexpectedEOF
		return 0
	}
	v := r.data[r.pos]
	r.pos++
	return v
}

func (r *readBuf) readU32() uint32 {
	if r.err != nil || r.pos+4 > len(r.data) {
		r.err = io.ErrUnexpectedEOF
		return 0
	}
	v := binary.LittleEndian.Uint32(r.data[r.pos:])
	r.pos += 4
	return v
}

func (r *readBuf) readU64() uint64 {
	if r.err != nil || r.pos+8 > len(r.data) {
		r.err = io.ErrUnexpectedEOF
		return 0
	}
	v := binary.LittleEndian.Uint64(r.data[r.pos:])
	r.pos += 8
	return v
}

func (r *readBuf) readBool() bool {
	return r.readU8() == 1
}

func (r *readBuf) readString() string {
	n := r.readU32()
	if r.err != nil || r.pos+int(n) > len(r.data) {
		r.err = io.ErrUnexpectedEOF
		return ""
	}
	s := string(r.data[r.pos : r.pos+int(n)])
	r.pos += int(n)
	return s
}

func (r *readBuf) readGlobals() []GlobalVar {
	n := r.readU32()
	globals := make([]GlobalVar, n)
	for i := uint32(0); i < n; i++ {
		globals[i].Name = r.readString()
		globals[i].Type = r.readString()
		if r.readBool() {
			globals[i].InitVal = r.readExpr()
		}
	}
	return globals
}

func (r *readBuf) readParams() []ParamDecl {
	n := r.readU32()
	params := make([]ParamDecl, n)
	for i := uint32(0); i < n; i++ {
		params[i].Name = r.readString()
		params[i].Type = r.readString()
		if r.readBool() {
			params[i].Default = r.readExpr()
		}
	}
	return params
}

func (r *readBuf) readFuncs() map[string]*FuncDef {
	n := r.readU32()
	funcs := make(map[string]*FuncDef, n)
	for i := uint32(0); i < n; i++ {
		name := r.readString()
		fn := &FuncDef{
			Name:   name,
			Params: r.readParams(),
			Body:   r.readStatements(),
		}
		funcs[name] = fn
	}
	return funcs
}

func (r *readBuf) readEnums() map[string]int32 {
	n := r.readU32()
	enums := make(map[string]int32, n)
	for i := uint32(0); i < n; i++ {
		name := r.readString()
		val := int32(r.readU32())
		enums[name] = val
	}
	return enums
}

func (r *readBuf) readStatements() []Statement {
	n := r.readU32()
	stmts := make([]Statement, n)
	for i := uint32(0); i < n; i++ {
		stmts[i] = *r.readStatement()
	}
	return stmts
}

func (r *readBuf) readStatement() *Statement {
	s := &Statement{Kind: StatementKind(r.readU8())}
	if r.readBool() {
		s.Expr = r.readExpr()
	}
	if r.readBool() {
		s.Cond = r.readExpr()
	}
	if r.readBool() {
		s.Init = r.readStatement()
	}
	if r.readBool() {
		s.Update = r.readStatement()
	}
	s.Body = r.readStatements()
	s.ElseBody = r.readStatements()
	nc := r.readU32()
	s.Cases = make([]SwitchCase, nc)
	for i := uint32(0); i < nc; i++ {
		if r.readBool() {
			s.Cases[i].Expr = r.readExpr()
		}
		s.Cases[i].Body = r.readStatements()
	}
	return s
}

func (r *readBuf) readExpr() *Expr {
	e := &Expr{Kind: ExprKind(r.readU8())}
	e.Val = r.readValue()
	e.Name = r.readString()
	e.Op = r.readString()
	e.IsAssign = r.readBool()
	na := r.readU32()
	e.Args = make([]Expr, na)
	for i := uint32(0); i < na; i++ {
		e.Args[i] = *r.readExpr()
	}
	if r.readBool() {
		e.Index = r.readExpr()
	}
	if r.readBool() {
		e.Cond = r.readExpr()
	}
	if r.readBool() {
		e.ThenExpr = r.readExpr()
	}
	if r.readBool() {
		e.ElseExpr = r.readExpr()
	}
	return e
}

func (r *readBuf) readValue() Value {
	v := Value{Kind: ValueKind(r.readU8())}
	v.Int = int32(r.readU32())
	decStr := r.readString()
	if decStr != "" {
		d, err := decimal.NewFromString(decStr)
		if err == nil {
			v.Decimal = d
		}
	}
	v.Str = r.readString()
	v.Bool = r.readBool()
	v.Datetime = int64(r.readU64())
	na := r.readU32()
	v.Array = make([]Value, na)
	for i := uint32(0); i < na; i++ {
		v.Array[i] = r.readValue()
	}
	if r.readBool() {
		v.Class = &ClassInstance{
			Name:   r.readString(),
			Fields: make(map[string]Value),
		}
		nf := r.readU32()
		for j := uint32(0); j < nf; j++ {
			fname := r.readString()
			v.Class.Fields[fname] = r.readValue()
		}
	}
	return v
}
