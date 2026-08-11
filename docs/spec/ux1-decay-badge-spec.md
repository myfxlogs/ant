# UX-1 Fix Spec · 衰减徽章买家可见（产品核心）

> **审计方（Claude Code）出 spec；施工方（Windsurf）实现 + 回填。**
> **依据**：registry POST-1 UX-1（2026-08-10 审计）：proto 有 `decay_status`（`marketplace_service.proto:219`，`none|decaying|decayed`）+ 5 语言 i18n key 齐全，但买家可见面（市场卡片/详情/分享页）零消费——衰减检测只存在于作者专属 OptimizationTab。**产品信任护城河的核心元素对买家不可见**。
> **Scope 铁律**：只做本 spec。不做其他清理/重构/优化。

---

## 背景与问题

- FEAT-5 已实现：`decay_monitor.go` 检测策略实盘表现衰减 → `marketplace_strategies.decay_status`（`none|decaying|decayed`）+ 停购门（`purchase.go:223` decayed → "no longer available"）。
- **买家侧断裂**：买家在购买前看不到衰减状态。买了才在作者 OptimizationTab 看到（且那是作者视角）。停购门把"已衰减"策略藏起来了，买家只看到"买不了"没有解释。
- **产品意义**：衰减透明 = 买家保护 = trust 护城河的前端呈现（FEAT-5 决策 ④「买家保护靠信息透明 = 验证战绩+衰减徽章+停止新购」——徽章就是信息透明的实体）。

## 现状核实（2026-08-11 审计方）

| 项 | 现状 |
|---|---|
| DB 字段 | `marketplace_strategies.decay_status`（decay_monitor.go:174 更新） |
| proto | `PublishedStrategy.decay_status = 27`（marketplace_service.proto:219）✅ 已生成 |
| 前端数据 | gen `PublishedStrategy.decayStatus` 存在（grep gen 命中）✅ |
| 前端消费 | **零**（`grep decayStatus src/` 除 gen 零命中；仅 OptimizationTab 用 decay_metrics 走作者线）❌ |
| i18n | 5 语言 key 已存在（registry 2026-08-10 核实"齐全"）——施工方开工前再 `grep decay i18n/resources/*/marketplace*.ts` 确认 key 名 |
| 分享页 | `share.proto` **无** decay 字段；`share_service` 不返回 → 分享页无法显示 ❌（需后端补） |

## 修复设计

### 任务 1（后端）：分享页数据链路补 decay_status

- `proto/ant/v1/share.proto`：`ShareData`（或 share 响应主 message）加 `string decay_status = <下一个可用字段号>`（注释 `none | decaying | decayed`）。改完跑 `buf generate`。
- `backend/internal/connect/user/share_service.go`（或 share 查询所在文件）：share 数据查询 JOIN/补查 `marketplace_strategies.decay_status`（`COALESCE(...,'none')`），填充 proto。**核实现有 share 查询从哪拿策略信息**——share 链接有 `strategy_id`，补 `LEFT JOIN marketplace_strategies ms ON ms.strategy_id = ...` 取 `decay_status`。
- 分享页分享的是**历史战绩快照**（不可篡改），衰减状态是**当下**的——查询时 JOIN 实时值即可（当前状态），不需要存快照（衰减是现状信息不是历史战绩，两者语义不同，勿混淆）。

### 任务 2（前端）：三处买家可见面渲染徽章

抽共享小组件（`pages/marketplace/components/DecayBadge.tsx`，REUSE 检查：如已有同类组件直接复用）：

- `StrategyMarketCard`：卡片右上角/标题旁渲染 Tag（decaying=橙、decayed=红、none=不渲染）。
- `StrategyDetailModal`：标题旁徽章 + **解释文案**（i18n："平台检测到该策略实盘表现持续下降，已停止新购买"；decaying 用"表现下降，正在监控"文案）。
- `SharePerformancePage`（分享页）：`data.decayStatus` 非 none 时渲染徽章 + 文案（公开面，买家最先看到的是分享页）。
- 停购门一致性：decayed 策略在购买按钮处给解释（不是冷 error "no longer available"）——若购买按钮已 disabled/隐藏则补 tooltip，若仍可点则给明确提示。

### 任务 3：i18n

- 核实现有 decay key（5 语言），缺的补 `marketplace.decay.*`：`badgeDecaying` / `badgeDecayed` / `descDecaying` / `descDecayed`（或复用现有 key 名——以实际为准）。

## 验收标准（审计方实测）

- 后端：share 响应含 decay_status；`go build ./...` + 相关 test 绿；`check-file-lines --strict` 0🔴。
- 前端：`tsc -p tsconfig.app.json --noEmit` 0 error + `npm run build` + `vitest run` 全绿。
- 对抗证明（必带）：
  1. **后端**：unit test——decayed 策略 share 响应 `decay_status == "decayed"`；删 JOIN/赋值 → 必红。
  2. **前端组件测试**：3 处渲染组件（DecayBadge）——`decayed` prop → 徽章渲染 + 文案存在；`none` → 不渲染。删渲染分支 → 必红。
  3. **grep 对抗**：修后 `grep -c "decayStatus" frontend/src --include="*.tsx" --include="*.ts" -r`（除 gen）从 0 → ≥3。

## 红队自审（施工方必过）

- [ ] decayed 策略**购买入口**是否真的不可达/有解释？（停购门已堵购买，前端入口提示一致性——否则买家点"购买"收到莫名 error）
- [ ] 分享页快照 vs 实时衰减状态语义区分——衰减状态**必须是查询时实时值**，不是历史快照
- [ ] share proto 字段号不冲突（看现有最大字段号 +1）
- [ ] i18n key 不重复（先 grep 现有，能复用就复用）
- [ ] 徽章在卡片/详情/分享页的**视觉一致性**（同一组件/同一配色）
- [ ] 移动端不破版（Tag 在窄屏不溢出）

## 完工回填纪律

1. registry POST-1 下 UX-1 条目：🟦→✅（标日期 + 根因/修复/对抗证明/测试结果）
2. handover 变更日志一行
3. **不自行宣告完成**——等审计方核对状态 + 实测
