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

// unsupportedSymbols lists MQL functions that are explicitly NOT supported.
// The compiler rejects these with a clear error message instead of silent failure.
var unsupportedSymbols = []APISymbol{
	{Name: "iCustom", Status: StatusUnsupported, Category: CatFunction, Reason: "custom indicators are not supported"},
	{Name: "MessageBox", Status: StatusUnsupported, Category: CatFunction, Reason: "GUI functions are not supported"},
	{Name: "ObjectCreate", Status: StatusUnsupported, Category: CatFunction, Reason: "chart objects are not supported"},
	{Name: "ObjectDelete", Status: StatusUnsupported, Category: CatFunction, Reason: "chart objects are not supported"},
	{Name: "ObjectSet", Status: StatusUnsupported, Category: CatFunction, Reason: "chart objects are not supported"},
	{Name: "ObjectGet", Status: StatusUnsupported, Category: CatFunction, Reason: "chart objects are not supported"},
	{Name: "ObjectFind", Status: StatusUnsupported, Category: CatFunction, Reason: "chart objects are not supported"},
	{Name: "ObjectName", Status: StatusUnsupported, Category: CatFunction, Reason: "chart objects are not supported"},
	{Name: "ObjectsTotal", Status: StatusUnsupported, Category: CatFunction, Reason: "chart objects are not supported"},
	{Name: "ObjectDescription", Status: StatusUnsupported, Category: CatFunction, Reason: "chart objects are not supported"},
	{Name: "ObjectSetText", Status: StatusUnsupported, Category: CatFunction, Reason: "chart objects are not supported"},
	{Name: "ObjectSetInteger", Status: StatusUnsupported, Category: CatFunction, Reason: "chart objects are not supported"},
	{Name: "ObjectGetInteger", Status: StatusUnsupported, Category: CatFunction, Reason: "chart objects are not supported"},
	{Name: "ObjectSetDouble", Status: StatusUnsupported, Category: CatFunction, Reason: "chart objects are not supported"},
	{Name: "ObjectGetDouble", Status: StatusUnsupported, Category: CatFunction, Reason: "chart objects are not supported"},
	{Name: "ObjectSetString", Status: StatusUnsupported, Category: CatFunction, Reason: "chart objects are not supported"},
	{Name: "ObjectGetString", Status: StatusUnsupported, Category: CatFunction, Reason: "chart objects are not supported"},
	{Name: "ObjectGetType", Status: StatusUnsupported, Category: CatFunction, Reason: "chart objects are not supported"},
	{Name: "ChartApplyTemplate", Status: StatusUnsupported, Category: CatFunction, Reason: "chart operations are not supported"},
	{Name: "ChartSaveTemplate", Status: StatusUnsupported, Category: CatFunction, Reason: "chart operations are not supported"},
	{Name: "ChartOpen", Status: StatusUnsupported, Category: CatFunction, Reason: "chart operations are not supported"},
	{Name: "ChartClose", Status: StatusUnsupported, Category: CatFunction, Reason: "chart operations are not supported"},
	{Name: "ChartNavigate", Status: StatusUnsupported, Category: CatFunction, Reason: "chart operations are not supported"},
	{Name: "ChartSymbol", Status: StatusUnsupported, Category: CatFunction, Reason: "chart operations are not supported"},
	{Name: "ChartPeriod", Status: StatusUnsupported, Category: CatFunction, Reason: "chart operations are not supported"},
	{Name: "ChartRedraw", Status: StatusUnsupported, Category: CatFunction, Reason: "chart operations are not supported"},
	{Name: "WindowFind", Status: StatusUnsupported, Category: CatFunction, Reason: "GUI functions are not supported"},
	{Name: "WindowOnDropped", Status: StatusUnsupported, Category: CatFunction, Reason: "GUI functions are not supported"},
	{Name: "WindowPriceOnDropped", Status: StatusUnsupported, Category: CatFunction, Reason: "GUI functions are not supported"},
	{Name: "WindowTimeOnDropped", Status: StatusUnsupported, Category: CatFunction, Reason: "GUI functions are not supported"},
	{Name: "WindowXOnDropped", Status: StatusUnsupported, Category: CatFunction, Reason: "GUI functions are not supported"},
	{Name: "WindowYOnDropped", Status: StatusUnsupported, Category: CatFunction, Reason: "GUI functions are not supported"},
	{Name: "WindowBarsPerChart", Status: StatusUnsupported, Category: CatFunction, Reason: "GUI functions are not supported"},
	{Name: "WindowFirstVisibleBar", Status: StatusUnsupported, Category: CatFunction, Reason: "GUI functions are not supported"},
	{Name: "WindowPriceMax", Status: StatusUnsupported, Category: CatFunction, Reason: "GUI functions are not supported"},
	{Name: "WindowPriceMin", Status: StatusUnsupported, Category: CatFunction, Reason: "GUI functions are not supported"},
	{Name: "WindowScreenShot", Status: StatusUnsupported, Category: CatFunction, Reason: "GUI functions are not supported"},
	{Name: "WindowExpertName", Status: StatusUnsupported, Category: CatFunction, Reason: "GUI functions are not supported"},
	{Name: "WindowHandle", Status: StatusUnsupported, Category: CatFunction, Reason: "GUI functions are not supported"},
	{Name: "SendNotification", Status: StatusUnsupported, Category: CatFunction, Reason: "push notifications are not supported"},
	{Name: "SendFTP", Status: StatusUnsupported, Category: CatFunction, Reason: "FTP is not supported"},
	{Name: "SendMail", Status: StatusUnsupported, Category: CatFunction, Reason: "email is not supported"},
	{Name: "PlaySound", Status: StatusUnsupported, Category: CatFunction, Reason: "audio is not supported"},
	{Name: "FileOpen", Status: StatusUnsupported, Category: CatFunction, Reason: "file I/O is not supported"},
	{Name: "FileClose", Status: StatusUnsupported, Category: CatFunction, Reason: "file I/O is not supported"},
	{Name: "FileWrite", Status: StatusUnsupported, Category: CatFunction, Reason: "file I/O is not supported"},
	{Name: "FileRead", Status: StatusUnsupported, Category: CatFunction, Reason: "file I/O is not supported"},
	{Name: "FileReadString", Status: StatusUnsupported, Category: CatFunction, Reason: "file I/O is not supported"},
	{Name: "FileReadInteger", Status: StatusUnsupported, Category: CatFunction, Reason: "file I/O is not supported"},
	{Name: "FileReadDouble", Status: StatusUnsupported, Category: CatFunction, Reason: "file I/O is not supported"},
	{Name: "FileReadNumber", Status: StatusUnsupported, Category: CatFunction, Reason: "file I/O is not supported"},
	{Name: "FileSeek", Status: StatusUnsupported, Category: CatFunction, Reason: "file I/O is not supported"},
	{Name: "FileSize", Status: StatusUnsupported, Category: CatFunction, Reason: "file I/O is not supported"},
	{Name: "FileTell", Status: StatusUnsupported, Category: CatFunction, Reason: "file I/O is not supported"},
	{Name: "FileIsEnding", Status: StatusUnsupported, Category: CatFunction, Reason: "file I/O is not supported"},
	{Name: "FileFlush", Status: StatusUnsupported, Category: CatFunction, Reason: "file I/O is not supported"},
	{Name: "FileCopy", Status: StatusUnsupported, Category: CatFunction, Reason: "file I/O is not supported"},
	{Name: "FileDelete", Status: StatusUnsupported, Category: CatFunction, Reason: "file I/O is not supported"},
	{Name: "FileMove", Status: StatusUnsupported, Category: CatFunction, Reason: "file I/O is not supported"},
	{Name: "FileIsExist", Status: StatusUnsupported, Category: CatFunction, Reason: "file I/O is not supported"},
	{Name: "FileFindFirst", Status: StatusUnsupported, Category: CatFunction, Reason: "file I/O is not supported"},
	{Name: "FileFindNext", Status: StatusUnsupported, Category: CatFunction, Reason: "file I/O is not supported"},
	{Name: "FileFindClose", Status: StatusUnsupported, Category: CatFunction, Reason: "file I/O is not supported"},
	{Name: "FolderCreate", Status: StatusUnsupported, Category: CatFunction, Reason: "file I/O is not supported"},
	{Name: "FolderDelete", Status: StatusUnsupported, Category: CatFunction, Reason: "file I/O is not supported"},
	{Name: "FolderClean", Status: StatusUnsupported, Category: CatFunction, Reason: "file I/O is not supported"},
	{Name: "CryptEncode", Status: StatusUnsupported, Category: CatFunction, Reason: "cryptographic functions are not supported"},
	{Name: "CryptDecode", Status: StatusUnsupported, Category: CatFunction, Reason: "cryptographic functions are not supported"},
	{Name: "SocketCreate", Status: StatusUnsupported, Category: CatFunction, Reason: "network sockets are not supported"},
	{Name: "SocketConnect", Status: StatusUnsupported, Category: CatFunction, Reason: "network sockets are not supported"},
	{Name: "SocketClose", Status: StatusUnsupported, Category: CatFunction, Reason: "network sockets are not supported"},
	{Name: "SocketSend", Status: StatusUnsupported, Category: CatFunction, Reason: "network sockets are not supported"},
	{Name: "SocketRead", Status: StatusUnsupported, Category: CatFunction, Reason: "network sockets are not supported"},
	{Name: "SocketTimeouts", Status: StatusUnsupported, Category: CatFunction, Reason: "network sockets are not supported"},
	{Name: "WebRequest", Status: StatusUnsupported, Category: CatFunction, Reason: "HTTP requests are not supported"},
	{Name: "ResourceCreate", Status: StatusUnsupported, Category: CatFunction, Reason: "graphical resources are not supported"},
	{Name: "ResourceFree", Status: StatusUnsupported, Category: CatFunction, Reason: "graphical resources are not supported"},
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
func LookupAPI(name string) (APISymbol, bool) {
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
