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
