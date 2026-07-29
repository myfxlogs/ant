// Package symbol provides canonical symbol resolution and seeding for ant.
// Canonical symbols are the normalised, broker-independent names (e.g. BTCUSD).
// Each broker maps its native names (BTCUSDm, EURUSD.x, etc.) to canonical via broker_symbols.
package symbol

import "strings"

const (
	cEUR    = "EUR"
	cJPY    = "JPY"
	cUSD    = "USD"
	cCrypto = "crypto"
	cForex  = "forex"
	cIndex  = "index"
)

// Canonicalize converts a broker-specific symbol name to canonical form.
// Rules:
//  1. Uppercase
//  2. Strip known suffixes: .ecn .raw .pro .stp .m .i .x and bare M I suffixes
//  3. Strip trailing # !
//
// .c suffix is NOT stripped — different contract, needs human review.
func Canonicalize(raw string) string {
	raw = strings.ToUpper(raw)

	// Dotted suffixes (ordered by specificity — longer first)
	for _, s := range []string{".ECN", ".RAW", ".PRO", ".STP", ".M", ".I", ".X"} {
		if strings.HasSuffix(raw, s) && len(raw) > len(s)+4 {
			return strings.TrimSuffix(raw, s)
		}
	}

	// Bare suffixes — only strip if preceding char is a letter (not digit)
	for _, s := range []string{"M", "I"} {
		if strings.HasSuffix(raw, s) && len(raw) > len(s)+5 {
			prev := raw[len(raw)-len(s)-1]
			if prev >= 'A' && prev <= 'Z' {
				return strings.TrimSuffix(raw, s)
			}
		}
	}

	// Trailing special chars
	if last := raw[len(raw)-1]; last == '#' || last == '!' {
		if len(raw) > 6 {
			return raw[:len(raw)-1]
		}
	}

	return raw
}

// CanonicalEntry is a row in the canonical_symbols seed table.
type CanonicalEntry struct {
	Canonical   string
	AssetClass  string // forex, crypto, commodity, index, stock
	BaseCCY     string
	QuoteCCY    string
	Description string
}

