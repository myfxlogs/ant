package mql2go

import (
	"context"

	sitter "github.com/smacker/go-tree-sitter"
	python "github.com/smacker/go-tree-sitter/python"
)

// ParsePython parses Python source into a tree-sitter CST.
// Uses the built-in Python grammar from smacker/go-tree-sitter.
func ParsePython(source string) (*sitter.Node, error) {
	parser := sitter.NewParser()
	defer parser.Close()
	parser.SetLanguage(python.GetLanguage())

	tree, err := parser.ParseCtx(context.Background(), nil, []byte(source))
	if err != nil {
		return nil, err
	}
	return tree.RootNode(), nil
}
