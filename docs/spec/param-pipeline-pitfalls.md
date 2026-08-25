# MQL 参数管线踩坑记录（xianhua 事故完整根因链）

> 本文档记录从 xianhua.chan 手数=0 事故中发现的**所有根因和踩过的坑**，供后续团队和 AI 学习。每个坑都有：症状、根因、修复 commit、教训。

---

## 坑 1：tree-sitter 浮点字面量 grammar quirk（影响 3 个函数）

**症状**：含浮点默认值的 input 参数（如 `input double Lots=0.1`），参数名/类型/默认值全部提取错。

**根因**：tree-sitter MQL grammar 对浮点字面量（如 `0.1`）的解析结构与整数不同。整数行 `TakeProfit=50` 的 AST：
```
declaration
  type_identifier = "input"
  ERROR = "double"          ← double 在 ERROR 节点（位置正常）
  init_declarator = "TakeProfit=50"
    identifier = "TakeProfit"
    number_literal = "50"
```
浮点行 `Lots=0.1` 的 AST：
```
declaration
  type_identifier = "input"
  init_declarator = "double Lots=0.1"   ← 整个 "double Lots=0.1" 被当成 init_declarator！
    identifier = "double"               ← double 变成 init_declarator 的第一个 identifier
    ERROR = "Lots"
    number_literal = "0.1"
```

这导致**三个遍历 init_declarator 的函数全部受影响**：

| 函数 | 应提取 | 实际提取（bug） | 修复方式 | commit |
|---|---|---|---|---|
| `findIdent` | Name="Lots" | Name="double"（取第一个 identifier） | 跳过 primitive type identifier + 递归 ERROR | 4050ab68 |
| `findType` | Type="double" | Type=""（找不到 primitive_type） | 递归找 primitive type identifier | 85673649 |
| `findInitValue` | Default=0.1 | Default="double"（identifier，非 number） | 跳过 primitive type identifier | e75e02a0 |

**教训**：tree-sitter grammar quirk 影响**所有遍历该节点结构的函数**，不只一个。修一个 quirk 必须**一次查全所有受影响的 find* 函数**（findIdent/findType/findInitValue/findArraySize...）。逐个半修 = 每次以为修完，用户反复踩坑。

---

## 坑 2：永久防线呈现层断（status 不降级 + 前端不展示 BlindSpot）

**症状**：防线 B 检测到手数=0（PG 有 `zero_volume_trade` BlindSpot + IsReliable=false），但 status 仍 SUCCEEDED，前端不展示 BlindSpot——用户看到"回测成功"，被错误结果骗。

**根因**：永久防线只实现了"检测+标记"（IsReliable/BlindSpot），没实现"阻止冒充成功"（status 降级 + 前端醒目展示）。**检测 ≠ 保护**——检测到了但用户看不到 = 没保护。

**修复**（✅done — 2026-08-06/07 落地，2026-08-25 每周对账核验；本节原标"待施工"已过期）：
- 后端：`StatusDegraded` 常量（`internal/connect/strategy/status_constants.go`）+ `saveBacktestResult` 检查 invariant BlindSpot → status=DEGRADED（`backtest_persistence.go:69-99`，`zero_volume_trade` 登记于 `invariantBlindSpotIDs`）
- 前端：DEGRADED 醒目展示 + BlindSpot 列出原因（`BacktestResultsTab.tsx:107,170` / `DiagnosticPanel.tsx:145` / SSE `backtestRunnerWatch.ts:55-66`）
- commit `d8256a90`（坑2 修复）+ `58273444`（BT-5 DEGRADED 状态推送断链）；e2e `e2e_defense_presentation_test.go`

**教训**：永久防线 = **检测 + 标记 + 阻止冒充成功**，三层缺一不可。只做前两层等于没做。详见 ADR-0028 §5.1。

---

## 坑 3：React setState 异步 + useCallback 闭包旧 state（参数传不进回测）

**症状**：弹窗填 lots=0.01 + 杠杆=2000，但回测 PG 收到 parameterOverrides 只有 key 没 value、leverage=默认 1。

**根因**：`WorkspaceDrawers.tsx` 的 `onConfirm`：
```tsx
// 循环 setParam（React 异步 state 更新）
for (const [name, value] of Object.entries(result.strategyParams)) {
  backtest.runner.setParam(name, value);  // setStrategyParamValues（异步）
}
backtest.run();  // 立即调用（useCallback 闭包捕获旧 state！）
```

React `setState` 是**异步**的——循环 `setParam` 后立即 `backtest.run()`，此时 React 还没 re-render，**`run` 的 `useCallback` 闭包捕获的还是旧的 `strategyParamValues`（空对象）**。leverage/commission 同理。

