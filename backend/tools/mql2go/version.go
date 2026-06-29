package mql2go

import (
	"context"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
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

// detectMQLVersion determines whether the source is MQL4 or MQL5
// based on characteristic MQL5 signals.
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
