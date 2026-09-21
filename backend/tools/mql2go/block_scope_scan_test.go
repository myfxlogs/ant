package mql2go

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
)

// VM-BLOCK-SCOPE-1 compat gate: flag identifiers DECLARED inside an
// if/else/for/while/do/switch compound_statement that are REFERENCED after the
// block closes — these break under strict block scoping.
func declIdents(src string, n *sitter.Node, out map[string]bool) {
	for i := 0; i < int(n.NamedChildCount()); i++ {
		c := n.NamedChild(i)
		if c.Type() == "declaration" {
			for j := 0; j < int(c.NamedChildCount()); j++ {
				ch := c.NamedChild(j)
				switch ch.Type() {
				case "init_declarator", "declarator":
					// first identifier = declared name; rest is initializer
					if id := childByType(ch, nodeIdentifier); id != nil {
						out[src[id.StartByte():id.EndByte()]] = true
					}
				case "array_declarator":
					if id := childByType(ch, nodeIdentifier); id != nil {
						out[src[id.StartByte():id.EndByte()]] = true
					}
				case nodeIdentifier:
					out[src[ch.StartByte():ch.EndByte()]] = true
				}
			}
		}
	}
}

func identsInRange(src string, n *sitter.Node, start, end uint32, out map[string]bool) {
	if n == nil {
		return
	}
	if n.StartByte() >= end || n.EndByte() <= start {
		return
	}
	if n.Type() == nodeIdentifier && n.StartByte() >= start && n.EndByte() <= end {
		out[src[n.StartByte():n.EndByte()]] = true
	}
	for i := 0; i < int(n.NamedChildCount()); i++ {
		identsInRange(src, n.NamedChild(i), start, end, out)
	}
}

func scanScopes(src string, n *sitter.Node, parent *sitter.Node, hits *[]string) {
	for i := 0; i < int(n.NamedChildCount()); i++ {
		c := n.NamedChild(i)
		isBranch := parent != nil && c.Type() == nodeCompoundStatement &&
			(parent.Type() == "if_statement" || parent.Type() == "for_statement" ||
				parent.Type() == "while_statement" || parent.Type() == "do_statement" ||
				parent.Type() == "switch_statement" || parent.Type() == "else_clause")
		if isBranch {
			decls := map[string]bool{}
			declIdents(src, c, decls)
			if len(decls) > 0 && parent.Parent() != nil {
				fn := parent.Parent() // function_definition
				after := map[string]bool{}
				identsInRange(src, fn, c.EndByte(), fn.EndByte(), after)
				var leaked []string
				for d := range decls {
					if after[d] {
						leaked = append(leaked, d)
					}
				}
				if len(leaked) > 0 {
					sort.Strings(leaked)
					*hits = append(*hits, fmt.Sprintf("line %d: block decls leaked: %s (block ends byte %d)", c.StartPoint().Row+1, strings.Join(leaked, ","), c.EndByte()))
				}
			}
		}
		scanScopes(src, c, n, hits)
	}
}

func TestBlockScopeCompatScan(t *testing.T) {
	// Repo fixtures are the permanent gate; external corpus drops in via
	// /tmp/strats/*.mq4 when present (real-strategy compat scan).
	files, _ := filepath.Glob("testdata/*.mq4")
	more, _ := filepath.Glob("testdata/honesty/*.mq4")
	files = append(files, more...)
	ext, _ := filepath.Glob("/tmp/strats/*.mq4")
	files = append(files, ext...)
	if len(files) == 0 {
		t.Fatal("no .mq4 sources found — compat scan gate would be vacuous")
	}
	var unclean int
	for _, fn := range files {
		data, _ := os.ReadFile(fn)
		name, code := filepath.Base(fn), string(data)
		root, err := ParseMQL(code)
		if err != nil {
			t.Logf("[%s] parse error: %v", name, err)
			continue
		}
		var hits []string
		scanScopes(code, root, nil, &hits)
		for _, h := range hits {
			t.Logf("[%s] %s", name, h)
			unclean++
		}
		if len(hits) == 0 {
			t.Logf("[%s] CLEAN", name)
		}
	}
	if unclean > 0 {
		t.Fatalf("%d block-scope leak(s) detected — strict block scoping would break these sources", unclean)
	}
}
