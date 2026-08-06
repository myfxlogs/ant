# 施工 Spec：xianhua 暴露的剩余修复（前端传参 + 永久防线呈现 + 端到端测试）

> xianhua 案例的根治（findIdent/findType/findInitValue）已完成，volume=0 解决。但暴露三个剩余问题，本 spec 合并处理。**分三部分施工，可独立提交**。

## 背景（教训，已写入 ADR-0028）

- tree-sitter 浮点 default quirk 影响三个函数（findIdent/findType/findInitValue），逐个半修才解决——教训：修一个 quirk 要一次查全所有遍历该结构的函数。
- 永久防线（防线 B）数据层工作（PG 有 zero_volume_trade），但**呈现层断**（status SUCCEEDED、前端没展示 BlindSpot）→ 用户没被保护。**永久防线 = 检测 + 呈现，缺呈现等于没做。**
- **端到端测试缺失**：从没测过"用户填参数 → 回测实际用这些值"，导致参数被忽略的 bug 一直没被抓。

---

## 第一部分：前端传参修复（让用户填的 lots/杠杆生效）

**症状**：弹窗填 lots=0.01 + 杠杆=2000，但回测没用这些值（PG：`parameter_overrides` 的 Lots **空值**；`backtest_runs.leverage=1` 默认；config_snapshot 无 2000）。

**线索**（已定位）：
- `frontend/src/components/backtest/useBacktestRunner.ts:191-196` 传了 `parameterOverrides: strategyParamValues` + `executionConfig: { leverage, ... }`——前端**有传**。
- 但 PG 没收到正确值 → **state 绑定或序列化环节丢**：弹窗填的 lots/杠杆没正确进 `strategyParamValues` / `leverage` state，或 proto 序列化丢值。

**任务**（GLM 在前端定位修复）：
1. 追参数弹窗组件（BacktestParamsModal 或参数面板）的 onChange → `strategyParamValues` state：确认用户填的 lots 值进 state（不空）。
2. 追 leverage state：确认弹窗填的 2000 进 `leverage` state。
3. 追 `parameterOverrides` / `executionConfig` 的 proto 序列化（`client/strategyRuntime.ts` startBacktestRun）：确认 state 值正确序列化进 proto（Lots key 有值、leverage 非 0）。
4. 修复断点（state 绑定 or 序列化）。

**验收**：弹窗填 lots=0.01 + 杠杆=2000 → 回测 PG 的 `parameter_overrides` Lots=0.01、`backtest_runs.leverage=2000`、回测 result 的 trades volume=0.01。

---

## 第二部分：永久防线呈现层（status 降级 + 前端 BlindSpot 醒目）

> 详见 `docs/spec/p0-defense-presentation-spec.md`。此处重申要点（让 GLM 一并做）。

**后端**：
1. `status_constants.go` 加 `StatusDegraded = "DEGRADED"`（执行成功但防线 B 判定不可信）。
2. `backtest_persistence.go saveBacktestResult`：`result.BlindSpots` 含 invariant 类（zero_volume_trade / capital_not_conserved / non_positive_price / invalid_side / time_order_violation）→ status=`StatusDegraded`（而非 SUCCEEDED）。加 helper `hasInvariantBlindSpot`。

**前端**：
3. 回测结果页：status=`DEGRADED` 醒目展示（红色/警告 + "结果不可信"），不显示成成功。
4. BlindSpot（invariant 类）醒目列出（每条 Id + Description），让用户看到"为什么不可信"。
5. i18n：DEGRADED 文案 + invariant BlindSpot 描述（中英日越繁中）。

**验收**：有 invariant BlindSpot 的回测 → status=DEGRADED（不再 SUCCEEDED）+ 前端醒目展示原因。

---

## 第三部分：端到端测试（参数链 + 防线呈现守护）

> 这是最关键的——之前所有 bug（参数被忽略、volume=0）都是端到端缺失导致没被抓。

**任务**：
1. **参数链端到端测试**（后端集成测试）：构造 StartBacktestRun 请求（含 parameterOverrides Lots=0.01 + executionConfig leverage=2000）→ 跑回测 → 断言 result.Trades volume=0.01、cfg.Leverage=2000。**用户填的参数 = 回测用的参数**。
2. **防线呈现端到端测试**：构造 volume=0 的回测（故意触发）→ 断言 status=DEGRADED + BlindSpot 含 zero_volume_trade。**防线触发 = status 降级 + 呈现**。

**验收**：这两个端到端测试存在且 PASS——它们是"参数被忽略"和"防线哑铃"两类 bug 的守护。

---

## 约束

- 三部分可独立施工 + 独立提交（前端传参 / 呈现 / 端到端）。
- 最小改动、不重构无关、不硬编码。
- 后端 `go build ./...` + `go test ./internal/connect/strategy/` 全绿；前端 `npm run build` 过。
- 对抗验收（不预先公布）。

## 优先级（我定）

1. **第一部分（前端传参）**——让你能正常用（填的 lots/杠杆生效）。最紧急。
2. **第三部分（端到端测试）**——防这类 bug 再现。和第一部分一起（测第一部分的修复）。
3. **第二部分（呈现层）**——让永久防线真保护你。重要但不阻塞"能用"。
