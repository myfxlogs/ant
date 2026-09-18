# 施工派工单 — PY-DECIMAL-CTOR-1

> 设计 SSOT（Devin CLI 设计实查 2026-09-18）。施工方只施工不决策；遇偏差停下上报。

## 0. 立项背景

registry 行 203（QS-1.4 复审发现）：`compile_py_expr.go` `case "Decimal"` 透传参数——`Decimal("0.1")` 产 `ValString` 非 `ValDecimal`。

## 1. 设计实查实证（已核实，勿重复审计）

### D1 缺陷实况（比 registry 原载更严重）

`case "Decimal"` :174-178 `return &args[0]` 透传。`interp/value.go` **`ToDecimal()` 无 `ValString` 分支**（default→`decimal.Zero`），`argD` 同通道——后果链不止 bool/类型失真：

| 表达式 | 现状 | 正确值 |
|---|---|---|
| `bool(Decimal("0"))` | `true`（"0"≠""） | `false` |
| `Decimal("1.5")+Decimal("1.5")` | 0+0=0 或串拼接 | 3 |
| `ctx.ask() - Decimal("0.0050")` | ask-0=ask（**sl 静默失效**） | ask-0.005 |
| `Decimal("0.0")==Decimal("0")` | "0.0"=="0"→false | true |
| `broker.buy(lot=Decimal("0.1"))` | lot→0 → invalid volume 路径 | 0.1 |

### D2 字面量形态（已核实）

`compile_py_expr.go`：string→`ExprLiteral{StringVal}`(:35)、int→`IntVal`(:123)、float→`DecimalVal`(:31)、true/false→`BoolVal`(:38-41)、none→`NoneVal`(:44)、concatenated_string→`StringVal`(:104)。

### D3 非字面量转换路径（已核实）

- `ExprCall{Name}` 经 `classifyCall`→`IsBuiltinImplemented`→`LookupAPI`（`interp/analyze.go:166`/`builtin_registry.go`）。
- `"StringToDouble"` 在 `implementedPlatform`（`builtin_registry.go:100`），`builtinStringToDouble`=`decimal.NewFromString`（err→0，MQL 语义，`vm_builtin_util.go:64`）——**合法且已注册的 ExprCall 目标**。
- `case "Decimal"` 在 generic ExprCall fallback 前拦截→`Decimal` 名不进 classifyCall，无需 registry 改动。

### D4 裁定

- 字面量**常量折叠**（覆盖全部真实用法——存量 12+ 处 `Decimal("x")` 全字面量）；非法字面量 `c.errorf` 编译拒绝（fail-closed，部署期暴露优于 Python 运行时 raise）。
- 非字面量→`ExprCall{Name:"StringToDouble"}`（ValDecimal 往返精确/ValInt 精确/ValString 解析）；**已知残留**：垃圾字符串→0（Python 应 raise）、`Decimal(bool_var)`→0（Python=1）——如实记录接受（全部真实用法为字面量；替代方案污染 api_registry 已否决）。
- 零参 `Decimal()`→Python 合法返 `Decimal('0')`——现状 `DecimalVal(0)` 已正确，**保留**。
- `case "Decimal"` 遮蔽用户同名函数——pre-existing 边界，不动。

### D5 同族观察（边界外，仅登记）

`int`/`float`/`str` 同型透传（`:150-164`）——`int("42")`→StringVal→ToInt→0 同静默零族。本债范围仅 `Decimal`；同族已在 registry 备注，另债处置。

### D6 依赖面

存量 `Decimal("...")` 测试全编译断言（IR/AST 成功），折叠后仍合法 IR——零破坏。无 `ExprCall{Name:"Decimal"}` 形态断言。

## 2. 施工步骤

### S1 `compile_py_expr.go` `case "Decimal"` 重写

