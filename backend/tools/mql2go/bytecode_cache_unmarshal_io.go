package mql2go

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"alphaforge/tools/mql2go/interp"
)

// This file contains event/param/enum/class unmarshal functions and
// the binary reader/writer types, extracted from
// bytecode_cache_unmarshal.go for file-size compliance.

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
	n, err := r.readCount(8, "eventLocals")
	if err != nil {
		return err
	}
	seen := make(map[int32]bool, n)
	for i := uint32(0); i < n; i++ {
		pc, err := r.readI32()
		if err != nil {
			return fmt.Errorf("bytecode: read eventLocal[%d] pc: %w", i, err)
		}
		// VM-CACHE-INTEGRITY-3: reject duplicate PC keys.
		if seen[pc] {
			return fmt.Errorf("bytecode: duplicate eventLocal pc %d at index %d", pc, i)
		}
		seen[pc] = true
		count, err := r.readI32()
		if err != nil {
			return fmt.Errorf("bytecode: read eventLocal[%d] count: %w", i, err)
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
	if uint64(paramsLen) > uint64(len(r.data)-r.pos) {
		return fmt.Errorf("bytecode: params length %d exceeds remaining data", paramsLen)
	}
	paramsRaw := make([]byte, paramsLen)
	if _, err := r.readBytes(paramsRaw); err != nil {
		return fmt.Errorf("bytecode: read params data: %w", err)
	}
	bc.Params = interp.DeserializeParams(paramsRaw)
	if !bytes.Equal(interp.SerializeParams(bc.Params), paramsRaw) {
		return fmt.Errorf("bytecode: invalid params payload")
	}
	return nil
}

func unmarshalEnums(r *bytecodeReader) (map[string]int32, error) {
	n, err := r.readCount(6, "enums")
	if err != nil {
		return nil, err
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
		if _, dup := enums[name]; dup {
			return nil, fmt.Errorf("bytecode: duplicate enum key %q", name)
		}
		enums[name] = val
	}
	return enums, nil
}

func unmarshalClassTypes(r *bytecodeReader) (map[string]bool, error) {
	n, err := r.readCount(2, "class types")
	if err != nil {
		return nil, err
	}
	classTypes := make(map[string]bool, n)
	for i := uint32(0); i < n; i++ {
		name, err := r.readString()
		if err != nil {
			return nil, fmt.Errorf("bytecode: read class type[%d] name: %w", i, err)
		}
		if _, dup := classTypes[name]; dup {
			return nil, fmt.Errorf("bytecode: duplicate class type key %q", name)
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
