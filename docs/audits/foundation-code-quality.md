# 地基审计 · 代码质量

> 工具扫描 + 人工判断。不追求零告警，只追真实问题。

## 1. 文件大小

**3 个文件超硬性红线（🔴 阻断 CI）**：

| 文件 | 行数 | 超限 | 方案 |
|------|------|------|------|
| `backend/cmd/server/handlers.go` | 462 | +54% | 按 handler 域拆分（marketplace/strategy/admin/user） |
| `backend/internal/service/account_service.go` | 452 | +50% | 拆为 account_crud.go + account_lifecycle.go + account_sync.go |
| `frontend/src/pages/marketplace/components/AutoGeneratePanel.tsx` | 433 | +73% | ⚠️ 此文件尚不存在——GLM 施工中。完工后需拆分为 AutoGeneratePanel + AutoGenerateProgress + AutoGenerateResult |

**21 个文件超软性参考线（🟡）**：均低于硬性红线，暂不处理。

## 2. golangci-lint

263 个告警，15 个类别。按严重度和实际影响分类：

| 类别 | 数量 | 真实问题？ | 处理 |
|------|------|----------|------|
| `staticcheck` | 36 | 🟡 可能有 | 人工审查 |
| `unused` | 30 | 🔴 死代码 | 删除未使用的函数/变量/类型 |
| `unparam` | 36 | 🟡 可能有 | 未使用的函数参数——可能是接口要求 |
| `gocognit` | 39 | 🟢 忽略 | 复杂函数都合理（VM 执行循环、OMS 状态机） |
| `goconst` | 50 | 🟢 忽略 | 魔法字符串提取——非紧急 |
| `errcheck` | 20 | 🔴 需检查 | 未检查的 error 返回值 |
| `forbidigo` | 12 | 🟢 已知 | `fmt.Printf` 在 CLI 工具中（hdgen/coldsign） |
| `funlen` | 10 | 🟢 忽略 | 长函数在编译器/VM 中合理 |
| `gocyclo` | 5 | 🟢 忽略 | 圈复杂度高的函数都在编译器/VM |
| `gosec` | 5 | 🟡 需检查 | 安全相关——可能不是误报 |
| `ineffassign` | 8 | 🟡 需检查 | 赋值了但未使用的变量 |
| `misspell` | 1 | 🟢 修 | 拼写错误 |
| `noctx` | 1 | 🟡 需检查 | 未传递 context |
| `revive` | 7 | 🟢 忽略 | 代码风格 |
| `unconvert` | 3 | 🟢 修 | 不必要的类型转换 |

**真正需要处理的**：`unused`（30）+ `errcheck`（20）+ `ineffassign`（8）+ `gosec`（5）= **63 个告警需要人工审查和修复。**

## 3. 死代码

**copytrade 清理**：✅ 已完成。
- `internal/marketplace/copytrade.go` 已删除
- migration 212 `DROP TABLE copytrade_signals, copy_trade_links` 已存在
- 当前最新 migration 是 223，212 应已执行

**冗余 schema**：`copy_trade_links` 和 `copytrade_signals` 从 migration 212 drop。BUG-2 已修复。✅

## 4. 代码重复

`dupl` 扫描发现 marketplace 包内有重复逻辑：

| 位置 | 类型 | 说明 |
|------|------|------|
| `leaderboard.go:95-178` | 3 组克隆 | 四种榜单的查询逻辑高度相似——可提取公共 builder |
| `service_subscription.go:245-251` | 1 组克隆 | 重复的 wallet 操作——已通过 `AdjustBalanceTx` 统一 |

**leaderboard.go 的重复是真正的代码异味**——四种榜单（收益/人气/新锐/跟单）的查询结构相同、仅排序字段不同。应提取为 `buildLeaderboardQuery(type, period)` 函数。**< 30 行改动。**

## 5. Go 版本

go.mod 声明 `go 1.26`，但当前运行的 Go 版本是 `go1.25`。部分工具链检查失败。**需要升级 Go 到 1.26。**

## 汇总

| 类别 | 问题数 | 优先级 |
|------|--------|--------|
| 文件大小 > 红线 | 3 | 🔴 P0 — 拆 |
| golangci-lint: unused | 30 | 🔴 P0 — 删死代码 |
| golangci-lint: errcheck | 20 | 🔴 P0 — 加 error check |
| 代码重复 (leaderboard.go) | 1 | 🟡 P1 — 提取公共函数 |
| golangci-lint: gosec | 5 | 🟡 P1 — 安全审查 |
| golangci-lint: ineffassign | 8 | 🟢 P2 — 清理 |
| golangci-lint: unconvert | 3 | 🟢 P2 — 清理 |
| golangci-lint: misspell | 1 | 🟢 P2 — 修 |
| Go 版本 | 1 | 🟢 P2 — 升级 |
| golangci-lint: 其余 | 165 | 🟢 忽略 |

**可立即执行的**：
1. 拆分 3 个超红线文件
2. 删除 30 个 unused 函数/变量
3. 修复 20 个未检查 error
4. leaderboard.go 重复逻辑提取

这四项做完，代码质量达到基准线。
