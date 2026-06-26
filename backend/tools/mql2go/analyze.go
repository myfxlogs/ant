package mql2go

import (
	"context"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// parseSource holds the MQL source being analyzed — set by Analyze(),
// used by CST helpers for text extraction.
var parseSource string

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
	// Keep tree alive — root node references tree memory.
	// In a CLI tool, the process exits after generating code.
	// For a long-running server, use a pool of trees.
	_ = tree
	return tree.RootNode(), nil
}

// Analyze parses MQL source and extracts a full StrategyIntent.
func Analyze(source string) (*StrategyIntent, error) {
	parseSource = source // store for CST text extraction
	root, err := ParseMQL(source)
	if err != nil {
		return analyzeFallback(source), nil
	}

	intent := &StrategyIntent{
		Meta: StrategyMeta{
			MQLVersion: detectMQLVersion(source),
		},
		Params:     extractParamsCST(source, root),
		State:      extractStateCST(source, root),
		Entry:      extractEntriesCST(root),
		Exit:       extractExitsCST(root),
		Indicators: extractIndicatorsCST(root),
		Execution:  detectExecCST(root),
		Timer:      detectTimerCST(root),
	}
	intent.Sizing = detectSizingCST(intent.Entry)
	return intent, nil
}

func analyzeFallback(source string) *StrategyIntent {
	return &StrategyIntent{
		Meta:      StrategyMeta{MQLVersion: detectMQLVersion(source)},
		Execution: ExecutionModel{Kind: ExecOnBar},
	}
}

func detectMQLVersion(source string) string {
	if strings.Contains(source, "class ") {
		return "mql5"
	}
	return "mql4"
}
