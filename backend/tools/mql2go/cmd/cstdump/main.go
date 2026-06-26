package main

import (
	"context"
	"fmt"
	"os"
	sitter "github.com/smacker/go-tree-sitter"
	m "anttrader/tools/mql2go"
)

func dump(n *sitter.Node, src string, depth int) {
	if n == nil { return }
	p := ""
	for i := 0; i < depth; i++ { p += "  " }
	t := ""
	if n.StartByte() < n.EndByte() && int(n.EndByte()) <= len(src) {
		t = src[n.StartByte():n.EndByte()]
		if len(t) > 70 { t = t[:70] + "..." }
	}
	fmt.Printf("%s%s %q\n", p, n.Type(), t)
	for i := 0; i < int(n.ChildCount()); i++ { dump(n.Child(i), src, depth+1) }
}

func main() {
	src, _ := os.ReadFile(os.Args[1])
	lang, _ := m.Language()
	p := sitter.NewParser()
	p.SetLanguage(lang)
	tree, _ := p.ParseCtx(context.Background(), nil, src)
	fmt.Println("=== CST dump ===")
	dump(tree.RootNode(), string(src), 0)
}
