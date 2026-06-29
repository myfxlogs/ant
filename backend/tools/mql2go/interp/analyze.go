package interp

import "sort"

// IRReport is the static analysis report derived from an IR.
// It is the single source for coverage/blind-spot reporting —
// the same IR that the interpreter executes.
type IRReport struct {
	Version        string
	Coverage       float64
	TotalCalls     int
	SupportedCalls int
	BlindSpots     []IRBlindSpot
	Params         []ParamDecl
	Indicators     []string
	ExecKind       string // "on_bar" | "on_tick" | "on_timer"
	EntryRules     int // count of entry-order calls (OrderSend, CTrade.Buy/Sell/BuyLimit/etc)
	ExitRules      int // count of exit-order calls (OrderClose, CTrade.PositionClose, OrderDelete)
}

// IRBlindSpot represents an unimplemented or stub builtin call found in the IR.
type IRBlindSpot struct {
	Builtin  string
	Severity string // "致命" | "警告" | "信息"
	Count    int
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
		if c.isUserFunc {
			rep.SupportedCalls++
			continue
		}
		if c.isImplemented {
			rep.SupportedCalls++
			if isIndicatorName(c.name) {
				indSet[c.name] = true
			}
			continue
		}
		// Not implemented → blind spot
		bs := blindMap[c.name]
		if bs == nil {
			bs = &IRBlindSpot{Builtin: c.name, Severity: classifySeverity(ir.Version, c)}
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
// Single source of truth: references builtinTable + implemented* slices.
func IsBuiltinImplemented(name string) bool {
	if _, ok := builtinTable[name]; ok {
		return true
	}
	if stubIndicators[name] {
		return true
	}
	return inSlice(name, implementedMarketData) ||
		inSlice(name, implementedIndicators) ||
		inSlice(name, implementedMQL4Trade) ||
		inSlice(name, implementedMQL5Position) ||
		inSlice(name, implementedAccount) ||
		inSlice(name, implementedPlatform)
}

// IsCTradeMethodImplemented checks if a CTrade method is implemented.
func IsCTradeMethodImplemented(method string) bool {
	return inSlice(method, implementedCTradeMethods)
}

// IsStubIndicator checks if an indicator is a stub (dispatch but SDK returns 0).
func IsStubIndicator(name string) bool {
	return stubIndicators[name]
}

// ── callInfo ────────────────────────────────────────────────────────

type callInfo struct {
	name          string
	isUserFunc    bool
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
			calls = append(calls, classifyCall(ir, e.Name, globalTypes))
		case ExprField:
			if !e.IsAssign && len(e.Args) > 1 {
				clsType := resolveClassType(&e.Args[0], globalTypes)
				if clsType != "" {
					calls = append(calls, classifyMethodCall(ir, clsType, e.Name))
				}
			}
		}
	}
	walkIR(ir, visit)
	return calls
}

func classifyCall(ir *IR, name string, _ map[string]string) callInfo {
	ci := callInfo{name: name}
	if ir.Funcs != nil {
		if _, ok := ir.Funcs[name]; ok {
			ci.isUserFunc = true
			return ci
		}
	}
	ci.isImplemented = IsBuiltinImplemented(name)
	return ci
}

