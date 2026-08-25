package interp

import (
	"strings"
)

// IRReport is the static analysis report derived from an IR.
// It is the single source for coverage/blind-spot reporting.
type IRReport struct {
	Version        string
	Coverage       float64
	TotalCalls     int
	SupportedCalls int
	BlindSpots     []IRBlindSpot
	Params         []ParamDecl
	Indicators     []string
	ExecKind       string // "on_bar" | "on_tick" | "on_timer"
	EntryRules     int    // count of entry-order calls (OrderSend, CTrade.Buy/Sell/BuyLimit/etc)
	ExitRules      int    // count of exit-order calls (OrderClose, CTrade.PositionClose, OrderDelete)
}

// IRBlindSpot represents an unimplemented or stub builtin call found in the IR.
type IRBlindSpot struct {
	Builtin  string
	Severity string // one of SeverityFatal / SeverityWarning / SeverityInfo
	Count    int
}

// Severity constants — shared between analyze.go and strategy_import_handler.go.
// The frontend filters on these exact strings, so they must stay in sync.
const (
	SeverityFatal   = "致命"
	SeverityWarning = "警告"
	SeverityInfo    = "信息"
)

// EnglishToChineseSeverity maps English severity strings (used by DiagnosticFinding)
// to Chinese severity constants (used by BlindSpot). Unknown values default to warning.
func EnglishToChineseSeverity(en string) string {
	switch en {
	case "fatal":
		return SeverityFatal
	case "warning":
		return SeverityWarning
	case "info":
		return SeverityInfo
	default:
		return SeverityWarning
	}
}

// Analyze performs a static analysis pass over the IR and returns a report.
func Analyze(ir *IR) *IRReport {
	rep := &IRReport{Version: ir.Version, Params: ir.Params}
	globalTypes := buildGlobalTypes(ir)

	calls := collectAllCalls(ir, globalTypes)
	rep.TotalCalls = len(calls)

	blindMap := map[string]*IRBlindSpot{}
	indSet := map[string]bool{}

	for _, c := range calls {
		if c.isUserFunc || c.isUserMethod {
			rep.SupportedCalls++
			continue
		}
		if c.isImplemented {
			rep.SupportedCalls++
			// Registry-driven: check if this is an indicator name.
			if isRegistryIndicator(c.name) {
				indSet[c.name] = true
			}
			continue
		}
		// Not implemented → blind spot
		bs := blindMap[c.name]
		if bs == nil {
			bs = &IRBlindSpot{Builtin: c.name, Severity: classifySeverity(c)}
			blindMap[c.name] = bs
		}
		bs.Count++
	}

	rep.BlindSpots = finalizeBlindSpots(blindMap)
	rep.Indicators = sortedKeys(indSet)
	rep.ExecKind = determineExecKind(ir)
	rep.EntryRules, rep.ExitRules = countEntryExitRules(ir)
	if rep.TotalCalls > 0 {
		rep.Coverage = float64(rep.SupportedCalls) / float64(rep.TotalCalls)
	}
	return rep
}

// IsBuiltinImplemented checks if a free-function builtin is implemented.
// Registry-driven: looks up the name in the API registry (Layer 0).
func IsBuiltinImplemented(name string) bool {
	s, ok := LookupAPI(name)
	return ok && s.Status == StatusImplemented && s.Category == CatFunction
}

// IsCTradeMethodImplemented checks if a CTrade method is implemented.
// Registry-driven: looks up the name with CatCTradeMethod category.
func IsCTradeMethodImplemented(method string) bool {
	s, ok := LookupAPI(method)
	return ok && s.Status == StatusImplemented && s.Category == CatCTradeMethod
}

// ── callInfo ────────────────────────────────────────────────────────

type callInfo struct {
	name          string
	isUserFunc    bool
	isUserMethod  bool // user-defined class method (non-CTrade)
	isImplemented bool
	classType     string // for ExprField method calls: class name (e.g. "CTrade")
}

// ── traversal helpers ───────────────────────────────────────────────

func buildGlobalTypes(ir *IR) map[string]string {
	m := map[string]string{}
	for _, g := range ir.Globals {
		m[g.Name] = g.Type
	}
	return m
}

func collectAllCalls(ir *IR, globalTypes map[string]string) []callInfo {
	var calls []callInfo
	visit := func(e *Expr) {
		switch e.Kind {
		case ExprCall:
			calls = append(calls, classifyCall(ir, e.Name))
		case ExprField:
			if !e.IsAssign && len(e.Args) > 1 {
				clsType := resolveClassType(&e.Args[0], globalTypes)
				if clsType != "" {
					calls = append(calls, classifyMethodCall(clsType, e.Name))
				}
			}
		}
	}
	walkIR(ir, visit)
	return calls
}

func classifyCall(ir *IR, name string) callInfo {
	ci := callInfo{name: name}
	if ir.Funcs != nil {
		if _, ok := ir.Funcs[name]; ok {
			ci.isUserFunc = true
			return ci
		}
	}
	// Python compiler produces ExprCall{Name: "CTrade.Buy"} for ctx.broker.buy().
	// MQL path uses ExprField with classType="CTrade" → classifyMethodCall.
	// Handle the dotted name here so Python IR gets correct coverage.
	if strings.HasPrefix(name, "CTrade.") {
		method := name[len("CTrade."):]
		ci.classType = "CTrade"
		ci.isImplemented = IsCTradeMethodImplemented(method)
		return ci
	}
	ci.isImplemented = IsBuiltinImplemented(name)
	return ci
}

