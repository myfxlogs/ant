package interp

// CompatFixes is the L0 deterministic fix registry.
//
// Maps problematic identifiers (aliases, naming variants, deprecated names)
// to their canonical MQL constant name. When LookupMQLConstant fails to find
// a name in MQLConstants, it checks CompatFixes as a fallback. If a mapping
// exists, the canonical name is resolved in MQLConstants — transparently,
// with zero tokens, zero blind spots, and zero LLM calls.
//
// This systematizes the HONESTY-1/2 alias patterns (clr* prefixed colors,
// MODE_SENKOU_A/B naming variants) into an extensible registry. New aliases
// can be added here without modifying the monolithic MQLConstants map.
//
// Categories:
//   - Color aliases: clrGreen → Green (MQL standard clr* prefix)
//   - Naming normalization: MODE_SENKOU_A → MODE_SENKOUA (MQL5 vs MQL4 naming)
//   - Deprecated → current: (future entries)
var CompatFixes = map[string]string{
	// ── clr* prefixed color aliases → unprefixed WebColor names ──────────
	"clrBlack":       "Black",
	"clrWhite":       "White",
	"clrRed":         "Red",
	"clrGreen":       "Green",
	"clrBlue":        "Blue",
	"clrYellow":      "Yellow",
	"clrAqua":        "Aqua",
	"clrOrange":      "Orange",
	"clrGold":        "Gold",
	"clrGray":        "Gray",
	"clrSilver":      "Silver",
	"clrLime":        "Lime",
	"clrOlive":       "Olive",
	"clrPurple":      "Purple",
	"clrTeal":        "Teal",
	"clrNavy":        "Navy",
	"clrMaroon":      "Maroon",
	"clrBrown":       "Brown",
	"clrKhaki":       "Khaki",
	"clrTomato":      "Tomato",
	"clrCoral":       "Coral",
	"clrSalmon":      "Salmon",
	"clrSienna":      "Sienna",
	"clrChocolate":   "Chocolate",
	"clrDarkRed":     "DarkRed",
	"clrDarkGreen":   "DarkGreen",
	"clrDarkBlue":    "Navy",
	"clrDarkGray":    "DarkGray",
	"clrLightGray":   "LightGray",
	"clrLightBlue":   "LightBlue",
	"clrLightGreen":  "LightGreen",
	"clrPaleGreen":   "PaleGreen",
	"clrSkyBlue":     "SkyBlue",
	"clrPink":        "Pink",
	"clrMagenta":     "Magenta",
	"clrCyan":        "Aqua",
	"clrDarkOrange":  "DarkOrange",
	"clrOrangeRed":   "OrangeRed",
	"clrYellowGreen": "YellowGreen",
	"clrSeaGreen":    "SeaGreen",
	"clrForestGreen": "ForestGreen",
	"clrDarkCyan":    "DarkCyan",
	"clrTransparent": "Transparent",
	"clrNONE":        "CLR_NONE",

	// ── Naming normalization: MQL5 underscored → MQL4 canonical ──────────
	"MODE_SENKOU_A": "MODE_SENKOUA",
	"MODE_SENKOU_B": "MODE_SENKOUB",
}

// LookupCompatFix checks if a name has a deterministic compat-fix mapping.
// Returns the canonical name and true if found, or "" and false if not.
func LookupCompatFix(name string) (string, bool) {
	// KB-first: check KB fix cache.
	if kbFixLookup != nil {
		if canonical, ok := kbFixLookup(name); ok {
			return canonical, true
		}
	}
	// Fallback: built-in CompatFixes map.
	canonical, ok := CompatFixes[name]
	return canonical, ok
}
