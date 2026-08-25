package mql2go

import (
	"fmt"

	"alphaforge/tools/mql2go/interp"
)

// Bytecode binary cache format (version 2).
// All integers are little-endian. Strings are uint16-length-prefixed.
// Slices/maps are uint32-count-prefixed.
//
// Layout:
//   magic: "BC01" (4 bytes)
//   compilerVersion: string (invalidates stale caches)
//   sourceHash: string (rejects bytecode compiled from different source)
//   consts: count + entries
//   code: count + entries
//   globalSlots: count + entries
//   globalDecls: count + entries (InitVal omitted — not needed at runtime)
//   funcs: count + entries
//   builtins: count + entries
//   events: 8 × int32
//   eventLocals: count + entries
//   params: uint32 length + raw bytes (SerializeParams format)
//   version: string
//   enums: count + entries
//   (Coverage omitted — not needed for execution)

const bytecodeMagic = "BC01"

// CompilerVersion is incremented when the bytecode format or compiler logic
// changes in a way that invalidates previously cached bytecode.
// This ensures stale caches from older compiler versions are rejected.
const CompilerVersion = "2026-08-24-v3"

// MarshalBytecode serializes a Bytecode to a compact binary format for DB storage.
// Coverage report is omitted (not needed for VM execution).
func MarshalBytecode(bc *Bytecode) ([]byte, error) {
	if err := validateBytecode(bc); err != nil {
		return nil, fmt.Errorf("bytecode: cannot marshal: %w", err)
	}
	w := &bytecodeWriter{buf: make([]byte, 0, 4096)}
	w.writeString(bytecodeMagic)
	w.writeString(CompilerVersion)
	w.writeString(bc.SourceHash)

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
	for _, name := range sortedVarNames(bc.GlobalSlots) {
		w.writeString(name)
		w.writeU16(uint16(bc.GlobalSlots[name]))
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
	for _, name := range sortedFuncNames(bc.Funcs) {
		fn := bc.Funcs[name]
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
	for _, name := range sortedBuiltinNames(bc.Builtins) {
		w.writeString(name)
		w.writeU16(uint16(bc.Builtins[name]))
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
	for _, pc := range sortedEventPCs(bc.EventLocals) {
		w.writeI32(pc)
		w.writeI32(int32(bc.EventLocals[pc]))
	}

	// Params (use existing SerializeParams format)
	paramsRaw := interp.SerializeParams(bc.Params)
	w.writeU32(uint32(len(paramsRaw)))
	w.writeBytes(paramsRaw)

	// Version
	w.writeString(bc.Version)

	// Enums
	w.writeU32(uint32(len(bc.Enums)))
	for _, name := range sortedEnumNames(bc.Enums) {
		w.writeString(name)
		w.writeI32(bc.Enums[name])
	}

	// ClassTypes
	w.writeU32(uint32(len(bc.ClassTypes)))
	for _, name := range sortedClassTypeNames(bc.ClassTypes) {
		w.writeString(name)
	}

	return w.buf, nil
}

// UnmarshalBytecode deserializes a Bytecode from the binary cache format.
// maxBytecodePayload is the maximum total size of a bytecode cache payload.
// VM-CACHE-INTEGRITY-5: prevents memory exhaustion from corrupt/malicious
// bytecode with absurdly large total payload (even if individual section
// counts are within limits, the total could be enormous).
const maxBytecodePayload = 64 << 20 // 64 MiB

func UnmarshalBytecode(data []byte) (*Bytecode, error) {
	// VM-CACHE-INTEGRITY-5: total payload size limit.
	if len(data) > maxBytecodePayload {
		return nil, fmt.Errorf("bytecode: payload size %d exceeds max %d", len(data), maxBytecodePayload)
	}
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

	sourceHash, err := r.readString()
	if err != nil {
		return nil, fmt.Errorf("bytecode: read source hash: %w", err)
	}

	bc := &Bytecode{
		SourceHash:         sourceHash,
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
	if bc.Enums, err = unmarshalEnums(r); err != nil {
		return nil, err
	}
	if bc.ClassTypes, err = unmarshalClassTypes(r); err != nil {
		return nil, err
	}
	if r.pos != len(data) {
		return nil, fmt.Errorf("bytecode: trailing data (%d bytes)", len(data)-r.pos)
	}
	if err := validateBytecode(bc); err != nil {
		return nil, fmt.Errorf("bytecode: invalid program: %w", err)
	}
	return bc, nil
}
