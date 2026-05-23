// Package symbol provides canonical symbol resolution and seeding for ant.
// Canonical symbols are the normalised, broker-independent names (e.g. BTCUSD).
// Each broker maps its native names (BTCUSDm, EURUSD.x, etc.) to canonical via broker_symbols.
package symbol

import "strings"

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
		{Canonical: "EURUSD", AssetClass: "forex", BaseCCY: "EUR", QuoteCCY: "USD", Description: "欧元/美元"},
		{Canonical: "GBPUSD", AssetClass: "forex", BaseCCY: "GBP", QuoteCCY: "USD", Description: "英镑/美元"},
		{Canonical: "USDJPY", AssetClass: "forex", BaseCCY: "USD", QuoteCCY: "JPY", Description: "美元/日元"},
		{Canonical: "USDCHF", AssetClass: "forex", BaseCCY: "USD", QuoteCCY: "CHF", Description: "美元/瑞郎"},
		{Canonical: "AUDUSD", AssetClass: "forex", BaseCCY: "AUD", QuoteCCY: "USD", Description: "澳元/美元"},
		{Canonical: "USDCAD", AssetClass: "forex", BaseCCY: "USD", QuoteCCY: "CAD", Description: "美元/加元"},
		{Canonical: "NZDUSD", AssetClass: "forex", BaseCCY: "NZD", QuoteCCY: "USD", Description: "纽元/美元"},
		{Canonical: "EURGBP", AssetClass: "forex", BaseCCY: "EUR", QuoteCCY: "GBP", Description: "欧元/英镑"},
		{Canonical: "EURJPY", AssetClass: "forex", BaseCCY: "EUR", QuoteCCY: "JPY", Description: "欧元/日元"},
		{Canonical: "GBPJPY", AssetClass: "forex", BaseCCY: "GBP", QuoteCCY: "JPY", Description: "英镑/日元"},

		// ── Forex Minors ──
		{Canonical: "EURCHF", AssetClass: "forex", BaseCCY: "EUR", QuoteCCY: "CHF", Description: "欧元/瑞郎"},
		{Canonical: "EURAUD", AssetClass: "forex", BaseCCY: "EUR", QuoteCCY: "AUD", Description: "欧元/澳元"},
		{Canonical: "GBPCHF", AssetClass: "forex", BaseCCY: "GBP", QuoteCCY: "CHF", Description: "英镑/瑞郎"},
		{Canonical: "GBPAUD", AssetClass: "forex", BaseCCY: "GBP", QuoteCCY: "AUD", Description: "英镑/澳元"},
		{Canonical: "AUDJPY", AssetClass: "forex", BaseCCY: "AUD", QuoteCCY: "JPY", Description: "澳元/日元"},
		{Canonical: "NZDJPY", AssetClass: "forex", BaseCCY: "NZD", QuoteCCY: "JPY", Description: "纽元/日元"},
		{Canonical: "CADJPY", AssetClass: "forex", BaseCCY: "CAD", QuoteCCY: "JPY", Description: "加元/日元"},
		{Canonical: "CHFJPY", AssetClass: "forex", BaseCCY: "CHF", QuoteCCY: "JPY", Description: "瑞郎/日元"},
		{Canonical: "EURNZD", AssetClass: "forex", BaseCCY: "EUR", QuoteCCY: "NZD", Description: "欧元/纽元"},
		{Canonical: "AUDNZD", AssetClass: "forex", BaseCCY: "AUD", QuoteCCY: "NZD", Description: "澳元/纽元"},

		// ── Commodities ──
		{Canonical: "XAUUSD", AssetClass: "commodity", BaseCCY: "XAU", QuoteCCY: "USD", Description: "黄金/美元"},
		{Canonical: "XAGUSD", AssetClass: "commodity", BaseCCY: "XAG", QuoteCCY: "USD", Description: "白银/美元"},
		{Canonical: "XTIUSD", AssetClass: "commodity", BaseCCY: "XTI", QuoteCCY: "USD", Description: "WTI原油"},
		{Canonical: "XBRUSD", AssetClass: "commodity", BaseCCY: "XBR", QuoteCCY: "USD", Description: "布伦特原油"},
		{Canonical: "XNGUSD", AssetClass: "commodity", BaseCCY: "XNG", QuoteCCY: "USD", Description: "天然气"},

		// ── Indices ──
		{Canonical: "US30", AssetClass: "index", BaseCCY: "US30", QuoteCCY: "USD", Description: "道琼斯工业指数"},
		{Canonical: "US100", AssetClass: "index", BaseCCY: "US100", QuoteCCY: "USD", Description: "纳斯达克100"},
		{Canonical: "US500", AssetClass: "index", BaseCCY: "US500", QuoteCCY: "USD", Description: "标普500"},
		{Canonical: "GER40", AssetClass: "index", BaseCCY: "GER40", QuoteCCY: "EUR", Description: "德国DAX40"},
		{Canonical: "UK100", AssetClass: "index", BaseCCY: "UK100", QuoteCCY: "GBP", Description: "英国富时100"},
		{Canonical: "JPN225", AssetClass: "index", BaseCCY: "JPN225", QuoteCCY: "JPY", Description: "日经225"},
		{Canonical: "HK50", AssetClass: "index", BaseCCY: "HK50", QuoteCCY: "HKD", Description: "恒生指数"},
		{Canonical: "AUS200", AssetClass: "index", BaseCCY: "AUS200", QuoteCCY: "AUD", Description: "澳洲ASX200"},
		{Canonical: "FRA40", AssetClass: "index", BaseCCY: "FRA40", QuoteCCY: "EUR", Description: "法国CAC40"},
		{Canonical: "EU50", AssetClass: "index", BaseCCY: "EU50", QuoteCCY: "EUR", Description: "欧洲斯托克50"},

		// ── Crypto ──
		{Canonical: "BTCUSD", AssetClass: "crypto", BaseCCY: "BTC", QuoteCCY: "USD", Description: "比特币/美元"},
		{Canonical: "ETHUSD", AssetClass: "crypto", BaseCCY: "ETH", QuoteCCY: "USD", Description: "以太坊/美元"},
		{Canonical: "XRPUSD", AssetClass: "crypto", BaseCCY: "XRP", QuoteCCY: "USD", Description: "瑞波币/美元"},
		{Canonical: "LTCUSD", AssetClass: "crypto", BaseCCY: "LTC", QuoteCCY: "USD", Description: "莱特币/美元"},
		{Canonical: "BCHUSD", AssetClass: "crypto", BaseCCY: "BCH", QuoteCCY: "USD", Description: "比特币现金/美元"},
		{Canonical: "SOLUSD", AssetClass: "crypto", BaseCCY: "SOL", QuoteCCY: "USD", Description: "Solana/美元"},
		{Canonical: "DOGEUSD", AssetClass: "crypto", BaseCCY: "DOGE", QuoteCCY: "USD", Description: "狗狗币/美元"},
		{Canonical: "ADAUSD", AssetClass: "crypto", BaseCCY: "ADA", QuoteCCY: "USD", Description: "Cardano/美元"},
		{Canonical: "DOTUSD", AssetClass: "crypto", BaseCCY: "DOT", QuoteCCY: "USD", Description: "Polkadot/美元"},
		{Canonical: "AVAXUSD", AssetClass: "crypto", BaseCCY: "AVAX", QuoteCCY: "USD", Description: "Avalanche/美元"},
	}
}