**修复**（✅done — 2026-08-25 每周对账核验；本节原标"待施工"已过期）：`run()` 不再从闭包 state 读参数，改为接收调用者**直接传入**的 `overrides`——`run(overrides?)` 用 `overrides?.params ?? strategyParamValues`（`useBacktestRunner.ts:165-190`）；`onConfirm` 把 `result.strategyParams` + `result.params`（同步值）直接传给 `run`（`WorkspaceDrawers.tsx:51-66`）。

**教训**：React `setState` + 立即调用依赖该 state 的 `useCallback` = **闭包旧 state 陷阱**。参数链必须同步传递，不经过异步 state 中转。

---

## 坑 4：executionConfig proto 序列化（杠杆/手续费不传）

**症状**：前端传了 executionConfig（含 leverage），但后端 PG 收到的 config_snapshot 只有 0.001，leverage=默认 1。

**根因**：`strategyRuntime.ts` 之前用**直接对象字面量**构造 executionConfig：
```ts
executionConfig: { commission: ..., leverage: ..., ... }  // 不是 proto message
```
没调用 `create(BacktestExecutionConfigSchema, {...})`，ConnectRPC 客户端不认这个对象——proto 字段没序列化，后端收到空/默认。

**修复**：改用 `create(BacktestExecutionConfigSchema, { commission: String(...), leverage: String(...), ... })` + `String()` 确保字符串类型。commit 6977d5fb。

**教训**：ConnectRPC/protobuf 的 message 必须用 `create(Schema, {...})` 构造，不能用裸对象字面量。数值字段必须是 string（proto 定义）。

---

## 坑 5：参数提取只修 compile 路径，漏了 AnalyzeImportCode 路径

**症状**：findIdent/findType/findInitValue 都修了（compile 路径 CompileToIR+CompileAST），但前端弹窗的参数定义还 Lots→double。

**根因**：`AnalyzeImportCode`（strategy_import_handler.go:46）走 `CompileToIR + CompileAST`（含修复后的 find*），但**前端缓存了导入时的旧参数定义**（findType 修之前的），没刷新。

**修复**：用户重新分析/导入 EA 即可刷新（AnalyzeImportCode 用修后的 find*）。或前端每次回测前重新调 validate（拿最新参数定义）。

**教训**：多条路径提取同一信息（compile / analyze / validate），修一处必须确认所有路径都走修复后的代码 + 缓存失效机制。

---

## 坑 6：bytecode_cache（本案例未触发，但曾怀疑）

**xianhua 案例的 has_sid=false（inline code），不走 bytecode_cache**。但曾怀疑缓存了旧 bytecode。bytecode_cache 有 `CompilerVersion` 版本戳（2026-07-02-v1），如果 compiler 逻辑变了但没 bump 版本，旧缓存仍有效——产出错误结果。

**教训**：每次改 compiler 逻辑（findIdent/findType/findInitValue/任何影响 bytecode 的改动），**必须 bump CompilerVersion**（bytecode_cache.go:38），否则旧缓存不会失效。

---

## 总结：参数管线完整链路

```
MQL 源码
  → tree-sitter 解析（CST）
    → findIdent 提取参数名 ← 坑 1（浮点 quirk）
    → findType 提取参数类型 ← 坑 1
    → findInitValue 提取默认值 ← 坑 1
  → AnalyzeImportCode / Validate（给前端参数定义）← 坑 5（缓存）
    → 前端弹窗展示参数（名+类型+默认值）
      → 用户修改参数值
        → onConfirm 传回 ← 坑 3（React 异步 state + 闭包旧值）
          → strategyRuntime.ts startBacktestRun ← 坑 4（proto 序列化）
            → 后端 StartBacktestRun handler
              → backtest_runs.parameter_overrides + config_snapshot（PG）
                → worker extractBacktestParams
                  → cfg.Params + cfg.Leverage
                    → VMRunner injectParams（用 Type 分发注入）← 坑 1 + 坑 7
                      → OrderSend volume = Lots slot 值 ← 坑 8（resolveVar 静默建 =0）
                        → SimBroker 撮合
                          → result.Trades
                            → 防线 B 检测 ← 坑 2（呈现断）
                              → 用户看到结果
```

**每个箭头都是一个可能断的环节。这次的 10 个坑分布在 7 个环节上。端到端测试是唯一的系统性防护。**

---

## 坑 7：injectParams switch Type 静默跳过（后端注入零容错）

**症状**：参数 Type 提取错（如 Type=""），injectParams 的 `switch p.Type` 不匹配任何 case → **静默跳过该参数，不注入、不报错、不警告**。Lots 没注入 → slot 保持 initGlobals 初始值（0）→ OrderSend volume=0。

