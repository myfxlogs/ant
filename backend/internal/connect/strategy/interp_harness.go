package strategy

import "encoding/binary"

// irLengthPrefix prepends a u32 little-endian length prefix to IR bytes,
// matching the protocol expected by WASMRunSetup / the interp harness.
func irLengthPrefix(irBytes []byte) []byte {
	buf := make([]byte, 4+len(irBytes))
	binary.LittleEndian.PutUint32(buf[:4], uint32(len(irBytes)))
	copy(buf[4:], irBytes)
	return buf
}