func classifyMethodCall(ir *IR, classType, method string) callInfo {
	ci := callInfo{name: method, classType: classType}
	if classType == "CTrade" {
		ci.isImplemented = IsCTradeMethodImplemented(method)
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

func classifySeverity(version string, c callInfo) string {
	name := c.name
	// Stub indicator → warning
	if IsStubIndicator(name) {
		return "警告"
	}
	// Permanent blind spots — will never be implemented
	if isPermanentBlindSpot(name) {
		return "永久盲区"
	}
	// CTrade method → fatal
	if c.classType == "CTrade" {
		return "致命"
	}
	// Known trade / indicator → fatal
	if isTradeName(name) || isIndicatorName(name) {
		return "致命"
	}
	// Unknown but looks like an indicator (iXxx pattern) → fatal
	if len(name) > 1 && name[0] == 'i' && name[1] >= 'A' && name[1] <= 'Z' {
		return "致命"
	}
	// Unknown but looks like a trade function (Order*/Position*/Account*) → fatal
	if startsWith(name, "Order") || startsWith(name, "Position") || startsWith(name, "Account") {
		return "致命"
	}
	return "警告"
}

// permanentBlindSpots lists functions that will never be implemented in the
// server-side interpreter, by design. These are MT client-side features or
// security risks that have no server-side equivalent.
var permanentBlindSpots = map[string]string{
	// File I/O — system has its own logging and parameter management
	"FileOpen":          "File I/O: system has own logging/params",
	"FileWrite":         "File I/O: system has own logging/params",
	"FileRead":          "File I/O: system has own logging/params",
	"FileReadArray":     "File I/O: system has own logging/params",
	"FileWriteArray":    "File I/O: system has own logging/params",
	"FileClose":         "File I/O: system has own logging/params",
	"FileDelete":        "File I/O: system has own logging/params",
	"FileIsEnding":      "File I/O: system has own logging/params",
	"FileIsLineEnding":  "File I/O: system has own logging/params",
	"FileSeek":          "File I/O: system has own logging/params",
	"FileTell":          "File I/O: system has own logging/params",
	"FileFlush":         "File I/O: system has own logging/params",
	"FileSize":          "File I/O: system has own logging/params",
	"FileCopy":          "File I/O: system has own logging/params",
	"FileMove":          "File I/O: system has own logging/params",
	"FileFindFirst":     "File I/O: system has own logging/params",
	"FileFindNext":      "File I/O: system has own logging/params",
	"FileFindClose":     "File I/O: system has own logging/params",
	"CSVReader":         "File I/O: system has own logging/params",
	"CSVWriter":         "File I/O: system has own logging/params",
	// Object/Chart — MT client-side UI, no server-side meaning
	"ObjectCreate":      "Object/Chart: MT client UI, no server meaning",
	"ObjectDelete":      "Object/Chart: MT client UI, no server meaning",
	"ObjectSet":         "Object/Chart: MT client UI, no server meaning",
	"ObjectSetInteger":  "Object/Chart: MT client UI, no server meaning",
	"ObjectSetDouble":   "Object/Chart: MT client UI, no server meaning",
	"ObjectSetString":   "Object/Chart: MT client UI, no server meaning",
	"ObjectGet":         "Object/Chart: MT client UI, no server meaning",
	"ObjectGetInteger":  "Object/Chart: MT client UI, no server meaning",
	"ObjectGetDouble":   "Object/Chart: MT client UI, no server meaning",
	"ObjectGetString":   "Object/Chart: MT client UI, no server meaning",
	"ObjectFind":        "Object/Chart: MT client UI, no server meaning",
	"ObjectName":        "Object/Chart: MT client UI, no server meaning",
	"ObjectsTotal":      "Object/Chart: MT client UI, no server meaning",
	"ObjectDescription": "Object/Chart: MT client UI, no server meaning",
	"ObjectMove":        "Object/Chart: MT client UI, no server meaning",
	"ObjectText":        "Object/Chart: MT client UI, no server meaning",
	"ObjectSetText":     "Object/Chart: MT client UI, no server meaning",
	"ObjectsDeleteAll":  "Object/Chart: MT client UI, no server meaning",
	"ChartCreate":       "Object/Chart: MT client UI, no server meaning",
	"ChartDelete":       "Object/Chart: MT client UI, no server meaning",
	"ChartSetInteger":   "Object/Chart: MT client UI, no server meaning",
	"ChartSetDouble":    "Object/Chart: MT client UI, no server meaning",
	"ChartSetString":    "Object/Chart: MT client UI, no server meaning",
	"ChartGetInteger":   "Object/Chart: MT client UI, no server meaning",
	"ChartGetDouble":    "Object/Chart: MT client UI, no server meaning",
	"ChartGetString":    "Object/Chart: MT client UI, no server meaning",
	"ChartNavigate":     "Object/Chart: MT client UI, no server meaning",
	"ChartRedraw":       "Object/Chart: MT client UI, no server meaning",
	"ChartApplyTemplate":  "Object/Chart: MT client UI, no server meaning",
	"ChartSaveTemplate":   "Object/Chart: MT client UI, no server meaning",
	// DLL import — security risk, not supported
	"DLLImport":         "DLL #import: security risk, not supported",
	// MQL5 native OrderSend — CTrade wrapper covers this
	"OrderSendMQL5":     "MQL5 native OrderSend: CTrade wrapper covers this",
}

// isPermanentBlindSpot returns true if the function is a known permanent blind spot.
func isPermanentBlindSpot(name string) bool {
	_, ok := permanentBlindSpots[name]
	return ok
}

func isTradeName(name string) bool {
	return inSlice(name, implementedMQL4Trade) ||
		inSlice(name, implementedMQL5Position) ||
		inSlice(name, implementedAccount)
}

func isIndicatorName(name string) bool {
	return inSlice(name, implementedIndicators)
}

// ── IR traversal ────────────────────────────────────────────────────

func walkIR(ir *IR, visit func(*Expr)) {
	walkStmts(ir.OnInit, visit)
	walkStmts(ir.OnBar, visit)
	walkStmts(ir.OnTick, visit)
	walkStmts(ir.OnTimer, visit)
	walkStmts(ir.OnTrade, visit)
	walkStmts(ir.OnTradeTransaction, visit)
	walkStmts(ir.OnDeinit, visit)
	for _, fn := range ir.Funcs {
		walkStmts(fn.Body, visit)
	}
}

func walkStmts(stmts []Statement, visit func(*Expr)) {
	for i := range stmts {
		walkStmt(&stmts[i], visit)
	}
}

func walkStmt(s *Statement, visit func(*Expr)) {
	if s.Expr != nil {
		walkExpr(s.Expr, visit)
	}
	if s.Cond != nil {
		walkExpr(s.Cond, visit)
	}
	if s.Init != nil {
		walkStmt(s.Init, visit)
	}
	if s.Update != nil {
		walkStmt(s.Update, visit)
	}
	walkStmts(s.Body, visit)
	walkStmts(s.ElseBody, visit)
	for i := range s.Cases {
		if s.Cases[i].Expr != nil {
			walkExpr(s.Cases[i].Expr, visit)
		}
		walkStmts(s.Cases[i].Body, visit)
	}
}

func walkExpr(e *Expr, visit func(*Expr)) {
	visit(e)
	for i := range e.Args {
		walkExpr(&e.Args[i], visit)
	}
	if e.Index != nil {
		walkExpr(e.Index, visit)
	}
	if e.Cond != nil {
		walkExpr(e.Cond, visit)
	}
	if e.ThenExpr != nil {
		walkExpr(e.ThenExpr, visit)
	}
	if e.ElseExpr != nil {
		walkExpr(e.ElseExpr, visit)
	}
}

// ── utilities ───────────────────────────────────────────────────────

func determineExecKind(ir *IR) string {
	if len(ir.OnTick) > 0 {
		return "on_tick"
	}
	if len(ir.OnBar) > 0 {
		return "on_bar"
	}
	if len(ir.OnTimer) > 0 {
		return "on_timer"
	}
	return ""
}

func finalizeBlindSpots(m map[string]*IRBlindSpot) []IRBlindSpot {
	result := make([]IRBlindSpot, 0, len(m))
	for _, bs := range m {
		result = append(result, *bs)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Severity != result[j].Severity {
			return result[i].Severity < result[j].Severity
		}
		return result[i].Builtin < result[j].Builtin
	})
	return result
}

func sortedKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func inSlice(s string, list []string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func startsWith(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

// countEntryExitRules counts entry-order and exit-order calls in the IR.
// Entry: OrderSend, CTrade.Buy/Sell/BuyLimit/SellLimit/BuyStop/SellStop
// Exit:  OrderClose, OrderCloseBy, OrderDelete, CTrade.PositionClose/PositionClosePartial/PositionCloseBy
func countEntryExitRules(ir *IR) (entry, exit int) {
	entryNames := map[string]bool{
		"OrderSend": true,
	}
	exitNames := map[string]bool{
		"OrderClose": true, "OrderCloseBy": true, "OrderDelete": true,
	}
	ctradeEntry := map[string]bool{
		"Buy": true, "Sell": true, "BuyLimit": true, "SellLimit": true,
		"BuyStop": true, "SellStop": true,
	}
	ctradeExit := map[string]bool{
		"PositionClose": true, "PositionClosePartial": true, "PositionCloseBy": true,
		"OrderDelete": true,
	}
	globalTypes := buildGlobalTypes(ir)
	visit := func(e *Expr) {
		switch e.Kind {
		case ExprCall:
			if entryNames[e.Name] {
				entry++
			}
			if exitNames[e.Name] {
				exit++
			}
		case ExprField:
			if !e.IsAssign && len(e.Args) > 1 {
				clsType := resolveClassType(&e.Args[0], globalTypes)
				if clsType == "CTrade" {
					if ctradeEntry[e.Name] {
						entry++
					}
					if ctradeExit[e.Name] {
						exit++
					}
				}
			}
		}
	}
	walkIR(ir, visit)
	return
}

// EvalExprLiteral evaluates a simple literal/var Expr to its string form.
// Used for extracting parameter default values without a full interpreter.
func EvalExprLiteral(e *Expr) string {
	if e == nil {
		return ""
	}
	switch e.Kind {
	case ExprLiteral:
		return e.Val.ToString()
	case ExprVar, ExprConst:
		return e.Name
	case ExprUnary:
		if e.Op == "-" {
			return "-" + EvalExprLiteral(&e.Args[0])
		}
		return EvalExprLiteral(&e.Args[0])
	}
	return ""
}
