package mql2go

import (
	"encoding/binary"
	"fmt"
	"io"
	"sort"

	"github.com/shopspring/decimal"

	"alphaforge/tools/mql2go/interp"
)

// Bytecode binary cache format (version 1).
// All integers are little-endian. Strings are uint16-length-prefixed.
// Slices/maps are uint32-count-prefixed.
//
// Layout:
//   magic: "BC01" (4 bytes)
//   compilerVersion: string (invalidates stale caches)
//   consts: count + entries
//   code: count + entries
//   globalSlots: count + entries
//   globalDecls: count + entries (InitVal omitted — not needed at runtime)
//   funcs: count + entries
//   builtins: count + entries
//   events: 6 × int32
//   eventLocals: count + entries
//   params: uint32 length + raw bytes (SerializeParams format)
//   version: string
//   sourceHash: string (VM-CACHE-INTEGRITY-1)
//   enums: count + entries
//   (Coverage omitted — not needed for execution)

const bytecodeMagic = "BC01"

// CompilerVersion is incremented when the bytecode format or compiler logic
// changes in a way that invalidates previously cached bytecode.
// This ensures stale caches from older compiler versions are rejected.
const CompilerVersion = "2026-07-02-v1"

// MarshalBytecode serializes a Bytecode to a compact binary format for DB storage.
// Coverage report is omitted (not needed for VM execution).
// VM-CACHE-INTEGRITY-1: returns error on nil bytecode (error must propagate,
// not be swallowed by caller).
func MarshalBytecode(bc *Bytecode) ([]byte, error) {
	if bc == nil {
		return nil, fmt.Errorf("marshal: nil bytecode")
	}
	w := &bytecodeWriter{buf: make([]byte, 0, 4096)}
	w.writeString(bytecodeMagic)
	w.writeString(CompilerVersion)

	// Consts
	w.writeU32(uint32(len(bc.Consts)))
	for _, c := range bc.Consts {
		w.writeU8(uint8(c.Kind))
		w.writeI32(c.Int)
		w.writeString(c.Dec.String())
		w.writeString(c.Str)
		w.writeBool(c.Bool)
	}

	// Code
	w.writeU32(uint32(len(bc.Code)))
	for _, ins := range bc.Code {
		w.writeU8(uint8(ins.Op))
		w.writeI32(ins.A)
		w.writeI32(ins.B)
		w.writeU32(ins.Line)
	}

	// GlobalSlots
	w.writeU32(uint32(len(bc.GlobalSlots)))
	for name, id := range bc.GlobalSlots {
		w.writeString(name)
		w.writeU16(uint16(id))
	}

	// GlobalDecls (InitVal omitted)
	w.writeU32(uint32(len(bc.GlobalDecls)))
	for _, g := range bc.GlobalDecls {
		w.writeString(g.Name)
		w.writeString(g.Type)
		w.writeBool(g.IsArray)
		w.writeI32(int32(g.ArraySize))
	}

	// Funcs
	w.writeU32(uint32(len(bc.Funcs)))
	for name, fn := range bc.Funcs {
		w.writeString(name)
		w.writeI32(fn.EntryPC)
		w.writeI32(int32(fn.NumParams))
		w.writeI32(int32(fn.NumLocals))
		w.writeU8(uint8(len(fn.ParamName)))
		for _, pn := range fn.ParamName {
			w.writeString(pn)
		}
	}

	// Builtins
	w.writeU32(uint32(len(bc.Builtins)))
	for name, id := range bc.Builtins {
		w.writeString(name)
		w.writeU16(uint16(id))
	}

	// Events
	w.writeI32(bc.OnInit)
	w.writeI32(bc.OnBar)
	w.writeI32(bc.OnTick)
	w.writeI32(bc.OnTrade)
	w.writeI32(bc.OnTimer)
	w.writeI32(bc.OnDeinit)
	w.writeI32(bc.OnTradeTransaction)
	w.writeI32(bc.OnBookEvent)

	// EventLocals
	w.writeU32(uint32(len(bc.EventLocals)))
	for pc, n := range bc.EventLocals {
		w.writeI32(pc)
		w.writeI32(int32(n))
	}

	// Params (use existing SerializeParams format)
	paramsRaw := interp.SerializeParams(bc.Params)
	w.writeU32(uint32(len(paramsRaw)))
	w.writeBytes(paramsRaw)

	// Version
	w.writeString(bc.Version)

	// SourceHash (VM-CACHE-INTEGRITY-1)
	w.writeString(bc.SourceHash)

	// Enums
	w.writeU32(uint32(len(bc.Enums)))
	for name, val := range bc.Enums {
		w.writeString(name)
		w.writeI32(val)
	}

	// ClassTypes (VM-COMPILER-SEMANTICS-1: sorted keys for deterministic serialization)
	classTypeNames := make([]string, 0, len(bc.ClassTypes))
	for name := range bc.ClassTypes {
		classTypeNames = append(classTypeNames, name)
	}
	sort.Strings(classTypeNames)
	w.writeU32(uint32(len(classTypeNames)))
	for _, name := range classTypeNames {
		w.writeString(name)
	}

	return w.buf, nil
}