func classifyMethodCall(classType, method string) callInfo {
	ci := callInfo{name: method, classType: classType}
	if classType == "CTrade" {
		ci.isImplemented = IsCTradeMethodImplemented(method)
	} else {
		// User-defined class method (e.g. LadinoBot.onTick, _logs.adicionarLog)
		// — not a missing builtin, just user code we can't see into
		ci.isUserMethod = true
	}
	return ci
}

func resolveClassType(e *Expr, globalTypes map[string]string) string {
	if e == nil {
		return ""
	}
	if e.Kind == ExprVar {
		if t, ok := globalTypes[e.Name]; ok {
			return t
		}
	}
	return ""
}

// ── severity ────────────────────────────────────────────────────────

// SeverityForBuiltin returns the severity string for a given builtin name.
// Exported so VM and backtest worker can use the same severity classification.
func SeverityForBuiltin(name string) string {
	return classifySeverity(callInfo{name: name})
}

func classifySeverity(c callInfo) string {
	name := c.name
	// CTrade method → fatal
	if c.classType == "CTrade" {
		return SeverityFatal
	}
	// User-defined class method call (e.g. _ladinoBot.onTick(), _logs.adicionarLog())
	// → info, not warning: these are user code, not missing builtins
	if c.classType != "" {
		return SeverityInfo
	}
	// Unknown but looks like an indicator (iXxx pattern) → fatal
	// This catches iCustom (StatusUnsupported) and any unknown indicator.
	if len(name) > 1 && name[0] == 'i' && name[1] >= 'A' && name[1] <= 'Z' {
		return SeverityFatal
	}
	// Unknown but looks like a trade function (Order*/Position*/Account*) → fatal
	if strings.HasPrefix(name, "Order") || strings.HasPrefix(name, "Position") || strings.HasPrefix(name, "Account") {
		return SeverityFatal
	}
	// StatusUnsupported functions that don't match fatal patterns above are
	// graceful no-ops (GUI/Chart/Window/Notification/FileIO/Network/Crypto).
	// They are rejected at compile time (compile_expr.go returns an error),
	// so they never reach the VM. If they did, returning 0 wouldn't corrupt
	// trading logic — they're advisory/UI/side-effect functions.
	// → SeverityInfo, not Warning.
	if sym, ok := LookupAPI(name); ok && sym.Status == StatusUnsupported {
		return SeverityInfo
	}
	// Doesn't match any known MQL builtin pattern → likely user code
	// (e.g. functions defined in #include .mqh files, free functions)
	if !looksLikeMQLBuiltin(name) {
		return SeverityInfo
	}
	return SeverityWarning
}

// isRegistryIndicator checks if a name is an indicator function in the registry.
func isRegistryIndicator(name string) bool {
	s, ok := LookupAPI(name)
	return ok && s.Status == StatusImplemented && s.Category == CatFunction &&
		len(name) > 1 && name[0] == 'i' && name[1] >= 'A' && name[1] <= 'Z'
}

// looksLikeMQLBuiltin checks if a function name matches known MQL builtin
// naming patterns. MQL builtins use PascalCase with recognizable prefixes.
// If a name doesn't match any pattern, it's likely user-defined code
// (e.g. from #include .mqh files).
func looksLikeMQLBuiltin(name string) bool {
	prefixes := []string{
		"Order", "Position", "Account", "History", "Deal",
		"Symbol", "Market", "Chart", "Object", "Terminal",
		"Event", "Expert", "MQL", "File", "Resource",
		"Array", "String", "Math", "Double", "Integer",
		"Normalize", "Period", "Series", "Bars", "Copy",
		"Time", "Date", "Struct", "Enum", "Char", "Short",
		"Color", "Bool",
	}
	for _, p := range prefixes {
		if strings.HasPrefix(name, p) {
			return true
		}
	}
	// Single-word builtins
	switch name {
	case "Print", "Alert", "Comment", "Sleep", "RefreshRates",
		"IsTesting", "IsOptimization", "IsVisualMode", "IsConnected",
		"IsDemo", "IsDllsAllowed", "IsExpertEnabled", "IsLibrariesAllowed",
		"IsTradeAllowed", "IsTradeContextBusy", "IsStopped",
		"UninitializeReason", "GetLastError", "ResetLastError",
		"GetTickCount", "GetTickCount64", "GetMicrosecondCount",
		"SetUserError", "SetReturnError", "PlaySound", "MessageBox",
		"SendMail", "SendNotification", "CurTime", "Day", "DayOfWeek",
		"Hour", "Minute", "Year", "Month", "Seconds", "DayOfYear",
		"TimeCurrent", "TimeGMT", "TimeLocal", "Point", "Bid", "Ask",
		"Digits", "Period", "Bars", "Volume", "Close", "Open", "High", "Low":
		return true
	}
	// iXxx indicator pattern
	if len(name) > 1 && name[0] == 'i' && name[1] >= 'A' && name[1] <= 'Z' {
		return true
	}
	// Is* checkup pattern
	if strings.HasPrefix(name, "Is") && len(name) > 2 && name[2] >= 'A' && name[2] <= 'Z' {
		return true
	}
	// Set*/Get* accessor pattern
	if (strings.HasPrefix(name, "Set") || strings.HasPrefix(name, "Get")) && len(name) > 3 && name[3] >= 'A' && name[3] <= 'Z' {
		return true
	}
	// MQL4 lowercase aliases (ceil, floor, cos, sin, etc.)
	if len(name) > 0 && name[0] >= 'a' && name[0] <= 'z' && len(name) <= 8 {
		return true
	}
	return false
}

// ── IR traversal ────────────────────────────────────────────────────
