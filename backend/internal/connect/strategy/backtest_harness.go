package strategy

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
)

// findStrategyTypeName parses Go source code and returns the name of the
// type that implements sdk.Strategy (the type with an OnInit receiver method).
func findStrategyTypeName(code string) (string, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "strategy.go", code, 0)
	if err != nil {
		return "", fmt.Errorf("parse strategy code: %w", err)
	}
	for _, decl := range f.Decls {
		fnDecl, ok := decl.(*ast.FuncDecl)
		if !ok || fnDecl.Recv == nil || len(fnDecl.Recv.List) == 0 {
			continue
		}
		if fnDecl.Name.Name != "OnInit" {
			continue
		}
		switch t := fnDecl.Recv.List[0].Type.(type) {
		case *ast.StarExpr:
			if ident, ok := t.X.(*ast.Ident); ok {
				return ident.Name, nil
			}
		case *ast.Ident:
			return t.Name, nil
		}
	}
	return "", fmt.Errorf("no type with OnInit method found in strategy code")
}

// generateBacktestHarness generates a backtest harness for a compiled Go strategy.
// The strategy type name is injected to instantiate the strategy at runtime.
func generateBacktestHarness(strategyTypeName string) string {
	return generateBacktestHarnessBase(
		fmt.Sprintf("strategy := &%s{}", strategyTypeName),
		"",
	)
}

// generateLiveHarness generates a live harness for a compiled Go strategy.
// The strategy type name is injected to instantiate the strategy at runtime.
func generateLiveHarness(strategyTypeName string) string {
	return generateLiveHarnessBase(
		fmt.Sprintf("strategy := &%s{}", strategyTypeName),
		"",
	)
}