// UnmarshalBytecode deserializes a Bytecode from the binary cache format.
func UnmarshalBytecode(data []byte) (*Bytecode, error) {
	r := &bytecodeReader{data: data}

	magic, err := r.readString()
	if err != nil {
		return nil, fmt.Errorf("bytecode: read magic: %w", err)
	}
	if magic != bytecodeMagic {
		return nil, fmt.Errorf("bytecode: invalid magic %q (expected %q)", magic, bytecodeMagic)
	}

	cachedVersion, err := r.readString()
	if err != nil {
		return nil, fmt.Errorf("bytecode: read compiler version: %w", err)
	}
	if cachedVersion != CompilerVersion {
		return nil, fmt.Errorf("bytecode: stale cache (compiler version %q, expected %q)", cachedVersion, CompilerVersion)
	}

	bc := &Bytecode{
		OnInit:             -1,
		OnBar:              -1,
		OnTick:             -1,
		OnTrade:            -1,
		OnTimer:            -1,
		OnDeinit:           -1,
		OnTradeTransaction: -1,
		OnBookEvent:        -1,
		EventLocals:        make(map[int32]int),
	}

	if bc.Consts, err = unmarshalConsts(r); err != nil {
		return nil, err
	}
	if bc.Code, err = unmarshalCode(r); err != nil {
		return nil, err
	}
	if bc.GlobalSlots, err = unmarshalGlobalSlots(r); err != nil {
		return nil, err
	}
	if bc.GlobalDecls, err = unmarshalGlobalDecls(r); err != nil {
		return nil, err
	}
	if bc.Funcs, err = unmarshalFuncs(r); err != nil {
		return nil, err
	}
	if bc.Builtins, err = unmarshalBuiltins(r); err != nil {
		return nil, err
	}
	if err = unmarshalEvents(r, bc); err != nil {
		return nil, err
	}
	if err = unmarshalEventLocals(r, bc); err != nil {
		return nil, err
	}
	if err = unmarshalParams(r, bc); err != nil {
		return nil, err
	}
	if bc.Version, err = r.readString(); err != nil {
		return nil, fmt.Errorf("bytecode: read version: %w", err)
	}
	if bc.SourceHash, err = r.readString(); err != nil {
		return nil, fmt.Errorf("bytecode: read sourceHash: %w", err)
	}
	if bc.Enums, err = unmarshalEnums(r); err != nil {
		return nil, err
	}
	// VM-COMPILER-SEMANTICS-1: ClassTypes
	if bc.ClassTypes, err = unmarshalClassTypes(r); err != nil {
		return nil, err
	}
	// VM-CACHE-INTEGRITY-1: reject trailing bytes to prevent corrupted cache
	// from producing non-deterministic output.
	if r.pos != len(r.data) {
		return nil, fmt.Errorf("bytecode: trailing bytes: %d", len(r.data)-r.pos)
	}
	return bc, nil
}

// readCount reads a uint32 count and verifies that count*minBytes does not
// exceed the remaining data. VM-CACHE-INTEGRITY-1: prevents corrupted cache
// from causing excessive allocation or non-deterministic output.
func (r *bytecodeReader) readCount(minBytes int) (uint32, error) {
	n, err := r.readU32()
	if err != nil {
		return 0, err
	}
	remaining := len(r.data) - r.pos
	if int(n) > remaining/minBytes {
		return 0, fmt.Errorf("bytecode: count %d exceeds remaining data (%d bytes left, min %d/entry)", n, remaining, minBytes)
	}
	return n, nil
}

