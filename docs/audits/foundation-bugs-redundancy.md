# 地基审计 · BUG 与冗余

> 已知 BUG 清单、冗余 schema、废弃代码、迁移完整性。

## 1. 已知 BUG

**git log 显示 GLM 正在积极修 bug**——最近 10 个 fix commit 覆盖了 marketplace 架构、OG image、lint、ci 等。没有积压的已知 BUG。

**0 个真实 TODO/FIXME**：Go 代码中无遗留 TODO。prompt 模板里的是给 AI 的指令，不是代码标注。

**结论**：✅ 没有已知遗留 BUG。

## 2. 冗余 Schema

| 表 | 状态 |
|----|------|
| `copy_trade_links` | ✅ migration 212 已 drop |
| `copytrade_signals` | ✅ migration 212 已 drop |
| `trading_accounts` | ⚠️ migrations 中未找到定义，Go 代码中 0 引用。可能是早期表名已被 `mt_accounts` 替代？需人工确认 |

**结论**：`trading_accounts` 需确认状态——如果在生产 DB 中存在但 Go 代码未引用，属于冗余。

## 3. 迁移完整性

| 指标 | 值 |
|------|-----|
| up migrations | 195 |
| down migrations | 140 |
| 缺 down | 55 |

**55 个迁移没有回滚脚本。** 这不是 bug——这是设计选择。大部分迁移是 schema 新增（加表、加列、加索引），不需要回滚。但需要回滚的迁移（DROP TABLE、DELETE、数据迁移）必须有 down。

**需检查的**：`migration 212`（DROP copytrade 表）有 down.sql——正确。

**结论**：🟡 55 个缺 down 的迁移，需要确认其中是否有破坏性操作（DROP/DELETE）缺少回滚脚本。

## 4. 废弃 Proto 字段

**结论**：✅ 未发现 deprecated 标注。Proto 定义干净。

## 5. 冗余代码

| 项 | 状态 |
|----|------|
| `copy_trade_links` 表（BUG-2） | ✅ 已修复 |
| copytrade.go | ✅ 已删除 |
| golangci-lint `unused` 30 项 | 🔴 待处理（见层3审计） |

## 6. 汇总

| 类别 | 问题 | 严重度 |
|------|------|--------|
| 已知 BUG | 0 | ✅ |
| 冗余 schema | `trading_accounts` 待确认 | 🟡 |
| 迁移完整性 | 55 个缺 down | 🟡 |
| 废弃 proto | 0 | ✅ |
| 冗余代码 | 30 unused（见层3） | 🔴 |
