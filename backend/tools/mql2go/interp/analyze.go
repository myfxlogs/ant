package interp

import (
	"sort"
	"strings"
)

const (
	bsFileIO       = "File I/O: system has own logging/params"
	bsObjectChart  = "Object/Chart: MT client UI, no server meaning"
	bsNetwork      = "Network: no socket access from server-side EA"
	bsDatabase     = "Database: use PostgreSQL via system"
	bsOpenCL       = "OpenCL: GPU compute not available server-side"
	bsDirectX      = "DirectX: graphics not available server-side"
	bsCalendar     = "Calendar: economic calendar not available server-side"
	bsCustomSymbol = "Custom Symbols: not applicable server-side"
	bsSignals      = "Signals: terminal client-side feature"
	bsCustomInd    = "Custom Indicators: indicator development only"
	bsWindow       = "Window: MT client UI, no server meaning"
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
		if c.isUserFunc || c.isUserMethod {
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
// Single source of truth: implemented* slices in builtin_registry.go.
func IsBuiltinImplemented(name string) bool {
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

func classifyMethodCall(ir *IR, classType, method string) callInfo {
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
	// User-defined class method call (e.g. _ladinoBot.onTick(), _logs.adicionarLog())
	// → info, not warning: these are user code, not missing builtins
	if c.classType != "" {
		return SeverityInfo
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
	// Doesn't match any known MQL builtin pattern → likely user code
	// (e.g. functions defined in #include .mqh files, free functions)
	if !looksLikeMQLBuiltin(name) {
		return SeverityInfo
	}
	return SeverityWarning
}

// permanentBlindSpots lists functions that will never be implemented in the
// VM, by design. These are MT client-side features or security risks
// that have no server-side equivalent.
var permanentBlindSpots = map[string]string{
	// File I/O — system has its own logging and parameter management
	"FileOpen":          bsFileIO,
	"FileWrite":         bsFileIO,
	"FileRead":          bsFileIO,
	"FileReadArray":     bsFileIO,
	"FileWriteArray":    bsFileIO,
	"FileClose":         bsFileIO,
	"FileDelete":        bsFileIO,
	"FileIsEnding":      bsFileIO,
	"FileIsLineEnding":  bsFileIO,
	"FileSeek":          bsFileIO,
	"FileTell":          bsFileIO,
	"FileFlush":         bsFileIO,
	"FileSize":          bsFileIO,
	"FileCopy":          bsFileIO,
	"FileMove":          bsFileIO,
	"FileFindFirst":     bsFileIO,
	"FileFindNext":      bsFileIO,
	"FileFindClose":     bsFileIO,
	"CSVReader":         bsFileIO,
	"CSVWriter":         bsFileIO,
	// Object/Chart — MT client-side UI, no server-side meaning
	"ObjectCreate":      bsObjectChart,
	"ObjectDelete":      bsObjectChart,
	"ObjectSet":         bsObjectChart,
	"ObjectSetInteger":  bsObjectChart,
	"ObjectSetDouble":   bsObjectChart,
	"ObjectSetString":   bsObjectChart,
	"ObjectGet":         bsObjectChart,
	"ObjectGetInteger":  bsObjectChart,
	"ObjectGetDouble":   bsObjectChart,
	"ObjectGetString":   bsObjectChart,
	"ObjectFind":        bsObjectChart,
	"ObjectName":        bsObjectChart,
	"ObjectsTotal":      bsObjectChart,
	"ObjectDescription": bsObjectChart,
	"ObjectMove":        bsObjectChart,
	"ObjectText":        bsObjectChart,
	"ObjectSetText":     bsObjectChart,
	"ObjectsDeleteAll":  bsObjectChart,
	"ChartCreate":       bsObjectChart,
	"ChartDelete":       bsObjectChart,
	"ChartSetInteger":   bsObjectChart,
	"ChartSetDouble":    bsObjectChart,
	"ChartSetString":    bsObjectChart,
	"ChartGetInteger":   bsObjectChart,
	"ChartGetDouble":    bsObjectChart,
	"ChartGetString":    bsObjectChart,
	"ChartNavigate":     bsObjectChart,
	"ChartRedraw":       bsObjectChart,
	"ChartApplyTemplate":  bsObjectChart,
	"ChartSaveTemplate":   bsObjectChart,
	// Remaining Object/Chart functions from MQL5 docs
	"ObjectGetTimeByValue":    bsObjectChart,
	"ObjectGetValueByTime":    bsObjectChart,
	"ChartClose":              bsObjectChart,
	"ChartFirst":              bsObjectChart,
	"ChartID":                 bsObjectChart,
	"ChartIndicatorAdd":       bsObjectChart,
	"ChartIndicatorDelete":    bsObjectChart,
	"ChartIndicatorGet":       bsObjectChart,
	"ChartIndicatorName":      bsObjectChart,
	"ChartIndicatorsTotal":    bsObjectChart,
	"ChartNext":               bsObjectChart,
	"ChartOpen":               bsObjectChart,
	"ChartPeriod":             bsObjectChart,
	"ChartPriceOnDropped":     bsObjectChart,
	"ChartScreenShot":         bsObjectChart,
	"ChartSetSymbolPeriod":    bsObjectChart,
	"ChartSymbol":             bsObjectChart,
	"ChartTimeOnDropped":      bsObjectChart,
	"ChartTimePriceToXY":      bsObjectChart,
	"ChartWindowFind":         bsObjectChart,
	"ChartWindowOnDropped":    bsObjectChart,
	"ChartXOnDropped":         bsObjectChart,
	"ChartXYToTimePrice":      bsObjectChart,
	"ChartYOnDropped":         bsObjectChart,
	"TextGetSize":             bsObjectChart,
	"TextOut":                 bsObjectChart,
	"TextSetFont":             bsObjectChart,
	// Remaining File functions from MQL5 docs
	"FileReadBool":            bsFileIO,
	"FileReadDatetime":        bsFileIO,
	"FileReadDouble":          bsFileIO,
	"FileReadFloat":           bsFileIO,
	"FileReadInteger":         bsFileIO,
	"FileReadLong":            bsFileIO,
	"FileReadNumber":          bsFileIO,
	"FileReadString":          bsFileIO,
	"FileReadStruct":          bsFileIO,
	"FileWriteDouble":         bsFileIO,
	"FileWriteFloat":          bsFileIO,
	"FileWriteInteger":        bsFileIO,
	"FileWriteLong":           bsFileIO,
	"FileWriteString":         bsFileIO,
	"FileWriteStruct":         bsFileIO,
	"FileGetInteger":          bsFileIO,
	"FileIsExist":             bsFileIO,
	"FolderClean":             bsFileIO,
	"FolderCreate":            bsFileIO,
	"FolderDelete":            bsFileIO,
	// Network — no server-side network access from EA
	"WebRequest":              "Network: no outbound HTTP from server-side EA",
	"SocketCreate":            bsNetwork,
	"SocketClose":             bsNetwork,
	"SocketConnect":           bsNetwork,
	"SocketIsConnected":       bsNetwork,
	"SocketIsReadable":        bsNetwork,
	"SocketIsWritable":        bsNetwork,
	"SocketTimeouts":          bsNetwork,
	"SocketRead":              bsNetwork,
	"SocketSend":              bsNetwork,
	"SocketTlsHandshake":      bsNetwork,
	"SocketTlsCertificate":    bsNetwork,
	"SocketTlsRead":           bsNetwork,
	"SocketTlsReadAvailable":  bsNetwork,
	"SocketTlsSend":           bsNetwork,
	"SendFTP":                 "Network: no FTP from server-side EA",
	"SendMail":                "Network: no email from server-side EA",
	"SendNotification":        "Network: no push notifications from server-side EA",
	// Database — use PostgreSQL instead
	"DatabaseOpen":            bsDatabase,
	"DatabaseClose":           bsDatabase,
	"DatabaseImport":          bsDatabase,
	"DatabaseExport":          bsDatabase,
	"DatabasePrint":           bsDatabase,
	"DatabaseTableExists":     bsDatabase,
	"DatabaseExecute":         bsDatabase,
	"DatabasePrepare":         bsDatabase,
	"DatabaseReset":           bsDatabase,
	"DatabaseBind":            bsDatabase,
	"DatabaseBindArray":       bsDatabase,
	"DatabaseRead":            bsDatabase,
	"DatabaseReadBind":        bsDatabase,
	"DatabaseFinalize":        bsDatabase,
	"DatabaseTransactionBegin":   bsDatabase,
	"DatabaseTransactionCommit":  bsDatabase,
	"DatabaseTransactionRollback": bsDatabase,
	"DatabaseColumnsCount":    bsDatabase,
	"DatabaseColumnName":      bsDatabase,
	"DatabaseColumnType":      bsDatabase,
	"DatabaseColumnSize":      bsDatabase,
	"DatabaseColumnText":      bsDatabase,
	"DatabaseColumnInteger":   bsDatabase,
	"DatabaseColumnLong":      bsDatabase,
	"DatabaseColumnDouble":    bsDatabase,
	"DatabaseColumnBlob":      bsDatabase,
	// OpenCL — GPU compute, not available server-side
	"CLBufferCreate":          bsOpenCL,
	"CLBufferFree":            bsOpenCL,
	"CLBufferRead":            bsOpenCL,
	"CLBufferWrite":           bsOpenCL,
	"CLContextCreate":         bsOpenCL,
	"CLContextFree":           bsOpenCL,
	"CLExecute":               bsOpenCL,
	"CLGetDeviceInfo":         bsOpenCL,
	"CLGetInfoInteger":        bsOpenCL,
	"CLHandleType":            bsOpenCL,
	"CLKernelCreate":          bsOpenCL,
	"CLKernelFree":            bsOpenCL,
	"CLProgramCreate":         bsOpenCL,
	"CLProgramFree":           bsOpenCL,
	"CLSetKernelArg":          bsOpenCL,
	"CLSetKernelArgMem":       bsOpenCL,
	// DirectX — graphics rendering, not available server-side
	"DXContextCreate":         bsDirectX,
	"DXContextSetSize":        bsDirectX,
	"DXContextClearColors":    bsDirectX,
	"DXContextClearDepth":     bsDirectX,
	"DXContextGetColors":      bsDirectX,
	"DXContextGetDepth":       bsDirectX,
	"DXBufferCreate":          bsDirectX,
	"DXTextureCreate":         bsDirectX,
	"DXInputCreate":           bsDirectX,
	"DXInputSet":              bsDirectX,
	"DXShaderCreate":          bsDirectX,
	"DXShaderSetLayout":       bsDirectX,
	"DXShaderInputsSet":       bsDirectX,
	"DXShaderTexturesSet":     bsDirectX,
	"DXDraw":                  bsDirectX,
	"DXDrawIndexed":           bsDirectX,
	"DXPrimiveTopologySet":    bsDirectX,
	"DXBufferSet":             bsDirectX,
	"DXShaderSet":             bsDirectX,
	"DXHandleType":            bsDirectX,
	"DXRelease":               bsDirectX,
	// Economic Calendar — not available in backtest
	"CalendarCountryById":         bsCalendar,
	"CalendarEventById":           bsCalendar,
	"CalendarValueById":           bsCalendar,
	"CalendarCountries":           bsCalendar,
	"CalendarEventByCountry":      bsCalendar,
	"CalendarEventByCurrency":     bsCalendar,
	"CalendarValueHistoryByEvent": bsCalendar,
	"CalendarValueHistory":        bsCalendar,
	"CalendarValueLastByEvent":    bsCalendar,
	"CalendarValueLast":           bsCalendar,
	// Custom Symbols — not applicable server-side
	"CustomSymbolCreate":          bsCustomSymbol,
	"CustomSymbolDelete":          bsCustomSymbol,
	"CustomSymbolSetInteger":      bsCustomSymbol,
	"CustomSymbolSetDouble":       bsCustomSymbol,
	"CustomSymbolSetString":       bsCustomSymbol,
	"CustomSymbolSetMarginRate":   bsCustomSymbol,
	"CustomSymbolSetSessionQuote": bsCustomSymbol,
	"CustomSymbolSetSessionTrade": bsCustomSymbol,
	"CustomRatesDelete":           bsCustomSymbol,
	"CustomRatesReplace":          bsCustomSymbol,
	"CustomRatesUpdate":           bsCustomSymbol,
	"CustomTicksAdd":              bsCustomSymbol,
	"CustomTicksDelete":           bsCustomSymbol,
	"CustomTicksReplace":          bsCustomSymbol,
	"CustomBookAdd":               bsCustomSymbol,
	// Global Variables of Terminal — remaining unimplemented
	"GlobalVariableSetOnCondition": "Global Variables: terminal client-side state",
	"GlobalVariablesFlush":         "Global Variables: terminal client-side state",
	"GlobalVariableTime":           "Global Variables: terminal client-side state",
	// Optimization Frames — strategy tester only
	"FrameAdd":                    "Optimization: strategy tester only",
	"FrameFilter":                 "Optimization: strategy tester only",
	"FrameFirst":                  "Optimization: strategy tester only",
	"FrameInputs":                 "Optimization: strategy tester only",
	"FrameNext":                   "Optimization: strategy tester only",
	"ParameterGetRange":           "Optimization: strategy tester only",
	"ParameterSetRange":           "Optimization: strategy tester only",
	// Trade Signals — terminal client-side
	"SignalBaseGetDouble":         bsSignals,
	"SignalBaseGetInteger":        bsSignals,
	"SignalBaseGetString":         bsSignals,
	"SignalBaseSelect":            bsSignals,
	"SignalBaseTotal":             bsSignals,
	"SignalInfoGetDouble":         bsSignals,
	"SignalInfoGetInteger":        bsSignals,
	"SignalInfoGetString":         bsSignals,
	"SignalInfoSetDouble":         bsSignals,
	"SignalInfoSetInteger":        bsSignals,
	"SignalSubscribe":             bsSignals,
	"SignalUnsubscribe":           bsSignals,
	// Custom Indicator functions — indicator development only
	"IndicatorCreate":             bsCustomInd,
	"IndicatorParameters":         bsCustomInd,
	"IndicatorRelease":            bsCustomInd,
	"IndicatorSetDouble":          bsCustomInd,
	"IndicatorSetInteger":         bsCustomInd,
	"IndicatorSetString":          bsCustomInd,
	"PlotIndexGetInteger":         bsCustomInd,
	"PlotIndexSetDouble":          bsCustomInd,
	"PlotIndexSetInteger":         bsCustomInd,
	"PlotIndexSetString":          bsCustomInd,
	"SetIndexBuffer":              bsCustomInd,
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
	"WindowBarsPerChart":     bsWindow,
	"WindowExpertName":       bsWindow,
	"WindowFind":             bsWindow,
	"WindowFirstVisibleBar":  bsWindow,
	"WindowHandle":           bsWindow,
	"WindowIsVisible":        bsWindow,
	"WindowOnDropped":        bsWindow,
	"WindowPriceMax":         bsWindow,
	"WindowPriceMin":         bsWindow,
	"WindowPriceOnDropped":   bsWindow,
	"WindowRedraw":           bsWindow,
	"WindowScreenShot":       bsWindow,
	"WindowsTotal":           bsWindow,
	"WindowTimeOnDropped":    bsWindow,
	"WindowXOnDropped":       bsWindow,
	"WindowYOnDropped":       bsWindow,
	// MQL4-only custom indicator functions — indicator development only
	"IndicatorBuffers":       bsCustomInd,
	"IndicatorCounted":       bsCustomInd,
	"IndicatorDigits":        bsCustomInd,
	"IndicatorShortName":     bsCustomInd,
	"SetIndexArrow":          bsCustomInd,
	"SetIndexDrawBegin":      bsCustomInd,
	"SetIndexEmptyValue":     bsCustomInd,
	"SetIndexLabel":          bsCustomInd,
	"SetIndexShift":          bsCustomInd,
	"SetIndexStyle":          bsCustomInd,
	"SetLevelStyle":          bsCustomInd,
	"SetLevelValue":          bsCustomInd,
	// MQL4-only Object functions
	"ObjectType":             bsObjectChart,
	"ObjectGetFiboDescription":   bsObjectChart,
	"ObjectSetFiboDescription":   bsObjectChart,
	"ObjectGetShiftByValue":  bsObjectChart,
	"ObjectGetValueByShift":  bsObjectChart,
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
	// Custom indicators — source-based execution not yet implemented
	"iCustom":                "Custom indicator: OnCalculate execution model + indicator source registration not implemented. VM returns 0.",
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
		if startsWith(name, p) {
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
	if startsWith(name, "Is") && len(name) > 2 && name[2] >= 'A' && name[2] <= 'Z' {
		return true
	}
	// Set*/Get* accessor pattern
	if (startsWith(name, "Set") || startsWith(name, "Get")) && len(name) > 3 && name[3] >= 'A' && name[3] <= 'Z' {
		return true
	}
	// MQL4 lowercase aliases (ceil, floor, cos, sin, etc.)
	if name[0] >= 'a' && name[0] <= 'z' && len(name) <= 8 {
		return true
	}
	return false
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
// Used for extracting parameter default values without a full evaluation pass.
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