func unmarshalConsts(r *bytecodeReader) ([]ConstValue, error) {
	n, err := r.readCount(10) // min 10 bytes/entry: u8 + i32 + u16(string len) + u16 + u8
	if err != nil {
		return nil, fmt.Errorf("bytecode: read consts count: %w", err)
	}
	consts := make([]ConstValue, n)
	for i := uint32(0); i < n; i++ {
		kind, err := r.readU8()
		if err != nil {
			return nil, fmt.Errorf("bytecode: read const[%d] kind: %w", i, err)
		}
		intVal, err := r.readI32()
		if err != nil {
			return nil, fmt.Errorf("bytecode: read const[%d] int: %w", i, err)
		}
		decStr, err := r.readString()
		if err != nil {
			return nil, fmt.Errorf("bytecode: read const[%d] dec: %w", i, err)
		}
		str, err := r.readString()
		if err != nil {
			return nil, fmt.Errorf("bytecode: read const[%d] str: %w", i, err)
		}
		b, err := r.readBool()
		if err != nil {
			return nil, fmt.Errorf("bytecode: read const[%d] bool: %w", i, err)
		}
		dec, _ := decimal.NewFromString(decStr)
		consts[i] = ConstValue{Kind: interp.ValueKind(kind), Int: intVal, Dec: dec, Str: str, Bool: b}
	}
	return consts, nil
}

func unmarshalCode(r *bytecodeReader) ([]Instruction, error) {
	n, err := r.readCount(13) // min 13 bytes/entry: u8 + i32 + i32 + u32
	if err != nil {
		return nil, fmt.Errorf("bytecode: read code count: %w", err)
	}
	code := make([]Instruction, n)
	for i := uint32(0); i < n; i++ {
		op, err := r.readU8()
		if err != nil {
			return nil, fmt.Errorf("bytecode: read code[%d] op: %w", i, err)
		}
		a, err := r.readI32()
		if err != nil {
			return nil, fmt.Errorf("bytecode: read code[%d] A: %w", i, err)
		}
		b, err := r.readI32()
		if err != nil {
			return nil, fmt.Errorf("bytecode: read code[%d] B: %w", i, err)
		}
		line, err := r.readU32()
		if err != nil {
			return nil, fmt.Errorf("bytecode: read code[%d] line: %w", i, err)
		}
		code[i] = Instruction{Op: Opcode(op), A: a, B: b, Line: line}
	}
	return code, nil
}

func unmarshalGlobalSlots(r *bytecodeReader) (map[string]VarID, error) {
	n, err := r.readU32()
	if err != nil {
		return nil, fmt.Errorf("bytecode: read globalSlots count: %w", err)
	}
	slots := make(map[string]VarID, n)
	for i := uint32(0); i < n; i++ {
		name, err := r.readString()
		if err != nil {
			return nil, fmt.Errorf("bytecode: read globalSlot[%d] name: %w", i, err)
		}
		id, err := r.readU16()
		if err != nil {
			return nil, fmt.Errorf("bytecode: read globalSlot[%d] id: %w", i, err)
		}
		if _, exists := slots[name]; exists {
			return nil, fmt.Errorf("bytecode: duplicate globalSlot key: %s", name)
		}
		slots[name] = VarID(id)
	}
	return slots, nil
}

func unmarshalGlobalDecls(r *bytecodeReader) ([]interp.GlobalVar, error) {
	n, err := r.readU32()
	if err != nil {
		return nil, fmt.Errorf("bytecode: read globalDecls count: %w", err)
	}
	decls := make([]interp.GlobalVar, n)
	for i := uint32(0); i < n; i++ {
		name, err := r.readString()
		if err != nil {
			return nil, fmt.Errorf("bytecode: read globalDecl[%d] name: %w", i, err)
		}
		typ, err := r.readString()
		if err != nil {
			return nil, fmt.Errorf("bytecode: read globalDecl[%d] type: %w", i, err)
		}
		isArray, err := r.readBool()
		if err != nil {
			return nil, fmt.Errorf("bytecode: read globalDecl[%d] isArray: %w", i, err)
		}
		arrSize, err := r.readI32()
		if err != nil {
			return nil, fmt.Errorf("bytecode: read globalDecl[%d] arrSize: %w", i, err)
		}
		decls[i] = interp.GlobalVar{Name: name, Type: typ, IsArray: isArray, ArraySize: int(arrSize)}
	}
	return decls, nil
}

