// Package mdtick provides shared market data types for mdgateway adapters.
// Extracted from adapter/mt4 and adapter/mt5 to eliminate ~140 lines of
// duplicated type definitions. Both mt4 and mt5 adapters import this package.
package mdtick

// AccountConfig mirrors mdgateway.AccountConfig fields needed by adapters.
type AccountConfig struct {
	Broker   string
	Login    string
	Password string
	Server   string
	Host     string
	Port     string
}

// Money is a monetary value with string-preserved decimal precision.
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
	UserID        string
	Broker        string
	Symbol        string
	Canonical     string
	TsUnixMs      int64
	ArrivedUnixMs int64
	Bid           *Money
	Ask           *Money
	BidVolume     float64
	AskVolume     float64
}

// TickHandler is called for each normalized Tick.
type TickHandler func(tick *Tick)

// CanonicalResolver resolves (broker, symbol_raw) -> canonical name.
type CanonicalResolver interface {
	Resolve(brokerID, symbolRaw string) string
}

// Normalizer converts broker-specific quote types to Tick.
type Normalizer struct {
	Resolver CanonicalResolver
}

// Tick creates a Tick with common fields filled, including canonical name.
func (n *Normalizer) Tick(userID, broker, symbol string, tsMs int64, bid, ask string) *Tick {
	canon := symbol
	if n.Resolver != nil {
		canon = n.Resolver.Resolve(broker, symbol)
	}
	return &Tick{
		UserID:    userID,
		Broker:    broker,
		Symbol:    symbol,
		Canonical: canon,
		TsUnixMs:  tsMs,
		Bid:       &Money{Value: bid},
		Ask:       &Money{Value: ask},
	}
}
