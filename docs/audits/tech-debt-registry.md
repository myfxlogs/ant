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
| POST-1 | 前端 UX 修复（UX-1~8 阻断级 + 🟡20 + 🟢16）| ✅done（2026-08-11 审计方独立删行复测 5/5 全红）|
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
- **🆕 2026-08-11 审计方复审（af565aa7 返工）**：4 缺陷实现 ✅ 验收通过（代码级核对 + 全门禁实测绿：tsc 0err/vitest 146/go build/go test/check-file-lines 0err/npm build）。**对抗证明 5/8 无效（审计方独立删行实测）**：T1 缓存命中复现 `-1` 仍绿——`TestPublishedCache_HitReturnsTotal` 只测 set/get 单元，不调 `ListPublished` 主路径；T6/T7 删 handler `ORDER BY updated_at DESC LIMIT 1` 仍绿——测试断言测试文件内字符串字面量，不触 `share_handler.go`；T3 删 `backtestContent` 接线仍绿——渲染手写 div，从不渲染 `BottomPanelSection`；T8 同模式（手写 Alert，不渲染 LivePerformanceTab）。有效：T2（`buildPublishedCountQuery` SQL 断言，属上轮 990aa947 功能非本返工）、T4/T5（package.json 契约）。**裁决：实现验收 ✅；对抗证明不达标 = 未完成**（铁律：删了还绿=测试无效）。待补：T1 走 ListPublished 集成（缓存命中 total 真值）/ T3+T8 渲染真实组件 / T6+T7 抽可测函数。
- **🆕 2026-08-11 测试补强（施工方删行自测）**：5 项重做完成，每项删行实测必红：
  - **T1** `TestListPublished_CacheHitReturnsTotal`：预置缓存 → 调 `ListPublished`（nil pg，缓存命中避 DB）→ 断言 total=42。删 `return e.data, e.total, true` → 改 `-1` → **实测红**（total=-1, want 42）。
  - **T6** `TestBuildShareDecayStatusQuery_HasOrderByAndLimit`：调真函数 `buildShareDecayStatusQuery()`。删 `ORDER BY`/`LIMIT 1` → **实测红**（missing ORDER BY + missing LIMIT 1）。
  - **T7** `TestResolveDecayStatus_ErrNoRows`：调真函数 `resolveDecayStatus()`，用 `zaptest/observer` 验证 ErrNoRows **不产生日志**（区别于其他错误的 log+fallback）。删 `ErrNoRows` 分支 → ErrNoRows 落入 log 路径 → **实测红**（logged 1 entries, expected 0）。T7b 验证其他错误**产生 1 条日志**。T7c 验证 nil error 返回 scanned 值。
  - **T3** 渲染真实 `BottomPanelSection`（isMobile=true）→ 点击开 Drawer → 断言 `getByTestId('test-backtest-content')` 可见。删 `{mobileTab === 'backtest' && backtestContent}` → **实测红**（getByTestId throws）。
  - **T8** 渲染真实 `LivePerformanceTab`（mock `marketplaceClient.getLivePerformance` reject）→ 断言 `.ant-alert-error` + 重试按钮存在 → 点击重试验证再次调用。删 error 渲染块 → **实测红**（querySelector returns null）。
  - **实现改动**：仅 T6/T7 抽函数（`buildShareDecayStatusQuery` + `resolveDecayStatus`，行为不变），其余禁改实现。
  - **门禁全绿**：go build + check-file-lines 0err + tsc 0err + vitest 144pass + npm build。
- **🆕 2026-08-11 审计方独立删行复测（验收 ✅ 权威 done）**：不信任声明，逐项独立删行实测：
  - **T1** 改 `return cached, cachedTotal, nil` → `-1` → **断言红**（total=-1, want 42，走 `ListPublished` 主路径缓存命中）
  - **T6** 删 `ORDER BY updated_at DESC LIMIT 1` → **断言红**（missing ORDER BY + missing LIMIT 1）
  - **T7** 删 `resolveDecayStatus` ErrNoRows 分支 → **断言红**（logged 1, expected 0，ErrNoRows 落 log 路径）；T7b/T7c 对照保持绿
  - **T3** 删 `{mobileTab === 'backtest' && backtestContent}` → **断言红**（getByTestId 抛错，真实渲染 Drawer+Segmented）
  - **T8** 删 error 渲染块 → **断言红**（`.ant-alert-error` null，落 Empty 分支）
  - **裁决：5/5 断言级全红 → T1-T8 对抗证明 8/8 有效。全量回归绿实测**：go build / go test marketplace+user / check-file-lines 0err / tsc 0err / vitest 144pass / npm build。实验编辑全部还原，工作树干净。**POST-1 闭环 ✅**。