func unmarshalFuncs(r *bytecodeReader) (map[string]FuncEntry, error) {
	n, err := r.readU32()
	if err != nil {
		return nil, fmt.Errorf("bytecode: read funcs count: %w", err)
	}
	funcs := make(map[string]FuncEntry, n)
	for i := uint32(0); i < n; i++ {
		name, err := r.readString()
		if err != nil {
			return nil, fmt.Errorf("bytecode: read func[%d] name: %w", i, err)
		}
		entryPC, err := r.readI32()
		if err != nil {
			return nil, fmt.Errorf("bytecode: read func[%d] entryPC: %w", i, err)
		}
		numParams, err := r.readI32()
		if err != nil {
			return nil, fmt.Errorf("bytecode: read func[%d] numParams: %w", i, err)
		}
		numLocals, err := r.readI32()
		if err != nil {
			return nil, fmt.Errorf("bytecode: read func[%d] numLocals: %w", i, err)
		}
		paramCount, err := r.readU8()
		if err != nil {
			return nil, fmt.Errorf("bytecode: read func[%d] paramCount: %w", i, err)
		}
		paramNames := make([]string, paramCount)
		for j := 0; j < int(paramCount); j++ {
			pn, err := r.readString()
			if err != nil {
				return nil, fmt.Errorf("bytecode: read func[%d] paramName[%d]: %w", i, j, err)
			}
			paramNames[j] = pn
		}
		if _, exists := funcs[name]; exists {
			return nil, fmt.Errorf("bytecode: duplicate func key: %s", name)
		}
		funcs[name] = FuncEntry{
			Name: name, EntryPC: entryPC,
			NumParams: int(numParams), NumLocals: int(numLocals),
			ParamName: paramNames,
		}
	}
	return funcs, nil
}

func unmarshalBuiltins(r *bytecodeReader) (map[string]BuiltinID, error) {
	n, err := r.readU32()
	if err != nil {
		return nil, fmt.Errorf("bytecode: read builtins count: %w", err)
	}
	builtins := make(map[string]BuiltinID, n)
	for i := uint32(0); i < n; i++ {
		name, err := r.readString()
		if err != nil {
			return nil, fmt.Errorf("bytecode: read builtin[%d] name: %w", i, err)
		}
		id, err := r.readU16()
		if err != nil {
			return nil, fmt.Errorf("bytecode: read builtin[%d] id: %w", i, err)
		}
		if _, exists := builtins[name]; exists {
			return nil, fmt.Errorf("bytecode: duplicate builtin key: %s", name)
		}
		builtins[name] = BuiltinID(id)
	}
	return builtins, nil
}

func unmarshalEvents(r *bytecodeReader, bc *Bytecode) error {
	var err error
	if bc.OnInit, err = r.readI32(); err != nil {
		return fmt.Errorf("bytecode: read OnInit: %w", err)
	}
	if bc.OnBar, err = r.readI32(); err != nil {
		return fmt.Errorf("bytecode: read OnBar: %w", err)
	}
	if bc.OnTick, err = r.readI32(); err != nil {
		return fmt.Errorf("bytecode: read OnTick: %w", err)
	}
	if bc.OnTrade, err = r.readI32(); err != nil {
		return fmt.Errorf("bytecode: read OnTrade: %w", err)
	}
	if bc.OnTimer, err = r.readI32(); err != nil {
		return fmt.Errorf("bytecode: read OnTimer: %w", err)
	}
	if bc.OnDeinit, err = r.readI32(); err != nil {
		return fmt.Errorf("bytecode: read OnDeinit: %w", err)
	}
	if bc.OnTradeTransaction, err = r.readI32(); err != nil {
		return fmt.Errorf("bytecode: read OnTradeTransaction: %w", err)
	}
	if bc.OnBookEvent, err = r.readI32(); err != nil {
		return fmt.Errorf("bytecode: read OnBookEvent: %w", err)
	}
	return nil
}

func unmarshalEventLocals(r *bytecodeReader, bc *Bytecode) error {
	n, err := r.readU32()
	if err != nil {
		return fmt.Errorf("bytecode: read eventLocals count: %w", err)
	}
	for i := uint32(0); i < n; i++ {
		pc, err := r.readI32()
		if err != nil {
			return fmt.Errorf("bytecode: read eventLocal[%d] pc: %w", i, err)
		}
		count, err := r.readI32()
		if err != nil {
			return fmt.Errorf("bytecode: read eventLocal[%d] count: %w", i, err)
		}
		if _, exists := bc.EventLocals[pc]; exists {
			return fmt.Errorf("bytecode: duplicate eventLocal pc: %d", pc)
		}
		bc.EventLocals[pc] = int(count)
	}
	return nil
}

func unmarshalParams(r *bytecodeReader, bc *Bytecode) error {
	paramsLen, err := r.readU32()
	if err != nil {
		return fmt.Errorf("bytecode: read params length: %w", err)
	}
	paramsRaw := make([]byte, paramsLen)
	if _, err := r.readBytes(paramsRaw); err != nil {
		return fmt.Errorf("bytecode: read params data: %w", err)
	}
	bc.Params = interp.DeserializeParams(paramsRaw)
	return nil
}

