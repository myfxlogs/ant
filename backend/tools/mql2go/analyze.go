package mql2go

import (
	"context"
	"strings"
	"sync"

	sitter "github.com/smacker/go-tree-sitter"
)

// parseSource holds the MQL source being analyzed — set by Analyze(),
// used by CST helpers for text extraction. Protected by analyzeMu.
var (
	parseSource string
	analyzeMu   sync.Mutex
)

// ParseMQL parses MQL source into a tree-sitter CST.
func ParseMQL(source string) (*sitter.Node, error) {
	lang, err := Language()
	if err != nil {
		return nil, err
	}
	parser := sitter.NewParser()
	defer parser.Close()
	parser.SetLanguage(lang)

	tree, err := parser.ParseCtx(context.Background(), nil, []byte(source))
	if err != nil {
		return nil, err
	}
	return tree.RootNode(), nil
}

// Analyze parses MQL source and extracts a full StrategyIntent.
// Thread-safe: serializes access to the global parseSource used by CST helpers.
// MQL4 and MQL5 have completely different trading APIs — extraction is
// version-aware to avoid cross-contamination.
func Analyze(source string) (*StrategyIntent, error) {
	analyzeMu.Lock()
	parseSource = source
	defer analyzeMu.Unlock()

	root, err := ParseMQL(source)
	if err != nil {
		return analyzeFallback(source), nil
	}

	version := detectMQLVersion(source)

	intent := &StrategyIntent{
		Meta: StrategyMeta{
			MQLVersion: version,
		},
		Params:     extractParamsCST(source, root),
		State:      extractStateCST(source, root),
		Entry:      extractEntriesCST(root, version),
		Exit:       extractExitsCST(root, version),
		Modifies:      extractModifiesCST(root, version),
		OrderLoops:    extractOrderLoopsCST(root, version),
		PositionLoops: extractPositionLoopsCST(root, version),
		Indicators:    extractIndicatorsCST(root),
		Execution:  detectExecCST(root),
		Timer:      detectTimerCST(root),
		Risk:       extractRiskChecksCST(root, version),
	}
	intent.Sizing = detectSizingCST(intent.Entry)
	intent.BlindSpots = detectBlindSpots(source, root, intent)
	return intent, nil
}

func analyzeFallback(source string) *StrategyIntent {
	return &StrategyIntent{
		Meta:      StrategyMeta{MQLVersion: detectMQLVersion(source)},
		Execution: ExecutionModel{Kind: ExecOnBar},
	}
}

func detectMQLVersion(source string) string {
	if strings.Contains(source, "class ") || strings.Contains(source, "CTrade") ||
		strings.Contains(source, "#include <Trade\\") ||
		strings.Contains(source, "MqlTradeRequest") || strings.Contains(source, "MqlTradeResult") ||
		strings.Contains(source, "OnTradeTransaction") || strings.Contains(source, "OnBookEvent") ||
		strings.Contains(source, "PositionGetDouble") || strings.Contains(source, "PositionGetInteger") ||
		strings.Contains(source, "PositionGetString") || strings.Contains(source, "PositionGetTicket") ||
		strings.Contains(source, "PositionSelectByTicket") {
		return "mql5"
	}
	return "mql4"
}
