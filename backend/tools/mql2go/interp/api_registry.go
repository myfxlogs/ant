package interp

// API Registry — Layer 0 of the MQL EA Compatibility Proposal (§12.1).
//
// Enumerates every known MQL4/MQL5 API symbol with a two-state status:
//   - StatusImplemented: fully functional in the VM
//   - StatusUnsupported: rejected at compile time (DLL, custom indicators, etc.)
//
// CI invariant (enforced by api_registry_test.go):
//   1. Every name in implemented* slices (builtin_registry.go) must appear
//      in the registry with StatusImplemented.
//   2. Every name in MQLConstants (constants.go) must appear in the registry
//      with StatusImplemented.
//   3. No duplicate names in the explicit apiRegistry list.
//
// When adding a new VM builtin:
//   1. Add the name to the appropriate implemented* slice in builtin_registry.go
//   2. The registry auto-populates it as StatusImplemented via init()
//   3. Run `go test ./tools/mql2go/interp/ -run TestAPIRegistry` to verify

// SymbolStatus is the two-state lifecycle of an API symbol.
type SymbolStatus int8

const (
	StatusImplemented SymbolStatus = iota
	StatusUnsupported
)

// SymbolCategory classifies the API domain.
type SymbolCategory int8

const (
	CatConstant SymbolCategory = iota
	CatFunction
	CatCTradeMethod
)

// APISymbol describes a single MQL API symbol.
type APISymbol struct {
	Name     string
	Status   SymbolStatus
	Category SymbolCategory
	MQLVer   int8 // 0 = both, 4 = MQL4 only, 5 = MQL5 only
	Reason   string
}

// unsupported reason constants (goconst: repeated strings > 5 occurrences).
const (
	reasonGUI      = "GUI functions are not supported"
	reasonChart    = "chart objects are not supported"
	reasonChartOps = "chart operations are not supported"
	reasonFileIO   = "file I/O is not supported"
	reasonNetwork  = "network sockets are not supported"
	reasonCrypto   = "cryptographic functions are not supported"
	reasonResource = "graphical resources are not supported"
	reasonCustom   = "custom indicators are not supported"
	reasonFTP      = "FTP is not supported"
	reasonEmail    = "email is not supported"
	reasonAudio    = "audio is not supported"
	reasonPush     = "push notifications are not supported"
	reasonHTTP     = "HTTP requests are not supported"
)

