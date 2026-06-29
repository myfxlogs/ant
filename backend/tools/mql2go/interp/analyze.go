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
		return SeverityWarning
	}
	// Permanent blind spots — will never be implemented
	if isPermanentBlindSpot(name) {
		return SeverityInfo
	}
	// CTrade method → fatal
	if c.classType == "CTrade" {
		return SeverityFatal
	}
	// Known trade / indicator → fatal
	if isTradeName(name) || isIndicatorName(name) {
		return SeverityFatal
	}
	// Unknown but looks like an indicator (iXxx pattern) → fatal
	if len(name) > 1 && name[0] == 'i' && name[1] >= 'A' && name[1] <= 'Z' {
		return SeverityFatal
	}
	// Unknown but looks like a trade function (Order*/Position*/Account*) → fatal
	if startsWith(name, "Order") || startsWith(name, "Position") || startsWith(name, "Account") {
		return SeverityFatal
	}
	return SeverityWarning
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
	// Remaining Object/Chart functions from MQL5 docs
	"ObjectGetTimeByValue":    "Object/Chart: MT client UI, no server meaning",
	"ObjectGetValueByTime":    "Object/Chart: MT client UI, no server meaning",
	"ChartClose":              "Object/Chart: MT client UI, no server meaning",
	"ChartFirst":              "Object/Chart: MT client UI, no server meaning",
	"ChartID":                 "Object/Chart: MT client UI, no server meaning",
	"ChartIndicatorAdd":       "Object/Chart: MT client UI, no server meaning",
	"ChartIndicatorDelete":    "Object/Chart: MT client UI, no server meaning",
	"ChartIndicatorGet":       "Object/Chart: MT client UI, no server meaning",
	"ChartIndicatorName":      "Object/Chart: MT client UI, no server meaning",
	"ChartIndicatorsTotal":    "Object/Chart: MT client UI, no server meaning",
	"ChartNext":               "Object/Chart: MT client UI, no server meaning",
	"ChartOpen":               "Object/Chart: MT client UI, no server meaning",
	"ChartPeriod":             "Object/Chart: MT client UI, no server meaning",
	"ChartPriceOnDropped":     "Object/Chart: MT client UI, no server meaning",
	"ChartScreenShot":         "Object/Chart: MT client UI, no server meaning",
	"ChartSetSymbolPeriod":    "Object/Chart: MT client UI, no server meaning",
	"ChartSymbol":             "Object/Chart: MT client UI, no server meaning",
	"ChartTimeOnDropped":      "Object/Chart: MT client UI, no server meaning",
	"ChartTimePriceToXY":      "Object/Chart: MT client UI, no server meaning",
	"ChartWindowFind":         "Object/Chart: MT client UI, no server meaning",
	"ChartWindowOnDropped":    "Object/Chart: MT client UI, no server meaning",
	"ChartXOnDropped":         "Object/Chart: MT client UI, no server meaning",
	"ChartXYToTimePrice":      "Object/Chart: MT client UI, no server meaning",
	"ChartYOnDropped":         "Object/Chart: MT client UI, no server meaning",
	"TextGetSize":             "Object/Chart: MT client UI, no server meaning",
	"TextOut":                 "Object/Chart: MT client UI, no server meaning",
	"TextSetFont":             "Object/Chart: MT client UI, no server meaning",
	// Remaining File functions from MQL5 docs
	"FileReadBool":            "File I/O: system has own logging/params",
	"FileReadDatetime":        "File I/O: system has own logging/params",
	"FileReadDouble":          "File I/O: system has own logging/params",
	"FileReadFloat":           "File I/O: system has own logging/params",
	"FileReadInteger":         "File I/O: system has own logging/params",
	"FileReadLong":            "File I/O: system has own logging/params",
	"FileReadNumber":          "File I/O: system has own logging/params",
	"FileReadString":          "File I/O: system has own logging/params",
	"FileReadStruct":          "File I/O: system has own logging/params",
	"FileWriteDouble":         "File I/O: system has own logging/params",
	"FileWriteFloat":          "File I/O: system has own logging/params",
	"FileWriteInteger":        "File I/O: system has own logging/params",
	"FileWriteLong":           "File I/O: system has own logging/params",
	"FileWriteString":         "File I/O: system has own logging/params",
	"FileWriteStruct":         "File I/O: system has own logging/params",
	"FileGetInteger":          "File I/O: system has own logging/params",
	"FileIsExist":             "File I/O: system has own logging/params",
	"FolderClean":             "File I/O: system has own logging/params",
	"FolderCreate":            "File I/O: system has own logging/params",
	"FolderDelete":            "File I/O: system has own logging/params",
	// Network — no server-side network access from EA
	"WebRequest":              "Network: no outbound HTTP from server-side EA",
	"SocketCreate":            "Network: no socket access from server-side EA",
	"SocketClose":             "Network: no socket access from server-side EA",
	"SocketConnect":           "Network: no socket access from server-side EA",
	"SocketIsConnected":       "Network: no socket access from server-side EA",
	"SocketIsReadable":        "Network: no socket access from server-side EA",
	"SocketIsWritable":        "Network: no socket access from server-side EA",
	"SocketTimeouts":          "Network: no socket access from server-side EA",
	"SocketRead":              "Network: no socket access from server-side EA",
	"SocketSend":              "Network: no socket access from server-side EA",
	"SocketTlsHandshake":      "Network: no socket access from server-side EA",
	"SocketTlsCertificate":    "Network: no socket access from server-side EA",
	"SocketTlsRead":           "Network: no socket access from server-side EA",
	"SocketTlsReadAvailable":  "Network: no socket access from server-side EA",
	"SocketTlsSend":           "Network: no socket access from server-side EA",
	"SendFTP":                 "Network: no FTP from server-side EA",
	"SendMail":                "Network: no email from server-side EA",
	"SendNotification":        "Network: no push notifications from server-side EA",
	// Database — use PostgreSQL instead
	"DatabaseOpen":            "Database: use PostgreSQL via system",
	"DatabaseClose":           "Database: use PostgreSQL via system",
	"DatabaseImport":          "Database: use PostgreSQL via system",
	"DatabaseExport":          "Database: use PostgreSQL via system",
	"DatabasePrint":           "Database: use PostgreSQL via system",
	"DatabaseTableExists":     "Database: use PostgreSQL via system",
	"DatabaseExecute":         "Database: use PostgreSQL via system",
	"DatabasePrepare":         "Database: use PostgreSQL via system",
	"DatabaseReset":           "Database: use PostgreSQL via system",
	"DatabaseBind":            "Database: use PostgreSQL via system",
	"DatabaseBindArray":       "Database: use PostgreSQL via system",
	"DatabaseRead":            "Database: use PostgreSQL via system",
	"DatabaseReadBind":        "Database: use PostgreSQL via system",
	"DatabaseFinalize":        "Database: use PostgreSQL via system",
	"DatabaseTransactionBegin":   "Database: use PostgreSQL via system",
	"DatabaseTransactionCommit":  "Database: use PostgreSQL via system",
	"DatabaseTransactionRollback": "Database: use PostgreSQL via system",
	"DatabaseColumnsCount":    "Database: use PostgreSQL via system",
	"DatabaseColumnName":      "Database: use PostgreSQL via system",
	"DatabaseColumnType":      "Database: use PostgreSQL via system",
	"DatabaseColumnSize":      "Database: use PostgreSQL via system",
	"DatabaseColumnText":      "Database: use PostgreSQL via system",
	"DatabaseColumnInteger":   "Database: use PostgreSQL via system",
	"DatabaseColumnLong":      "Database: use PostgreSQL via system",
	"DatabaseColumnDouble":    "Database: use PostgreSQL via system",
	"DatabaseColumnBlob":      "Database: use PostgreSQL via system",
	// OpenCL — GPU compute, not available server-side
	"CLBufferCreate":          "OpenCL: GPU compute not available server-side",
	"CLBufferFree":            "OpenCL: GPU compute not available server-side",
	"CLBufferRead":            "OpenCL: GPU compute not available server-side",
	"CLBufferWrite":           "OpenCL: GPU compute not available server-side",
	"CLContextCreate":         "OpenCL: GPU compute not available server-side",
	"CLContextFree":           "OpenCL: GPU compute not available server-side",
	"CLExecute":               "OpenCL: GPU compute not available server-side",
	"CLGetDeviceInfo":         "OpenCL: GPU compute not available server-side",
	"CLGetInfoInteger":        "OpenCL: GPU compute not available server-side",
	"CLHandleType":            "OpenCL: GPU compute not available server-side",
	"CLKernelCreate":          "OpenCL: GPU compute not available server-side",
	"CLKernelFree":            "OpenCL: GPU compute not available server-side",
	"CLProgramCreate":         "OpenCL: GPU compute not available server-side",
	"CLProgramFree":           "OpenCL: GPU compute not available server-side",
	"CLSetKernelArg":          "OpenCL: GPU compute not available server-side",
	"CLSetKernelArgMem":       "OpenCL: GPU compute not available server-side",
	// DirectX — graphics rendering, not available server-side
	"DXContextCreate":         "DirectX: graphics not available server-side",
	"DXContextSetSize":        "DirectX: graphics not available server-side",
	"DXContextClearColors":    "DirectX: graphics not available server-side",
	"DXContextClearDepth":     "DirectX: graphics not available server-side",
	"DXContextGetColors":      "DirectX: graphics not available server-side",
	"DXContextGetDepth":       "DirectX: graphics not available server-side",
	"DXBufferCreate":          "DirectX: graphics not available server-side",
	"DXTextureCreate":         "DirectX: graphics not available server-side",
	"DXInputCreate":           "DirectX: graphics not available server-side",
	"DXInputSet":              "DirectX: graphics not available server-side",
	"DXShaderCreate":          "DirectX: graphics not available server-side",
	"DXShaderSetLayout":       "DirectX: graphics not available server-side",
	"DXShaderInputsSet":       "DirectX: graphics not available server-side",
	"DXShaderTexturesSet":     "DirectX: graphics not available server-side",
	"DXDraw":                  "DirectX: graphics not available server-side",
	"DXDrawIndexed":           "DirectX: graphics not available server-side",
	"DXPrimiveTopologySet":    "DirectX: graphics not available server-side",
	"DXBufferSet":             "DirectX: graphics not available server-side",
	"DXShaderSet":             "DirectX: graphics not available server-side",
	"DXHandleType":            "DirectX: graphics not available server-side",
	"DXRelease":               "DirectX: graphics not available server-side",
	// Economic Calendar — not available in backtest
	"CalendarCountryById":         "Calendar: economic calendar not available server-side",
	"CalendarEventById":           "Calendar: economic calendar not available server-side",
	"CalendarValueById":           "Calendar: economic calendar not available server-side",
	"CalendarCountries":           "Calendar: economic calendar not available server-side",
	"CalendarEventByCountry":      "Calendar: economic calendar not available server-side",
	"CalendarEventByCurrency":     "Calendar: economic calendar not available server-side",
	"CalendarValueHistoryByEvent": "Calendar: economic calendar not available server-side",
	"CalendarValueHistory":        "Calendar: economic calendar not available server-side",
	"CalendarValueLastByEvent":    "Calendar: economic calendar not available server-side",
	"CalendarValueLast":           "Calendar: economic calendar not available server-side",
	// Custom Symbols — not applicable server-side
	"CustomSymbolCreate":          "Custom Symbols: not applicable server-side",
	"CustomSymbolDelete":          "Custom Symbols: not applicable server-side",
	"CustomSymbolSetInteger":      "Custom Symbols: not applicable server-side",
	"CustomSymbolSetDouble":       "Custom Symbols: not applicable server-side",
	"CustomSymbolSetString":       "Custom Symbols: not applicable server-side",
	"CustomSymbolSetMarginRate":   "Custom Symbols: not applicable server-side",
	"CustomSymbolSetSessionQuote": "Custom Symbols: not applicable server-side",
	"CustomSymbolSetSessionTrade": "Custom Symbols: not applicable server-side",
	"CustomRatesDelete":           "Custom Symbols: not applicable server-side",
	"CustomRatesReplace":          "Custom Symbols: not applicable server-side",
	"CustomRatesUpdate":           "Custom Symbols: not applicable server-side",
	"CustomTicksAdd":              "Custom Symbols: not applicable server-side",
	"CustomTicksDelete":           "Custom Symbols: not applicable server-side",
	"CustomTicksReplace":          "Custom Symbols: not applicable server-side",
	"CustomBookAdd":               "Custom Symbols: not applicable server-side",
	// Global Variables of Terminal — client-side state
	"GlobalVariableCheck":         "Global Variables: terminal client-side state",
	"GlobalVariableDel":           "Global Variables: terminal client-side state",
	"GlobalVariableGet":           "Global Variables: terminal client-side state",
	"GlobalVariableName":          "Global Variables: terminal client-side state",
	"GlobalVariablesDeleteAll":    "Global Variables: terminal client-side state",
	"GlobalVariableSet":           "Global Variables: terminal client-side state",
	"GlobalVariableSetOnCondition": "Global Variables: terminal client-side state",
	"GlobalVariablesFlush":        "Global Variables: terminal client-side state",
	"GlobalVariablesTotal":        "Global Variables: terminal client-side state",
	"GlobalVariableTemp":          "Global Variables: terminal client-side state",
	"GlobalVariableTime":          "Global Variables: terminal client-side state",
	// Optimization Frames — strategy tester only
	"FrameAdd":                    "Optimization: strategy tester only",
	"FrameFilter":                 "Optimization: strategy tester only",
	"FrameFirst":                  "Optimization: strategy tester only",
	"FrameInputs":                 "Optimization: strategy tester only",
	"FrameNext":                   "Optimization: strategy tester only",
	"ParameterGetRange":           "Optimization: strategy tester only",
	"ParameterSetRange":           "Optimization: strategy tester only",
	// Trade Signals — terminal client-side
	"SignalBaseGetDouble":         "Signals: terminal client-side feature",
	"SignalBaseGetInteger":        "Signals: terminal client-side feature",
	"SignalBaseGetString":         "Signals: terminal client-side feature",
	"SignalBaseSelect":            "Signals: terminal client-side feature",
	"SignalBaseTotal":             "Signals: terminal client-side feature",
	"SignalInfoGetDouble":         "Signals: terminal client-side feature",
	"SignalInfoGetInteger":        "Signals: terminal client-side feature",
	"SignalInfoGetString":         "Signals: terminal client-side feature",
	"SignalInfoSetDouble":         "Signals: terminal client-side feature",
	"SignalInfoSetInteger":        "Signals: terminal client-side feature",
	"SignalSubscribe":             "Signals: terminal client-side feature",
	"SignalUnsubscribe":           "Signals: terminal client-side feature",
	// Custom Indicator functions — indicator development only
	"IndicatorCreate":             "Custom Indicators: indicator development only",
	"IndicatorParameters":         "Custom Indicators: indicator development only",
	"IndicatorRelease":            "Custom Indicators: indicator development only",
	"IndicatorSetDouble":          "Custom Indicators: indicator development only",
	"IndicatorSetInteger":         "Custom Indicators: indicator development only",
	"IndicatorSetString":          "Custom Indicators: indicator development only",
	"PlotIndexGetInteger":         "Custom Indicators: indicator development only",
	"PlotIndexSetDouble":          "Custom Indicators: indicator development only",
	"PlotIndexSetInteger":         "Custom Indicators: indicator development only",
	"PlotIndexSetString":          "Custom Indicators: indicator development only",
	"SetIndexBuffer":              "Custom Indicators: indicator development only",
	// Market Book — DOM not available server-side
	"MarketBookAdd":               "Market Book: DOM not available server-side",
	"MarketBookGet":               "Market Book: DOM not available server-side",
	"MarketBookRelease":           "Market Book: DOM not available server-side",
	// Misc client-only / not applicable
	"MessageBox":                  "Client UI: message box not available server-side",
	"PlaySound":                   "Client UI: sound not available server-side",
	"ResourceCreate":              "Client UI: resource not available server-side",
	"ResourceFree":                "Client UI: resource not available server-side",
	"ResourceReadImage":           "Client UI: resource not available server-side",
	"ResourceSave":                "Client UI: resource not available server-side",
	"TerminalClose":               "Client UI: terminal control not available server-side",
	"TesterStatistics":            "Strategy tester only",
	"EventChartCustom":            "Chart event: client-side only",
	"OnChartEvent":                "Chart event: client-side only",
	"OnStart":                     "Script event: not applicable for EA",
	"OnCalculate":                 "Indicator event: not applicable for EA",
	"OnBookEvent":                 "Market depth event: not available server-side",
	"OnTesterInit":                "Strategy tester optimization only",
	"OnTesterDeinit":              "Strategy tester optimization only",
	"OnTesterPass":                "Strategy tester optimization only",
	"CheckPointer":                "Memory management: Go handles this",
	"GetPointer":                  "Memory management: Go handles this",
	"DebugBreak":                  "Debugging: not applicable in production",
	"ZeroMemory":                  "Memory management: Go handles this",
	"PrintFormat":                 "Use Print instead",
	// DLL import — security risk, not supported
	"DLLImport":         "DLL #import: security risk, not supported",
	// MQL5 native OrderSend — CTrade wrapper covers this
	"OrderSendMQL5":     "MQL5 native OrderSend: CTrade wrapper covers this",
	"OrderSendAsync":    "MQL5 async OrderSend: use CTrade wrapper instead",
	// MQL4-only Window functions — client UI, no server meaning
	"WindowBarsPerChart":     "Window: MT client UI, no server meaning",
	"WindowExpertName":       "Window: MT client UI, no server meaning",
	"WindowFind":             "Window: MT client UI, no server meaning",
	"WindowFirstVisibleBar":  "Window: MT client UI, no server meaning",
	"WindowHandle":           "Window: MT client UI, no server meaning",
	"WindowIsVisible":        "Window: MT client UI, no server meaning",
	"WindowOnDropped":        "Window: MT client UI, no server meaning",
	"WindowPriceMax":         "Window: MT client UI, no server meaning",
	"WindowPriceMin":         "Window: MT client UI, no server meaning",
	"WindowPriceOnDropped":   "Window: MT client UI, no server meaning",
	"WindowRedraw":           "Window: MT client UI, no server meaning",
	"WindowScreenShot":       "Window: MT client UI, no server meaning",
	"WindowsTotal":           "Window: MT client UI, no server meaning",
	"WindowTimeOnDropped":    "Window: MT client UI, no server meaning",
	"WindowXOnDropped":       "Window: MT client UI, no server meaning",
	"WindowYOnDropped":       "Window: MT client UI, no server meaning",
	// MQL4-only custom indicator functions — indicator development only
	"IndicatorBuffers":       "Custom Indicators: indicator development only",
	"IndicatorCounted":       "Custom Indicators: indicator development only",
	"IndicatorDigits":        "Custom Indicators: indicator development only",
	"IndicatorShortName":     "Custom Indicators: indicator development only",
	"SetIndexArrow":          "Custom Indicators: indicator development only",
	"SetIndexDrawBegin":      "Custom Indicators: indicator development only",
	"SetIndexEmptyValue":     "Custom Indicators: indicator development only",
	"SetIndexLabel":          "Custom Indicators: indicator development only",
	"SetIndexShift":          "Custom Indicators: indicator development only",
	"SetIndexStyle":          "Custom Indicators: indicator development only",
	"SetLevelStyle":          "Custom Indicators: indicator development only",
	"SetLevelValue":          "Custom Indicators: indicator development only",
	// MQL4-only Object functions
	"ObjectType":             "Object/Chart: MT client UI, no server meaning",
	"ObjectGetFiboDescription":   "Object/Chart: MT client UI, no server meaning",
	"ObjectSetFiboDescription":   "Object/Chart: MT client UI, no server meaning",
	"ObjectGetShiftByValue":  "Object/Chart: MT client UI, no server meaning",
	"ObjectGetValueByShift":  "Object/Chart: MT client UI, no server meaning",
	// MQL4-only Array functions
	"ArrayCopyRates":         "Array: MQL4 timeseries copy, use Copy* functions",
	"ArrayCopySeries":        "Array: MQL4 timeseries copy, use Copy* functions",
	"ArrayDimension":         "Array: MQL4 legacy, use ArrayRange",
	// Tester functions
	"HideTestIndicators":     "Strategy tester only",
	"TesterHideIndicators":   "Strategy tester only",
	"TesterStop":             "Strategy tester only",
	"TesterDeposit":          "Strategy tester only",
	"TesterWithdrawal":       "Strategy tester only",
	// Misc client-only
	"TranslateKey":           "Client UI: keyboard input, not applicable server-side",
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
