// V1-LEGACY: will be replaced by M7.1-7.4 cards. Do not extend; new code goes alongside.
// Package factorsvc — factor computation engine.
package factorsvc

// Bar is a market data bar (local type, replaces protobuf Bar from alfq).
type Bar struct {
	UserID        string
	Symbol        string
	Period        string
	Open          string
	High          string
	Low           string
	Close         string
	Volume        float64
	CloseTsUnixMs int64
}