// unsupportedSymbols lists MQL functions that are explicitly NOT supported.
// The compiler rejects these with a clear error message instead of silent failure.
var unsupportedSymbols = []APISymbol{
	{Name: "iCustom", Status: StatusUnsupported, Category: CatFunction, Reason: reasonCustom},
	{Name: "MessageBox", Status: StatusUnsupported, Category: CatFunction, Reason: reasonGUI},
	{Name: "ObjectCreate", Status: StatusUnsupported, Category: CatFunction, Reason: reasonChart},
	{Name: "ObjectDelete", Status: StatusUnsupported, Category: CatFunction, Reason: reasonChart},
	{Name: "ObjectSet", Status: StatusUnsupported, Category: CatFunction, Reason: reasonChart},
	{Name: "ObjectGet", Status: StatusUnsupported, Category: CatFunction, Reason: reasonChart},
	{Name: "ObjectFind", Status: StatusUnsupported, Category: CatFunction, Reason: reasonChart},
	{Name: "ObjectName", Status: StatusUnsupported, Category: CatFunction, Reason: reasonChart},
	{Name: "ObjectsTotal", Status: StatusUnsupported, Category: CatFunction, Reason: reasonChart},
	{Name: "ObjectDescription", Status: StatusUnsupported, Category: CatFunction, Reason: reasonChart},
	{Name: "ObjectSetText", Status: StatusUnsupported, Category: CatFunction, Reason: reasonChart},
	{Name: "ObjectSetInteger", Status: StatusUnsupported, Category: CatFunction, Reason: reasonChart},
	{Name: "ObjectGetInteger", Status: StatusUnsupported, Category: CatFunction, Reason: reasonChart},
	{Name: "ObjectSetDouble", Status: StatusUnsupported, Category: CatFunction, Reason: reasonChart},
	{Name: "ObjectGetDouble", Status: StatusUnsupported, Category: CatFunction, Reason: reasonChart},
	{Name: "ObjectSetString", Status: StatusUnsupported, Category: CatFunction, Reason: reasonChart},
	{Name: "ObjectGetString", Status: StatusUnsupported, Category: CatFunction, Reason: reasonChart},
	{Name: "ObjectGetType", Status: StatusUnsupported, Category: CatFunction, Reason: reasonChart},
	{Name: "ChartApplyTemplate", Status: StatusUnsupported, Category: CatFunction, Reason: reasonChartOps},
	{Name: "ChartSaveTemplate", Status: StatusUnsupported, Category: CatFunction, Reason: reasonChartOps},
	{Name: "ChartOpen", Status: StatusUnsupported, Category: CatFunction, Reason: reasonChartOps},
	{Name: "ChartClose", Status: StatusUnsupported, Category: CatFunction, Reason: reasonChartOps},
	{Name: "ChartNavigate", Status: StatusUnsupported, Category: CatFunction, Reason: reasonChartOps},
	{Name: "ChartSymbol", Status: StatusUnsupported, Category: CatFunction, Reason: reasonChartOps},
	{Name: "ChartPeriod", Status: StatusUnsupported, Category: CatFunction, Reason: reasonChartOps},
	{Name: "ChartRedraw", Status: StatusUnsupported, Category: CatFunction, Reason: reasonChartOps},
	{Name: "WindowFind", Status: StatusUnsupported, Category: CatFunction, Reason: reasonGUI},
	{Name: "WindowOnDropped", Status: StatusUnsupported, Category: CatFunction, Reason: reasonGUI},
	{Name: "WindowPriceOnDropped", Status: StatusUnsupported, Category: CatFunction, Reason: reasonGUI},
	{Name: "WindowTimeOnDropped", Status: StatusUnsupported, Category: CatFunction, Reason: reasonGUI},
	{Name: "WindowXOnDropped", Status: StatusUnsupported, Category: CatFunction, Reason: reasonGUI},
	{Name: "WindowYOnDropped", Status: StatusUnsupported, Category: CatFunction, Reason: reasonGUI},
	{Name: "WindowBarsPerChart", Status: StatusUnsupported, Category: CatFunction, Reason: reasonGUI},
	{Name: "WindowFirstVisibleBar", Status: StatusUnsupported, Category: CatFunction, Reason: reasonGUI},
	{Name: "WindowPriceMax", Status: StatusUnsupported, Category: CatFunction, Reason: reasonGUI},
	{Name: "WindowPriceMin", Status: StatusUnsupported, Category: CatFunction, Reason: reasonGUI},
	{Name: "WindowScreenShot", Status: StatusUnsupported, Category: CatFunction, Reason: reasonGUI},
	{Name: "WindowExpertName", Status: StatusUnsupported, Category: CatFunction, Reason: reasonGUI},
	{Name: "WindowHandle", Status: StatusUnsupported, Category: CatFunction, Reason: reasonGUI},
	{Name: "SendNotification", Status: StatusUnsupported, Category: CatFunction, Reason: reasonPush},
	{Name: "SendFTP", Status: StatusUnsupported, Category: CatFunction, Reason: reasonFTP},
	{Name: "SendMail", Status: StatusUnsupported, Category: CatFunction, Reason: reasonEmail},
	{Name: "PlaySound", Status: StatusUnsupported, Category: CatFunction, Reason: reasonAudio},
	{Name: "FileOpen", Status: StatusUnsupported, Category: CatFunction, Reason: reasonFileIO},
	{Name: "FileClose", Status: StatusUnsupported, Category: CatFunction, Reason: reasonFileIO},
	{Name: "FileWrite", Status: StatusUnsupported, Category: CatFunction, Reason: reasonFileIO},
	{Name: "FileRead", Status: StatusUnsupported, Category: CatFunction, Reason: reasonFileIO},
	{Name: "FileReadString", Status: StatusUnsupported, Category: CatFunction, Reason: reasonFileIO},
	{Name: "FileReadInteger", Status: StatusUnsupported, Category: CatFunction, Reason: reasonFileIO},
	{Name: "FileReadDouble", Status: StatusUnsupported, Category: CatFunction, Reason: reasonFileIO},
	{Name: "FileReadNumber", Status: StatusUnsupported, Category: CatFunction, Reason: reasonFileIO},
	{Name: "FileSeek", Status: StatusUnsupported, Category: CatFunction, Reason: reasonFileIO},
	{Name: "FileSize", Status: StatusUnsupported, Category: CatFunction, Reason: reasonFileIO},
	{Name: "FileTell", Status: StatusUnsupported, Category: CatFunction, Reason: reasonFileIO},
	{Name: "FileIsEnding", Status: StatusUnsupported, Category: CatFunction, Reason: reasonFileIO},
	{Name: "FileFlush", Status: StatusUnsupported, Category: CatFunction, Reason: reasonFileIO},
	{Name: "FileCopy", Status: StatusUnsupported, Category: CatFunction, Reason: reasonFileIO},
	{Name: "FileDelete", Status: StatusUnsupported, Category: CatFunction, Reason: reasonFileIO},
	{Name: "FileMove", Status: StatusUnsupported, Category: CatFunction, Reason: reasonFileIO},
	{Name: "FileIsExist", Status: StatusUnsupported, Category: CatFunction, Reason: reasonFileIO},
	{Name: "FileFindFirst", Status: StatusUnsupported, Category: CatFunction, Reason: reasonFileIO},
	{Name: "FileFindNext", Status: StatusUnsupported, Category: CatFunction, Reason: reasonFileIO},
	{Name: "FileFindClose", Status: StatusUnsupported, Category: CatFunction, Reason: reasonFileIO},
	{Name: "FolderCreate", Status: StatusUnsupported, Category: CatFunction, Reason: reasonFileIO},
	{Name: "FolderDelete", Status: StatusUnsupported, Category: CatFunction, Reason: reasonFileIO},
	{Name: "FolderClean", Status: StatusUnsupported, Category: CatFunction, Reason: reasonFileIO},
	{Name: "CryptEncode", Status: StatusUnsupported, Category: CatFunction, Reason: reasonCrypto},
	{Name: "CryptDecode", Status: StatusUnsupported, Category: CatFunction, Reason: reasonCrypto},
	{Name: "SocketCreate", Status: StatusUnsupported, Category: CatFunction, Reason: reasonNetwork},
	{Name: "SocketConnect", Status: StatusUnsupported, Category: CatFunction, Reason: reasonNetwork},
	{Name: "SocketClose", Status: StatusUnsupported, Category: CatFunction, Reason: reasonNetwork},
	{Name: "SocketSend", Status: StatusUnsupported, Category: CatFunction, Reason: reasonNetwork},
	{Name: "SocketRead", Status: StatusUnsupported, Category: CatFunction, Reason: reasonNetwork},
	{Name: "SocketTimeouts", Status: StatusUnsupported, Category: CatFunction, Reason: reasonNetwork},
	{Name: "WebRequest", Status: StatusUnsupported, Category: CatFunction, Reason: reasonHTTP},
	{Name: "ResourceCreate", Status: StatusUnsupported, Category: CatFunction, Reason: reasonResource},
	{Name: "ResourceFree", Status: StatusUnsupported, Category: CatFunction, Reason: reasonResource},
}

