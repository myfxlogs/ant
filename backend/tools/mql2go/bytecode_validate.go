package mql2go

import (
	"fmt"
	"sort"

	"alphaforge/tools/mql2go/interp"
)

const (
	maxBytecodeStringLen = 1<<16 - 1
	maxParamCount        = 1<<8 - 1
)

func validateBytecode(bc *Bytecode) error {
	if bc == nil {
		return fmt.Errorf("nil bytecode")
	}
	if len(bc.Code) > int(^uint32(0)>>1) {
		return fmt.Errorf("too many instructions: %d", len(bc.Code))
	}
	if err := validateString("source hash", bc.SourceHash, maxBytecodeStringLen); err != nil {
		return err
	}
	for i, c := range bc.Consts {
		if c.Kind > interp.ValClass {
			return fmt.Errorf("const[%d] has invalid kind %d", i, c.Kind)
		}
		if c.Kind == interp.ValArray || c.Kind == interp.ValClass {
			return fmt.Errorf("const[%d] has non-serializable kind %d", i, c.Kind)
		}
		if err := validateString(fmt.Sprintf("const[%d].decimal", i), c.Dec.String(), maxBytecodeStringLen); err != nil {
			return err
		}
		if err := validateString(fmt.Sprintf("const[%d].string", i), c.Str, maxBytecodeStringLen); err != nil {
			return err
		}
	}

	seenSlots := make(map[VarID]bool, len(bc.GlobalSlots))
	for name, id := range bc.GlobalSlots {
		if err := validateString("global slot name", name, maxBytecodeStringLen); err != nil {
			return err
		}
		if int(id) >= len(bc.GlobalSlots) || seenSlots[id] {
			return fmt.Errorf("global slot %q has invalid or duplicate id %d", name, id)
		}
		seenSlots[id] = true
	}
	for i, decl := range bc.GlobalDecls {
		if err := validateString(fmt.Sprintf("global declaration[%d] name", i), decl.Name, maxBytecodeStringLen); err != nil {
			return err
		}
		if err := validateString(fmt.Sprintf("global declaration[%d] type", i), decl.Type, maxBytecodeStringLen); err != nil {
			return err
		}
		if decl.ArraySize < 0 {
			return fmt.Errorf("global declaration[%d] has negative array size", i)
		}
	}
	if len(bc.Params) > maxParamCount {
		return fmt.Errorf("too many parameters: %d", len(bc.Params))
	}
	for i, p := range bc.Params {
		for label, value := range map[string]string{"name": p.Name, "type": p.Type} {
			if err := validateString(fmt.Sprintf("param[%d].%s", i, label), value, maxParamCount); err != nil {
				return err
			}
		}
		if p.Default != nil {
			if err := validateString(fmt.Sprintf("param[%d].default", i), interp.EvalExprLiteral(p.Default), maxParamCount); err != nil {
				return err
			}
		}
	}

	entryByPC := make(map[int32]string, len(bc.Funcs))
	for name, fn := range bc.Funcs {
		if err := validateString("function name", name, maxBytecodeStringLen); err != nil {
			return err
		}
		if err := validateString("function stored name", fn.Name, maxBytecodeStringLen); err != nil {
			return err
		}
		if fn.EntryPC < 0 || fn.EntryPC >= int32(len(bc.Code)) {
			return fmt.Errorf("function %q has invalid entry pc %d", name, fn.EntryPC)
		}
		if fn.NumParams < 0 || fn.NumLocals < fn.NumParams || fn.NumLocals > MaxStackDepth {
			return fmt.Errorf("function %q has invalid local counts params=%d locals=%d", name, fn.NumParams, fn.NumLocals)
		}
		if len(fn.ParamName) > maxParamCount {
			return fmt.Errorf("function %q has too many parameter names", name)
		}
		if previous, exists := entryByPC[fn.EntryPC]; exists {
			return fmt.Errorf("functions %q and %q share entry pc %d", previous, name, fn.EntryPC)
		}
		entryByPC[fn.EntryPC] = name
		for i, paramName := range fn.ParamName {
			if err := validateString(fmt.Sprintf("function %q param[%d]", name, i), paramName, maxBytecodeStringLen); err != nil {
				return err
			}
		}
	}
	for name, id := range bc.Builtins {
		if err := validateString("builtin name", name, maxBytecodeStringLen); err != nil {
			return err
		}
		if int(id) >= len(builtinRegistry) {
			return fmt.Errorf("builtin %q has invalid id %d", name, id)
		}
	}
	for name, value := range bc.Enums {
		if err := validateString("enum name", name, maxBytecodeStringLen); err != nil {
			return err
		}
		_ = value
	}

	for i, ins := range bc.Code {
		if err := validateInstruction(ins, i, len(bc.Consts), len(bc.GlobalSlots), len(bc.Code)); err != nil {
			return err
		}
		if ins.Op == OP_CALL_USER {
			if _, ok := entryByPC[ins.A]; !ok {
				return fmt.Errorf("instruction %d calls unknown function entry %d", i, ins.A)
			}
		}
	}
	for name, entry := range map[string]int32{
		"OnInit": bc.OnInit, "OnBar": bc.OnBar, "OnTick": bc.OnTick,
		"OnTrade": bc.OnTrade, "OnTimer": bc.OnTimer, "OnDeinit": bc.OnDeinit,
		"OnTradeTransaction": bc.OnTradeTransaction, "OnBookEvent": bc.OnBookEvent,
	} {
		if err := validateEntryPoint(name, entry, len(bc.Code)); err != nil {
			return err
		}
	}
	for pc, count := range bc.EventLocals {
		if err := validateEntryPoint("event locals", pc, len(bc.Code)); err != nil {
			return err
		}
		if count < 0 || count > MaxStackDepth {
			return fmt.Errorf("event locals at pc %d has invalid count %d", pc, count)
		}
	}
	return nil
}