**根因**：`interp_runner.go:329` 的 `switch p.Type` 没有 `default` 分支——Type 不匹配（""/未知值）时什么也不做。**注入逻辑对提取错误零容错**：提取错了，注入静默丢弃，用户毫不知情。

**位置**：`backend/tools/mql2go/interp_runner.go:329-359`（injectParams 函数）

**教训**：注入逻辑**不能静默跳过**。如果参数有 Name + GlobalSlot 但 Type 不匹配，应该：① 尝试从 Default 推断类型（有 Default=0.1 → 按 double 注入）；② 或记 warning/blind spot；③ 最差也要 panic（比静默=0 好）。**静默=0 是最坏的行为——让用户以为注入了，实际没注入。**

---

## 坑 8：resolveVar 静默建 =0 全局（编译器吞错）

**症状**：Lots 参数名提取错（→"double"），代码里引用 "Lots" 在 GlobalSlots 里找不到 → resolveVar **静默注册一个 =0 的新全局变量** → Lots=0。

**根因**：`compile.go:191 resolveVar` 的逻辑：
```
查 local scope → 没找到
查 GlobalSlots → 没找到（因为注册成了 "double" 不是 "Lots"）
查 MQL constant/series → 不是
查 enum → 不是
→ MQL4/Python：record blind spot + 静默建 =0 新全局
```

注释说"MQL4 allows implicit variable declaration"——MQL4 确实允许隐式变量，但 **Lots 不是隐式声明，是编译器提取错误导致的引用脱节**。resolveVar 无法区分"合法隐式声明"和"编译器 bug 导致的脱节"，一律静默建 =0。

**位置**：`backend/tools/mql2go/compile.go:191-231`（resolveVar 函数）

**教训**：编译器对"未定义引用"的处理，**MQL4 的隐式声明语义**让它无法简单报错（合法 EA 可能用隐式变量）。但可以**增强诊断**：如果未定义引用的名字恰好是某个 input/extern 参数名（但因为提取错误没注册），应该给更显著的警告（不是普通 blind spot，而是"参数名提取可能有问题"）。这是防线 A 的范畴。

---

## 坑 9：编译器对 tree-sitter ERROR 节点处理不完整

**症状**：tree-sitter 把浮点 default 行的 "Lots" 标识符放进 ERROR 节点（语法恢复）。编译器的 findIdent 等函数之前**不递归 ERROR 节点**，导致 ERROR 里的标识符（Lots）提取不到。

**根因**：tree-sitter 在语法不确定时把部分内容放进 ERROR 节点（错误恢复）。编译器的遍历函数（findIdent/findType/findInitValue）只遍历 NamedChildren，不递归 ERROR。这导致 ERROR 里的标识符被遗漏。

**位置**：`backend/tools/mql2go/compile_interp_expr.go`（findIdent/findType/findInitValue）

**教训**：编译器**必须处理 tree-sitter 的 ERROR 节点**——ERROR 不是"垃圾"，是 tree-sitter 对不完整语法的最佳猜测，里面可能有有效标识符。所有遍历函数都应该递归进 ERROR 找标识符（findIdent 修复时加了，findType/findInitValue 同理）。

---

## 坑 10：参数链端到端测试缺失（系统性根因）

**症状**：以上所有坑（1/7/8/9）都靠**用户回测踩坑**发现的，没有任何测试提前抓到。

**根因**：参数管线（提取→注入→OrderSend→结果）**没有端到端测试**。现有测试都是单元级的（测 findIdent/findType/injectParams 各自），没有"编译 EA → 注入参数 → 跑回测 → 断言 result.Trades.volume = 传入值"的端到端测试。

**教训**：参数链是多环节链路（tree-sitter→提取→注入→VM→撮合），**单元测试只能验证各环节自身正确，不能验证环节间的衔接**。只有端到端测试（从 MQL 源码 + 用户参数 → 回测 result 里的实际值）才能抓衔接断点。这是所有坑的共同系统性根因——**缺乏端到端测试 = 靠用户踩坑发现 bug**。

**每个箭头都是一个可能断的环节。这次的 10 个坑分布在 7 个环节上。端到端测试是唯一的系统性防护。**

> **坑10 状态更新（2026-08-25 每周对账）**：本坑的系统性根因"缺乏端到端测试"**已闭合**——commit `30668f64`（2026-08-06）新增 `backend/tools/mql2go/e2e_param_pipeline_test.go`（MQL 源码 → CompileMQL → cfg.Params → engine.Run → 断言实际成交手数 0.5 / 默认 0.1）+ `e2e_defense_presentation_test.go`。**残留软肋**：两个参数链测试在 0 成交时走 `t.Skip`（软断言），环境无成交时会 SKIP 而非 FAIL，建议后续改为硬失败。