// SeedCanonicals returns the ~50 mainstream canonical symbols.
// Used by M1-3 seed script and the seed_strategy_templates command.
func SeedCanonicals() []CanonicalEntry {
	return []CanonicalEntry{
		// ── Forex Majors ──
		{Canonical: "EURUSD", AssetClass: cForex, BaseCCY: cEUR, QuoteCCY: cUSD, Description: "欧元/美元"},
		{Canonical: "GBPUSD", AssetClass: cForex, BaseCCY: "GBP", QuoteCCY: cUSD, Description: "英镑/美元"},
		{Canonical: "USDJPY", AssetClass: cForex, BaseCCY: cUSD, QuoteCCY: cJPY, Description: "美元/日元"},
		{Canonical: "USDCHF", AssetClass: cForex, BaseCCY: cUSD, QuoteCCY: "CHF", Description: "美元/瑞郎"},
		{Canonical: "AUDUSD", AssetClass: cForex, BaseCCY: "AUD", QuoteCCY: cUSD, Description: "澳元/美元"},
		{Canonical: "USDCAD", AssetClass: cForex, BaseCCY: cUSD, QuoteCCY: "CAD", Description: "美元/加元"},
		{Canonical: "NZDUSD", AssetClass: cForex, BaseCCY: "NZD", QuoteCCY: cUSD, Description: "纽元/美元"},
		{Canonical: "EURGBP", AssetClass: cForex, BaseCCY: cEUR, QuoteCCY: "GBP", Description: "欧元/英镑"},
		{Canonical: "EURJPY", AssetClass: cForex, BaseCCY: cEUR, QuoteCCY: cJPY, Description: "欧元/日元"},
		{Canonical: "GBPJPY", AssetClass: cForex, BaseCCY: "GBP", QuoteCCY: cJPY, Description: "英镑/日元"},

		// ── Forex Minors ──
		{Canonical: "EURCHF", AssetClass: cForex, BaseCCY: cEUR, QuoteCCY: "CHF", Description: "欧元/瑞郎"},
		{Canonical: "EURAUD", AssetClass: cForex, BaseCCY: cEUR, QuoteCCY: "AUD", Description: "欧元/澳元"},
		{Canonical: "GBPCHF", AssetClass: cForex, BaseCCY: "GBP", QuoteCCY: "CHF", Description: "英镑/瑞郎"},
		{Canonical: "GBPAUD", AssetClass: cForex, BaseCCY: "GBP", QuoteCCY: "AUD", Description: "英镑/澳元"},
		{Canonical: "AUDJPY", AssetClass: cForex, BaseCCY: "AUD", QuoteCCY: cJPY, Description: "澳元/日元"},
		{Canonical: "NZDJPY", AssetClass: cForex, BaseCCY: "NZD", QuoteCCY: cJPY, Description: "纽元/日元"},
		{Canonical: "CADJPY", AssetClass: cForex, BaseCCY: "CAD", QuoteCCY: cJPY, Description: "加元/日元"},
		{Canonical: "CHFJPY", AssetClass: cForex, BaseCCY: "CHF", QuoteCCY: cJPY, Description: "瑞郎/日元"},
		{Canonical: "EURNZD", AssetClass: cForex, BaseCCY: cEUR, QuoteCCY: "NZD", Description: "欧元/纽元"},
		{Canonical: "AUDNZD", AssetClass: cForex, BaseCCY: "AUD", QuoteCCY: "NZD", Description: "澳元/纽元"},

		// ── Commodities ──
		{Canonical: "XAUUSD", AssetClass: "commodity", BaseCCY: "XAU", QuoteCCY: cUSD, Description: "黄金/美元"},
		{Canonical: "XAGUSD", AssetClass: "commodity", BaseCCY: "XAG", QuoteCCY: cUSD, Description: "白银/美元"},
		{Canonical: "XTIUSD", AssetClass: "commodity", BaseCCY: "XTI", QuoteCCY: cUSD, Description: "WTI原油"},
		{Canonical: "XBRUSD", AssetClass: "commodity", BaseCCY: "XBR", QuoteCCY: cUSD, Description: "布伦特原油"},
		{Canonical: "XNGUSD", AssetClass: "commodity", BaseCCY: "XNG", QuoteCCY: cUSD, Description: "天然气"},

		// ── Indices ──
		{Canonical: "US30", AssetClass: cIndex, BaseCCY: "US30", QuoteCCY: cUSD, Description: "道琼斯工业指数"},
		{Canonical: "US100", AssetClass: cIndex, BaseCCY: "US100", QuoteCCY: cUSD, Description: "纳斯达克100"},
		{Canonical: "US500", AssetClass: cIndex, BaseCCY: "US500", QuoteCCY: cUSD, Description: "标普500"},
		{Canonical: "GER40", AssetClass: cIndex, BaseCCY: "GER40", QuoteCCY: "EUR", Description: "德国DAX40"},
		{Canonical: "UK100", AssetClass: cIndex, BaseCCY: "UK100", QuoteCCY: "GBP", Description: "英国富时100"},
		{Canonical: "JPN225", AssetClass: cIndex, BaseCCY: "JPN225", QuoteCCY: cJPY, Description: "日经225"},
		{Canonical: "HK50", AssetClass: cIndex, BaseCCY: "HK50", QuoteCCY: "HKD", Description: "恒生指数"},
		{Canonical: "AUS200", AssetClass: cIndex, BaseCCY: "AUS200", QuoteCCY: "AUD", Description: "澳洲ASX200"},
		{Canonical: "FRA40", AssetClass: cIndex, BaseCCY: "FRA40", QuoteCCY: "EUR", Description: "法国CAC40"},
		{Canonical: "EU50", AssetClass: cIndex, BaseCCY: "EU50", QuoteCCY: "EUR", Description: "欧洲斯托克50"},

		// ── Crypto ──
		{Canonical: "BTCUSD", AssetClass: cCrypto, BaseCCY: "BTC", QuoteCCY: cUSD, Description: "比特币/美元"},
		{Canonical: "ETHUSD", AssetClass: cCrypto, BaseCCY: "ETH", QuoteCCY: cUSD, Description: "以太坊/美元"},
		{Canonical: "XRPUSD", AssetClass: cCrypto, BaseCCY: "XRP", QuoteCCY: cUSD, Description: "瑞波币/美元"},
		{Canonical: "LTCUSD", AssetClass: cCrypto, BaseCCY: "LTC", QuoteCCY: cUSD, Description: "莱特币/美元"},
		{Canonical: "BCHUSD", AssetClass: cCrypto, BaseCCY: "BCH", QuoteCCY: cUSD, Description: "比特币现金/美元"},
		{Canonical: "SOLUSD", AssetClass: cCrypto, BaseCCY: "SOL", QuoteCCY: cUSD, Description: "Solana/美元"},
		{Canonical: "DOGEUSD", AssetClass: cCrypto, BaseCCY: "DOGE", QuoteCCY: cUSD, Description: "狗狗币/美元"},
		{Canonical: "ADAUSD", AssetClass: cCrypto, BaseCCY: "ADA", QuoteCCY: cUSD, Description: "Cardano/美元"},
		{Canonical: "DOTUSD", AssetClass: cCrypto, BaseCCY: "DOT", QuoteCCY: cUSD, Description: "Polkadot/美元"},
		{Canonical: "AVAXUSD", AssetClass: cCrypto, BaseCCY: "AVAX", QuoteCCY: cUSD, Description: "Avalanche/美元"},
	}
}