// registryMap is the lookup index built from unsupportedSymbols + builtin_registry.go + constants.go.
var registryMap map[string]APISymbol

func init() {
	registryMap = make(map[string]APISymbol, 600)

	// 1. Add unsupported functions first (explicit entries).
	for _, s := range unsupportedSymbols {
		registryMap[s.Name] = s
	}

	// 2. Add all implemented functions from builtin_registry.go as StatusImplemented.
	//    These override any unsupported entry with the same name (shouldn't happen,
	//    but implemented takes precedence).
	for _, name := range implementedMarketData {
		registryMap[name] = APISymbol{Name: name, Status: StatusImplemented, Category: CatFunction}
	}
	for _, name := range implementedIndicators {
		registryMap[name] = APISymbol{Name: name, Status: StatusImplemented, Category: CatFunction}
	}
	for _, name := range implementedMQL4Trade {
		registryMap[name] = APISymbol{Name: name, Status: StatusImplemented, Category: CatFunction, MQLVer: 4}
	}
	for _, name := range implementedMQL5Position {
		registryMap[name] = APISymbol{Name: name, Status: StatusImplemented, Category: CatFunction, MQLVer: 5}
	}
	for _, name := range implementedAccount {
		registryMap[name] = APISymbol{Name: name, Status: StatusImplemented, Category: CatFunction}
	}
	for _, name := range implementedPlatform {
		registryMap[name] = APISymbol{Name: name, Status: StatusImplemented, Category: CatFunction}
	}
	for _, name := range implementedCTradeMethods {
		registryMap[name] = APISymbol{Name: name, Status: StatusImplemented, Category: CatCTradeMethod}
	}

	// 3. Add all constants from MQLConstants as StatusImplemented.
	for name := range MQLConstants {
		registryMap[name] = APISymbol{Name: name, Status: StatusImplemented, Category: CatConstant}
	}
}

