//go:build !cgo

package mql2go

// ParseMQL fallback when cgo is not available.
func ParseMQL(source string) (*SourceFile, error) {
	return nil, nil
}
