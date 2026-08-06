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

**修复**（待施工）：
- 后端：`StatusDegraded` 常量 + `saveBacktestResult` 检查 invariant BlindSpot → status=DEGRADED
- 前端：DEGRADED 醒目展示 + BlindSpot 列出原因

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

**修复方向**（待施工）：`run()` 不应从闭包 state 读参数（异步陷阱），而应从调用者**直接传入**的参数读。`onConfirm` 有 `result.strategyParams` + `result.params`（同步值），直接传给 `run`。

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
                    → VMRunner injectParams（用 Type 分发注入）← 坑 1（Type="" 跳过）
                      → OrderSend volume = Lots slot 值
                        → SimBroker 撮合
                          → result.Trades
                            → 防线 B 检测 ← 坑 2（呈现断）
                              → 用户看到结果
```

**每个箭头都是一个可能断的环节。这次的 6 个坑分布在 5 个环节上。端到端测试是唯一的系统性防护。**
