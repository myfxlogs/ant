// Package mdgateway — shared market data types.
// Replaces proto-generated pb.Tick / pb.Bar / pb.Money from alfq.
package mdgateway

// Money represents a monetary value with string-preserved decimal precision.
type Money struct {
	Value string
}

// GetValue returns the string value (compat with proto getter pattern).
func (m *Money) GetValue() string {
	if m == nil {
		return ""
	}
	return m.Value
}

// Tick is a normalized market data tick.
type Tick struct {
	UserID       string
	Broker       string
	Symbol       string
	Canonical    string
	TsUnixMs     int64
	ArrivedUnixMs int64
	Bid          *Money
	Ask          *Money
	BidVolume    float64
	AskVolume    float64
}

// GetBid returns the bid money (compat with proto getter pattern).
func (t *Tick) GetBid() *Money {
	if t == nil {
		return nil
	}
	return t.Bid
}

// GetAsk returns the ask money (compat with proto getter pattern).
func (t *Tick) GetAsk() *Money {
	if t == nil {
		return nil
	}
	return t.Ask
}

// Bar is a completed OHLCV bar.
type Bar struct {
	UserID        string
	Broker        string
	SymbolRaw     string
	Canonical     string
	Period        string
	OpenTsUnixMs  int64
	CloseTsUnixMs int64
	Open          float64
	High          float64
	Low           float64
	Close         float64
	Volume        float64
	TickCount     uint32
}