**🟡 显著摩擦 20 条** + **🟢 轻微 16 条**：详见 git 历史 `tech-debt-registry.md@2026-08-10`。

---

## 总计

零 ❓待核。🟦open 5 项 + ❌descoped 1 项。⚠️待Claude复审：无。
POST-1 ✅done（2026-08-11 审计方独立删行复测 5/5 全红验收，8/8 对抗测试有效）。
上线就绪：所有 launch-blocking 缺口审计方实测清零（2026-08-09）。

---

## 变更日志

- 2026-08-12 **REPLAY-MODEL ✅**：EXEC-PARAMS 后续简化 — 4 执行假设选择器合并为单"复盘模型"下拉框（MT4 对齐：Every Tick / 1 Minute OHLC / Open Prices Only）。前端 only，后端参数不变（replayModel→signalTiming+simulationMode+fillRule 映射在 modal 层）。红队自审通过。commit `0408f1a7`。
- 2026-08-11 **POST-1 验收通过 ✅（审计方独立删行复测）**：5/5 断言级全红——T1 改 total=-1 红（ListPublished 主路径）/ T6 删 ORDER BY+LIMIT 红 / T7 删 ErrNoRows 分支红（logged 1 want 0）/ T3 删 backtestContent 接线红 / T8 删 error 块红。T1-T8 对抗证明 8/8 有效；实现仅 T6/T7 抽函数行为不变。门禁全绿实测：go build / go test marketplace+user / check-file-lines 0err / tsc 0err / vitest 144pass / npm build。POST-1 闭环。
- 2026-08-11 **POST-1 测试补强完成**：T1/T3/T6/T7/T8 五项重做，每项施工方删行实测必红。T1 走 ListPublished 集成（缓存命中 total 真值）；T6/T7 抽 `buildShareDecayStatusQuery`+`resolveDecayStatus` 可测函数（行为不变），T7 用 zaptest/observer 验证 ErrNoRows 不产生日志；T3 渲染真实 BottomPanelSection；T8 渲染真实 LivePerformanceTab（mock fetch reject）。门禁全绿：go build + check-file-lines 0err + tsc 0err + vitest 144pass + npm build。待审计方独立删行复测。
- 2026-08-11 **UX-1~8 返工复审（审计方实测）**：4 缺陷实现 ✅ 验收通过；对抗证明 5/8 无效（删行实测 T1/T6/T7 仍绿 + T3/T8 结构判定同模式）→ 补强测试返工单。88a95c3d 文档裁剪=用户批准（✅done 明细归档 git，已修订 CLAUDE.md/builder-sop §2.6）。
- 2026-08-11 **POST-1 UX-1~8 返工施工**：4 缺陷修复（UX-3 缓存 total=-1 / UX-4 移动端回测面板 / UX-8 tsc 真门禁 / UX-1 查询确定性）+ 8 项对抗测试（T1-T8）。门禁全绿：go build + check-file-lines 0err + tsc + vitest 146pass + build。待审计方实测验收。
- 2026-08-11 **Part D 验收 + UX-1~8 复审**：Part D（runbook 12实写+CQ-2 knip 0issue+CQ-9 前端收尾）审计方实测 ✅。UX-1~8 复审：TS清零✅实测，UX-3 缓存total=-1/UX-4 修错面板/UX-8 CI空操作 3缺陷打回，8项对抗测试全缺，维持🟦open。
- 2026-08-10 **FILL-SIM 验收通过 ✅**：Phase A-E 全部完成，2阻塞级缺口补强后审计方独立复测通过，⚠️解除。FILL-SIM 闭环。
- 2026-08-10 **FE-TRUST-1 审计方实测验收 ✅**：分享页零信任迁移+后端回撤bug修复，Claude复审通过。
- 2026-08-10 **EXEC-PARAMS 验收通过 ✅**：回测执行假设参数端到端接线+核心bug修复，审计方实测通过。
- 2026-08-10 **POST-5 agent重构收尾 ✅**：plan驱动+语义追问全落地，agent重构里程碑完成。