func unmarshalEnums(r *bytecodeReader) (map[string]int32, error) {
	n, err := r.readU32()
	if err != nil {
		return nil, fmt.Errorf("bytecode: read enums count: %w", err)
	}
	enums := make(map[string]int32, n)
	for i := uint32(0); i < n; i++ {
		name, err := r.readString()
		if err != nil {
			return nil, fmt.Errorf("bytecode: read enum[%d] name: %w", i, err)
		}
		val, err := r.readI32()
		if err != nil {
			return nil, fmt.Errorf("bytecode: read enum[%d] val: %w", i, err)
		}
		if _, exists := enums[name]; exists {
			return nil, fmt.Errorf("bytecode: duplicate enum key: %s", name)
		}
		enums[name] = val
	}
	return enums, nil
}

// unmarshalClassTypes reads the ClassTypes map from the bytecode cache.
// VM-COMPILER-SEMANTICS-1: used by UnmarshalBytecode to restore ClassTypes.
func unmarshalClassTypes(r *bytecodeReader) (map[string]bool, error) {
	n, err := r.readCount(2) // minBytes = 2 (u16 length prefix for string)
	if err != nil {
		return nil, fmt.Errorf("bytecode: read classTypes count: %w", err)
	}
	classTypes := make(map[string]bool, n)
	for i := uint32(0); i < n; i++ {
		name, err := r.readString()
		if err != nil {
			return nil, fmt.Errorf("bytecode: read classType[%d] name: %w", i, err)
		}
		if _, exists := classTypes[name]; exists {
			return nil, fmt.Errorf("bytecode: duplicate classType key: %s", name)
		}
		classTypes[name] = true
	}
	return classTypes, nil
}

// ── binary writer ────────────────────────────────────────────────────

type bytecodeWriter struct {
	buf []byte
}

func (w *bytecodeWriter) writeU8(v uint8) {
	w.buf = append(w.buf, v)
}

func (w *bytecodeWriter) writeU16(v uint16) {
	w.buf = binary.LittleEndian.AppendUint16(w.buf, v)
}

func (w *bytecodeWriter) writeU32(v uint32) {
	w.buf = binary.LittleEndian.AppendUint32(w.buf, v)
}

func (w *bytecodeWriter) writeI32(v int32) {
	w.buf = binary.LittleEndian.AppendUint32(w.buf, uint32(v))
}

func (w *bytecodeWriter) writeBool(v bool) {
	if v {
		w.buf = append(w.buf, 1)
	} else {
		w.buf = append(w.buf, 0)
	}
}

func (w *bytecodeWriter) writeString(s string) {
	w.writeU16(uint16(len(s)))
	w.buf = append(w.buf, s...)
}

func (w *bytecodeWriter) writeBytes(b []byte) {
	w.buf = append(w.buf, b...)
}

// ── binary reader ────────────────────────────────────────────────────

type bytecodeReader struct {
	data []byte
	pos  int
}

func (r *bytecodeReader) readU8() (uint8, error) {
	if r.pos >= len(r.data) {
		return 0, io.ErrUnexpectedEOF
	}
	v := r.data[r.pos]
	r.pos++
	return v, nil
}

func (r *bytecodeReader) readU16() (uint16, error) {
	if r.pos+2 > len(r.data) {
		return 0, io.ErrUnexpectedEOF
	}
	v := binary.LittleEndian.Uint16(r.data[r.pos:])
	r.pos += 2
	return v, nil
}

func (r *bytecodeReader) readU32() (uint32, error) {
	if r.pos+4 > len(r.data) {
		return 0, io.ErrUnexpectedEOF
	}
	v := binary.LittleEndian.Uint32(r.data[r.pos:])
	r.pos += 4
	return v, nil
}

func (r *bytecodeReader) readI32() (int32, error) {
	v, err := r.readU32()
	return int32(v), err
}

func (r *bytecodeReader) readBool() (bool, error) {
	b, err := r.readU8()
	return b != 0, err
}

func (r *bytecodeReader) readString() (string, error) {
	length, err := r.readU16()
	if err != nil {
		return "", err
	}
	if r.pos+int(length) > len(r.data) {
		return "", io.ErrUnexpectedEOF
	}
	s := string(r.data[r.pos : r.pos+int(length)])
	r.pos += int(length)
	return s, nil
}

func (r *bytecodeReader) readBytes(dst []byte) (int, error) {
	if r.pos+len(dst) > len(r.data) {
		return 0, io.ErrUnexpectedEOF
	}
	n := copy(dst, r.data[r.pos:])
	r.pos += n
	return n, nil
}