// LookupAPI looks up a symbol in the API registry.
// Returns the symbol and true if found, or zero value and false if not.
// KB-first: when the KB function hook is set, it can override the status
// (e.g., a function newly recorded as supported via RecordFact).
func LookupAPI(name string) (APISymbol, bool) {
	if kbFunctionLookup != nil {
		if supported, severity := kbFunctionLookup(name); supported {
			return APISymbol{Name: name, Status: StatusImplemented, Category: CatFunction}, true
		} else if severity != "" {
			return APISymbol{Name: name, Status: StatusUnsupported, Category: CatFunction, Reason: severity}, true
		}
	}
	s, ok := registryMap[name]
	return s, ok
}

// IsAPIImplemented returns true if the symbol exists and is StatusImplemented.
func IsAPIImplemented(name string) bool {
	s, ok := registryMap[name]
	return ok && s.Status == StatusImplemented
}

// IsAPIUnsupported returns true if the symbol exists and is StatusUnsupported.
func IsAPIUnsupported(name string) bool {
	s, ok := registryMap[name]
	return ok && s.Status == StatusUnsupported
}

// AllImplementedFunctions returns all function/CTrade names with StatusImplemented.
// Used by the CI reconciliation test.
func AllImplementedFunctions() []string {
	result := make([]string, 0, 256)
	for name, s := range registryMap {
		if s.Status == StatusImplemented && (s.Category == CatFunction || s.Category == CatCTradeMethod) {
			result = append(result, name)
		}
	}
	return result
}

// AllUnsupportedFunctions returns all function names with StatusUnsupported.
func AllUnsupportedFunctions() []string {
	result := make([]string, 0, len(unsupportedSymbols))
	for name, s := range registryMap {
		if s.Status == StatusUnsupported && s.Category == CatFunction {
			result = append(result, name)
		}
	}
	return result
}

// AllRegisteredConstants returns all constant names in the registry.
func AllRegisteredConstants() []string {
	result := make([]string, 0, 256)
	for name, s := range registryMap {
		if s.Category == CatConstant {
			result = append(result, name)
		}
	}
	return result
}
