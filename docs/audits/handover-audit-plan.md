# 接手审计计划书（Handover Ground-Truth Audit）

> 7 管线 + account-mgmt 全部审完 ✅。上线就绪：所有 launch-blocking 缺口审计方实测清零（2026-08-09）。
> 历史变更日志已删除，靠 git 追溯。仅保留最近 3 条。

---

## 范围

| # | 管线 | 状态 |
|---|------|------|
| 6 | 实盘调度 | ✅ 已审 |
| 1 | 行情引入 | ✅ 已审 |
| 2 | 策略执行 | ✅ 已审 |
| 3 | 订单对账 | ✅ 已审 |
| 5 | 回测 | ✅ 已审 |
| 4 | Agent循环 | ✅ 已审 |
| 7 | 策略市场 | ✅ 已审 |
| — | account-mgmt | ✅ 已审 |

---

## 变更日志

- 2026-08-11 **POST-1 验收通过 ✅（审计方独立删行复测）**：5/5 断言级全红——T1 改 total=-1 红（走 `ListPublished` 主路径）/ T6 删 ORDER BY+LIMIT 1 红 / T7 删 ErrNoRows 分支红（logged 1 want 0）/ T3 删 backtestContent 接线红（getByTestId 抛错）/ T8 删 error 块红（Alert null）。**T1-T8 对抗证明 8/8 有效**。实现仅 T6/T7 抽函数（行为不变）。门禁全绿实测：go build / go test marketplace+user / check-file-lines 0err / tsc 0err / vitest 144pass / npm build。实验编辑全还原，工作树干净。POST-1 闭环。
- 2026-08-11 **POST-1 测试补强完成**：T1/T3/T6/T7/T8 五项重做，施工方删行实测全红。T1 走 `ListPublished` 集成（预置缓存+nil pg，断言 total=42，改 -1→红）；T6 调真 `buildShareDecayStatusQuery()`（删 ORDER BY/LIMIT 1→红）；T7 调真 `resolveDecayStatus()`+zaptest/observer（ErrNoRows 不日志，删分支→ErrNoRows 落 log 路径→红）；T3 渲染真实 `BottomPanelSection`（删 backtestContent 接线→红）；T8 渲染真实 `LivePerformanceTab`（mock reject，删 error 块→红）。实现仅 T6/T7 抽函数（行为不变）。门禁全绿。待审计方独立删行复测。
- 2026-08-11 **UX-1~8 返工复审（审计方实测）**：4 缺陷实现全部验收通过（UX-3 缓存 entry 带 total 命中返真值 / UX-4 移动端 Segmented 复用 BacktestResultsTab / UX-8 CI+build tsc 真门禁+动因披露 / UX-1 ORDER BY+LIMIT 1+ErrNoRows→'none'）。门禁全绿实测：tsc 0err / vitest 146 / go build / go test marketplace+user / check-file-lines 0err / npm build。**对抗证明 5/8 无效（审计方独立删行实测）**：T1 缓存命中复现 -1 仍绿（未走 ListPublished 主路径）、T6/T7 删 handler ORDER BY/LIMIT 1 仍绿（测测试文件内字符串字面量）、T3 删 backtestContent 接线仍绿（手写 div 不渲染 BottomPanelSection）、T8 同模式；有效仅 T2（count SQL）/T4/T5（package.json 契约）。→ 补强测试返工单（T1 集成级 / T3+T8 渲染真实组件 / T6+T7 抽可测函数）。**88a95c3d 文档裁剪 = 用户批准**（✅done 明细归档 git），已修订 CLAUDE.md 无损接手铁律 + builder-sop §2.6 + 钩子自动执行。
- 2026-08-11 **POST-1 UX-1~8 返工施工**：4 缺陷修复（UX-3 缓存 total=-1 / UX-4 移动端回测面板 / UX-8 tsc 真门禁 / UX-1 查询确定性）+ 8 项对抗测试（T1-T8）。门禁全绿。待审计方实测验收。
- 2026-08-11 **Part D 验收 + UX-1~8 复审**：Part D（runbook 12实写+CQ-2 knip 0issue+CQ-9 前端收尾）审计方实测 ✅。UX-1~8 复审：TS清零✅实测，UX-3 缓存total=-1/UX-4 修错面板/UX-8 CI空操作 3缺陷打回，8项对抗测试全缺，维持🟦open。
- 2026-08-10 **FILL-SIM 验收通过 ✅**：Phase A-E 全部完成，2阻塞级缺口补强后审计方独立复测通过，⚠️解除。FILL-SIM 闭环。