func validateInstruction(ins Instruction, index, constCount, globalCount, codeCount int) error {
	indexOf := func(value, length int, operand string) error {
		if value < 0 || value >= length {
			return fmt.Errorf("instruction %d has invalid %s %d", index, operand, value)
		}
		return nil
	}
	nonNegative := func(value int32, operand string) error {
		if value < 0 {
			return fmt.Errorf("instruction %d has negative %s %d", index, operand, value)
		}
		return nil
	}
	switch ins.Op {
	case OP_PUSH_CONST:
		return indexOf(int(ins.A), constCount, "constant id")
	case OP_PUSH_GLOBAL, OP_STORE_GLOBAL, OP_PUSH_ARRAY, OP_STORE_ARRAY:
		return indexOf(int(ins.A), globalCount, "global slot")
	case OP_CALL_BUILTIN:
		if err := indexOf(int(ins.A), len(builtinRegistry), "builtin id"); err != nil {
			return err
		}
		return nonNegative(ins.B, "builtin argument count")
	case OP_CALL_USER:
		if err := nonNegative(ins.A, "user-call target"); err != nil {
			return err
		}
		return nonNegative(ins.B, "user-call argument count")
	case OP_JMP, OP_JMP_IF_FALSE, OP_JMP_IF_TRUE:
		return indexOf(int(ins.A), codeCount, "jump target")
	case OP_PUSH_SERIES:
		return indexOf(int(ins.B), constCount, "series constant id")
	case OP_GET_FIELD, OP_SET_FIELD:
		return indexOf(int(ins.A), constCount, "field constant id")
	case OP_PUSH_VAR, OP_STORE_VAR:
		return nonNegative(ins.A, "local slot")
	case OP_ENTER_FUNC:
		return nonNegative(ins.A, "function local count")
	case OP_ENTER_ONINIT, OP_ENTER_ONBAR, OP_ENTER_ONTICK, OP_ENTER_ONTRADE,
		OP_ENTER_ONTIMER, OP_ENTER_ONDEINIT, OP_ENTER_ONTRADETRANSACTION,
		OP_ENTER_ONBOOKEVENT, OP_RETURN, OP_POP, OP_DUP, OP_SWAP, OP_ADD,
		OP_SUB, OP_MUL, OP_DIV,
		OP_MOD, OP_FLOOR_DIV, OP_NEG, OP_EQ, OP_NE, OP_LT, OP_LE, OP_GT,
		OP_GE, OP_AND, OP_OR, OP_NOT, OP_LEAVE_FUNC, OP_HALT:
		return nil
	default:
		return fmt.Errorf("instruction %d has invalid opcode %d", index, ins.Op)
	}
}

func validateEntryPoint(name string, entry int32, codeLen int) error {
	if entry == -1 {
		return nil
	}
	if entry < 0 || entry >= int32(codeLen) {
		return fmt.Errorf("%s has invalid entry pc %d", name, entry)
	}
	return nil
}

func validateString(label, value string, maxLen int) error {
	if len(value) > maxLen {
		return fmt.Errorf("%s is too long: %d bytes (max %d)", label, len(value), maxLen)
	}
	return nil
}

func sortedVarNames(values map[string]VarID) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sortedFuncNames(values map[string]FuncEntry) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sortedBuiltinNames(values map[string]BuiltinID) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sortedEventPCs(values map[int32]int) []int32 {
	keys := make([]int32, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	return keys
}

func sortedEnumNames(values map[string]int32) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sortedClassTypeNames(values map[string]bool) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// maxBytecodeCount is the maximum number of entries in any single section
// of the bytecode cache. VM-CACHE-INTEGRITY-3: prevents memory exhaustion
// from corrupt/malicious bytecode with absurdly large counts.
const maxBytecodeCount = 1 << 20 // 1,048,576

func (r *bytecodeReader) readCount(minBytes int, label string) (uint32, error) {
	n, err := r.readU32()
	if err != nil {
		return 0, fmt.Errorf("bytecode: read %s count: %w", label, err)
	}
	// VM-CACHE-INTEGRITY-3: absolute upper bound on section entry count.
	if n > maxBytecodeCount {
		return 0, fmt.Errorf("bytecode: %s count %d exceeds max %d", label, n, maxBytecodeCount)
	}
	remaining := len(r.data) - r.pos
	if minBytes > 0 && uint64(n)*uint64(minBytes) > uint64(remaining) {
		return 0, fmt.Errorf("bytecode: %s count %d exceeds remaining data", label, n)
	}
	return n, nil
}
