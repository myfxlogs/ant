// vm_builtin_diag.go — L2 indicator capture helpers for VM builtins.
// Records indicator values at builtin return points with zero-alloc memoized keys.
package mql2go

import (
	"strconv"

	"github.com/shopspring/decimal"
)

// Indicator name indices for diagHash. Used as first hash input to distinguish builtins.
const (
	diagNameMA    = 1
	diagNameMACD  = 2
	diagNameRSI   = 3
	diagNameATR   = 4
	diagNameBands = 5
	diagNameStoch = 6
	diagNameADX   = 7
)

// FNV-1a constants for zero-alloc hashing.
const (
	fnvOffset    = 1469598103934665603
	fnvPrime     = 1099511628211
)

// diagHash2 computes a zero-alloc FNV-1a hash of two int32 values.
func diagHash2(a, b int32) uint64 {
	h := uint64(fnvOffset)
	h ^= uint64(a); h *= fnvPrime
	h ^= uint64(b); h *= fnvPrime
	return h
}

// diagHash3 computes a zero-alloc FNV-1a hash of three int32 values.
func diagHash3(a, b, c int32) uint64 {
	h := uint64(fnvOffset)
	h ^= uint64(a); h *= fnvPrime
	h ^= uint64(b); h *= fnvPrime
	h ^= uint64(c); h *= fnvPrime
	return h
}

// diagHash4 computes a zero-alloc FNV-1a hash of four int32 values.
func diagHash4(a, b, c, d int32) uint64 {
	h := uint64(fnvOffset)
	h ^= uint64(a); h *= fnvPrime
	h ^= uint64(b); h *= fnvPrime
	h ^= uint64(c); h *= fnvPrime
	h ^= uint64(d); h *= fnvPrime
	return h
}

// diagHash5 computes a zero-alloc FNV-1a hash of five int32 values.
func diagHash5(a, b, c, d, e int32) uint64 {
	h := uint64(fnvOffset)
	h ^= uint64(a); h *= fnvPrime
	h ^= uint64(b); h *= fnvPrime
	h ^= uint64(c); h *= fnvPrime
	h ^= uint64(d); h *= fnvPrime
	h ^= uint64(e); h *= fnvPrime
	return h
}

// recordDiag records an indicator value to the VM's lastIndicators map.
// On cache hit (normal hot path), this is zero-allocation.
// On cache miss (first call per unique param set), the key string is built once and cached.
func recordDiag(vm *VM, hash uint64, key string, val decimal.Decimal) {
	if vm.lastIndicators == nil {
		return
	}
	if cached, ok := vm.diagKeyCache[hash]; ok {
		vm.lastIndicators[cached] = val
		return
	}
	vm.diagKeyCache[hash] = key
	vm.lastIndicators[key] = val
}

// diagKeyMA builds the display key for iMA indicators.
func diagKeyMA(period int, method string, appliedPrice int) string {
	return "iMA[" + strconv.Itoa(period) + "," + method + "," + strconv.Itoa(appliedPrice) + "]"
}

// diagKeyRSI builds the display key for iRSI indicators.
func diagKeyRSI(period, appliedPrice int) string {
	return "iRSI[" + strconv.Itoa(period) + "," + strconv.Itoa(appliedPrice) + "]"
}

// diagKeyATR builds the display key for iATR indicators.
func diagKeyATR(period int) string {
	return "iATR[" + strconv.Itoa(period) + "]"
}

// diagKeyMACD builds the display key for iMACD indicators with sub-line.
func diagKeyMACD(fast, slow, signal, appliedPrice int, sub string) string {
	return "iMACD[" + strconv.Itoa(fast) + "," + strconv.Itoa(slow) + "," +
		strconv.Itoa(signal) + "," + strconv.Itoa(appliedPrice) + "]." + sub
}

// diagKeyBands builds the display key for iBands indicators with sub-line.
func diagKeyBands(period int, deviation decimal.Decimal, appliedPrice int, sub string) string {
	return "iBands[" + strconv.Itoa(period) + "," + deviation.String() + "," +
		strconv.Itoa(appliedPrice) + "]." + sub
}

// diagKeyStoch builds the display key for iStochastic indicators with sub-line.
func diagKeyStoch(k, d, slowing int, sub string) string {
	return "iStoch[" + strconv.Itoa(k) + "," + strconv.Itoa(d) + "," +
		strconv.Itoa(slowing) + "]." + sub
}

// diagKeyADX builds the display key for iADX indicators.
func diagKeyADX(period int) string {
	return "iADX[" + strconv.Itoa(period) + "]"
}