```go
case "Decimal":
	if len(args) == 0 {
		return &interp.Expr{Kind: interp.ExprLiteral, Val: interp.DecimalVal(decimalZero)}
	}
	arg := args[0]
	if arg.Kind == interp.ExprLiteral {
		switch arg.Val.Kind {
		case interp.ValDecimal:
			return &arg
		case interp.ValInt:
			return &interp.Expr{Kind: interp.ExprLiteral,
				Val: interp.DecimalVal(decimal.NewFromInt(int64(arg.Val.Int)))}
		case interp.ValBool:
			v := int64(0)
			if arg.Val.Bool {
				v = 1
			}
			return &interp.Expr{Kind: interp.ExprLiteral,
				Val: interp.DecimalVal(decimal.NewFromInt(v))}
		case interp.ValString:
			d, err := decimal.NewFromString(arg.Val.Str)
			if err != nil {
				c.errorf(n, "Decimal(%q): invalid decimal literal", arg.Val.Str)
				return nil
			}
			return &interp.Expr{Kind: interp.ExprLiteral, Val: interp.DecimalVal(d)}
		default:
			// Decimal(None)/other non-convertible literals: Python raises
			// TypeError — fail closed at compile time.
			c.errorf(n, "Decimal(): argument cannot be converted to decimal")
			return nil
		}
	}
	// PY-DECIMAL-CTOR-1: non-literal → runtime conversion. Residual vs
	// Python: unparseable string → 0 (Python raises); bool → 0 (Python=1).
	return &interp.Expr{Kind: interp.ExprCall, Name: "StringToDouble", Args: args}
```

注意：`compilePyCall` 的 receiver 有 `c.errorf(n, ...)` 惯例（:324/:335 先例）；`n` 节点变量在 case 作用域可用。`decimal` 包已 import（:27 使用）。

### S2 新测试 `compile_py_decimal_test.go`

运行时断言经 `CompilePython(source)`→`vmRunner.vm.RunOnBar`→`vm.GetGlobal(name)`（`self.x`→global "x"，`compile_py_locals_test.go:389` 先例）：

- `TestCompilePython_DecimalCtorStringLiteral`：`self.r = Decimal("0.1")` → `GetGlobal("r").Kind==ValDecimal` 且 `ToDecimal()==0.1`（**类型+值双断言**——无折叠时 Kind=ValString RED）。
- `TestCompilePython_DecimalCtorBool`：`self.r = bool(Decimal("0"))` → false（现状 true RED）。
- `TestCompilePython_DecimalCtorArithmetic`：`self.r = Decimal("1.5") + Decimal("1.5")` → 3。
- `TestCompilePython_DecimalCtorEquality`：`self.r = Decimal("0.0") == Decimal("0")` → true（现状 str 比较 false RED）。
- `TestCompilePython_DecimalCtorIntLiteral`：`Decimal(5)` → ValDecimal 5（非 IntVal——**Kind 断言判别透传**）。
- `TestCompilePython_DecimalCtorInvalidLiteral`：`Decimal("abc")` → `CompilePython` 返 error 含 "Decimal"（fail-closed）。
- `TestCompilePython_DecimalCtorNone`：`Decimal(None)` → 编译 error。
- `TestCompilePython_DecimalCtorNonLiteral`：`x = "2.5"; self.r = Decimal(x)` → 2.5（StringToDouble 路径）。
- `TestCompilePython_DecimalCtorZeroArg`：`Decimal()` → 0 ValDecimal（保留语义回归）。

### S3 全量回归

`go test ./tools/mql2go/`（py 测试集全量，IR 断言不受影响）。

## 3. mutation 验收项（独立复审执行）

1. `case "Decimal"` 还原 `return &args[0]` → StringLiteral/Bool/Arithmetic/Equality/IntLiteral 全 RED。
2. 删 ValString 折叠只留透传 → StringLiteral RED 且 InvalidLiteral 编译不再拒绝 RED。
3. 非字面量删 StringToDouble 改透传 → NonLiteral 测试 RED。

## 4. 验收门

```bash
cd backend && go build ./...
go test ./tools/mql2go/
go test -race -count=3 ./tools/mql2go/
go vet ./tools/mql2go/
gofmt -l tools/mql2go/compile_py_expr.go tools/mql2go/compile_py_decimal_test.go
go run ./tools/check-file-lines --strict
git diff --check
```

## 5. 边界/不做

- 不动 `int`/`float`/`str` 透传（同族另债，registry 备注）。
- 不新增 builtin/api_registry 条目。
- 不改 `StringToDouble` builtin 语义。
- 非字面量残留（garbage→0、bool→0）如实记录，不加掩盖。
- 零参/多参边缘：零参保留现状；`Decimal("0.1", ctx)` 多参保持 lenient（不动）。
- 勿部署、勿 push、禁 `--no-verify`。

## 6. 完成报告格式

`[施工完成:PY-DECIMAL-CTOR-1] @<commit-hash>` + 机检五件套 + 9 测试名逐一列出。勿部署，停手等 Devin CLI 复审。
