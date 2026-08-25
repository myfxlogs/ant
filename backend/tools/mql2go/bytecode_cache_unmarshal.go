package mql2go

import (
	"fmt"

	"github.com/shopspring/decimal"

	"alphaforge/tools/mql2go/interp"
)

func unmarshalConsts(r *bytecodeReader) ([]ConstValue, error) {
	n, err := r.readCount(10, "consts")
	if err != nil {
		return nil, err
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
		dec, err := decimal.NewFromString(decStr)
		if err != nil {
			return nil, fmt.Errorf("bytecode: invalid const[%d] decimal %q: %w", i, decStr, err)
		}
		consts[i] = ConstValue{Kind: interp.ValueKind(kind), Int: intVal, Dec: dec, Str: str, Bool: b}
	}
	return consts, nil
}

func unmarshalCode(r *bytecodeReader) ([]Instruction, error) {
	n, err := r.readCount(13, "code")
	if err != nil {
		return nil, err
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
	n, err := r.readCount(4, "globalSlots")
	if err != nil {
		return nil, err
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
		if _, dup := slots[name]; dup {
			return nil, fmt.Errorf("bytecode: duplicate globalSlot key %q", name)
		}
		slots[name] = VarID(id)
	}
	return slots, nil
}

func unmarshalGlobalDecls(r *bytecodeReader) ([]interp.GlobalVar, error) {
	n, err := r.readCount(9, "globalDecls")
	if err != nil {
		return nil, err
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
	n, err := r.readCount(15, "funcs")
	if err != nil {
		return nil, err
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
		if _, dup := funcs[name]; dup {
			return nil, fmt.Errorf("bytecode: duplicate func key %q", name)
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
	n, err := r.readCount(4, "builtins")
	if err != nil {
		return nil, err
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
		if _, dup := builtins[name]; dup {
			return nil, fmt.Errorf("bytecode: duplicate builtin key %q", name)
		}
		builtins[name] = BuiltinID(id)
	}
	return builtins, nil
}
