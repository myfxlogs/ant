# 技术债务总账（Tech Debt Registry）

> **目的**：把全项目"以前记录过但可能没处理完"的债务**单一登记**，驱动后续逐条清理。
>
> **状态约定**：`🟦open` = 已核验仍存在；`✅done` = 已清；`❌descoped` = 取消。
>
> **关联**：本总账是 `memory/open-items-registry.md` 的详细展开。历史 ✅done 项已删除，靠 git 追溯。

---

## Open Items

| ID | 项 | 状态 |
|----|----|------|
| MQL-LOOP-4 | P2-T4/T5 扩展（useAIFix 扩到 coverage fatal + T5 实盘门控）| 🟦open（P2 暂缓）|
| LEAKAGE-2 | ~~跟单检测~~ | ❌ descoped（2026-08-08：技术不可行，MetaQuotes 无 API 检测提供方/订阅者）|
| POST-1 | 前端 UX 修复（UX-1~8 阻断级 + 🟡20 + 🟢16）| 🟦open（返工施工完成，待审计方实测验收）|
| POST-2 | 性能/容量压测（下单/回测/SSE）| 🟦open |
| FEAT-3 | 受保护回测对齐 | 🟦open（roadmap）|
| TUNING-OVERFIT-2 | OOS-at-publish 惰性闸（`quality.go:302` 条件性惰性，优化快照未填 OOS 字段）| 🟦open（低优 follow-up）|
| CQ-5 | eslint-disable 残留 11 处缺注释 | 🟦open（低优，补理由注释）|

---

## POST-1 UX 审计发现清单（2026-08-10 审计完成，修复待验收）

**🔴 阻断级（UX-1~8，返工施工完成 2026-08-11，待审计方实测验收）**：
- **UX-1** 衰减徽章从未渲染 → 已实现（DecayBadge 3处+购买disabled），✅ 返工：`share_handler.go:79-90` 加 `ORDER BY updated_at DESC LIMIT 1` + ErrNoRows→'none' + 错误日志（禁 `_ =` 吞）。T6/T7 对抗测试 ✅
- **UX-2** 实盘战绩接口失败静默 → 已实现（error态+Alert+重试），✅ T8 对抗测试 ✅
- **UX-3** 客户端筛选+服务端分页空页 → ✅ 返工：`publishedCacheEntry` 加 `total` 字段，`set` 存入 COUNT，命中路径 `return cached, entry.total, nil`。T1/T2 对抗测试 ✅
- **UX-4** 移动端回测结果不可见 → ✅ 返工：`BottomPanelSection` 移动端 Drawer 加 Segmented（Backtest|Positions），Backtest tab 渲染 `BacktestResultsTab`（复用，非复制）。T3 对抗测试 ✅
- **UX-5** AI Fix strategyId 空静默 → 已实现（禁用Apply+Alert"先保存"）
- **UX-6** 实盘SSE断流伪装无策略 → 已实现（保留旧数据+2s重连+Alert横幅）
- **UX-7** 4公开路由无ErrorBoundary → 已实现（全wrap()）
- **UX-8** build无类型检查 → ✅ 返工：`package.json` build=`tsc --noEmit -p tsconfig.app.json && vite build`，CI `npx tsc --noEmit -p tsconfig.app.json`。erasableSyntaxOnly 移除动因：恢复 flag → src/gen/ 11×TS1294 enum 错误（被 import 的 gen 不受 exclude 挡），故移除。T4/T5 对抗测试 ✅

**🟡 显著摩擦 20 条** + **🟢 轻微 16 条**：详见 git 历史 `tech-debt-registry.md@2026-08-10`。

---

## 总计

零 ❓待核。🟦open 6 项 + ❌descoped 1 项。⚠️待Claude复审：无。
POST-1 返工施工完成（4 缺陷 + 8 对抗测试），待审计方实测验收。
上线就绪：所有 launch-blocking 缺口审计方实测清零（2026-08-09）。

---

## 变更日志

- 2026-08-11 **POST-1 UX-1~8 返工施工**：4 缺陷修复（UX-3 缓存 total=-1 / UX-4 移动端回测面板 / UX-8 tsc 真门禁 / UX-1 查询确定性）+ 8 项对抗测试（T1-T8）。门禁全绿：go build + check-file-lines 0err + tsc + vitest 146pass + build。待审计方实测验收。
- 2026-08-11 **Part D 验收 + UX-1~8 复审**：Part D（runbook 12实写+CQ-2 knip 0issue+CQ-9 前端收尾）审计方实测 ✅。UX-1~8 复审：TS清零✅实测，UX-3 缓存total=-1/UX-4 修错面板/UX-8 CI空操作 3缺陷打回，8项对抗测试全缺，维持🟦open。
- 2026-08-10 **FILL-SIM 验收通过 ✅**：Phase A-E 全部完成，2阻塞级缺口补强后审计方独立复测通过，⚠️解除。FILL-SIM 闭环。
- 2026-08-10 **FE-TRUST-1 审计方实测验收 ✅**：分享页零信任迁移+后端回撤bug修复，Claude复审通过。
- 2026-08-10 **EXEC-PARAMS 验收通过 ✅**：回测执行假设参数端到端接线+核心bug修复，审计方实测通过。
- 2026-08-10 **POST-5 agent重构收尾 ✅**：plan驱动+语义追问全落地，agent重构里程碑完成。
