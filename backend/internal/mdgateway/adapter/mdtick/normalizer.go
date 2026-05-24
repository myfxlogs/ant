// Package mdtick — CanonicalResolver interface for normalizer.
package mdtick

// CanonicalResolver resolves (broker, symbol_raw) → canonical symbol name.
// Implemented by mdgateway/normalizer.go.
type CanonicalResolver interface {
	Resolve(broker, raw string) string
}
