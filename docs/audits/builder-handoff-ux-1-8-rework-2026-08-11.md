# UX-1~8 返工单·补强版 v2（2026-08-11 审计方复审后）

> 状态：`af565aa7` 已完成 4 缺陷实现（审计方实测验收 ✅）+ 8 项对抗测试（**审计方删行实测 5/8 无效**）。本版只做**测试补强**，one task = one scope，**禁改实现代码**（唯一例外：T6/T7 抽函数暴露可测点，行为不得变）。

## §1 实现已完成 ✅（勿动）

UX-3 缓存 entry 带 total 命中返真值 / UX-4 移动端 Segmented 复用 `BacktestResultsTab` / UX-8 CI+build tsc 真门禁+动因披露 / UX-1 `ORDER BY updated_at DESC LIMIT 1`+ErrNoRows→'none'。门禁全绿（审计方实测：tsc 0 / vitest 146 / go build / go test / check-file-lines 0err / build）。

## §2 补强对抗测试（5 项重做，删关键行必红，审计方将独立复测）

| # | 无效原因（审计方实测证据） | 重做要求 |
|---|---|---|
| T1 | `publish_cache_test.go` 只测 set/get 单元；缓存命中复现 `-1` 仍绿 | **走 `ListPublished` 集成**：同参数调 2 次（预置冷缓存），断言第 2 次（命中）返回 total = 过滤后真实 COUNT > 0。删缓存命中 total 返回 → 红 |
| T3 | `ux-rework.test.tsx` 渲染手写 `<div>`；删 `backtestContent` 接线仍绿 | **渲染真实 `BottomPanelSection`**（isMobile=true + Drawer open），断言回测内容可见。删 Drawer 内 backtestContent 渲染 → 红 |
| T6 | `share_decay_test.go` 断言测试文件内字符串字面量；删 handler `ORDER BY`/`LIMIT 1` 仍绿 | **抽可测函数**（如 `queryShareDecayStatus(ctx, pool, accountID)` 或 SQL builder），单测调真函数。删 ORDER BY/LIMIT 1 → 红 |
| T7 | ErrNoRows 分支逻辑复制进测试 = 同义反复 | 同 T6 抽函数后测真分支：删 ErrNoRows 分支 → 红 |
| T8 | 手写 Alert 不渲染 `LivePerformanceTab` | **渲染真实 `LivePerformanceTab`**（mock 请求失败），断言 Alert+重试按钮、点击重试重新请求。删 error 渲染 → 红 |

**保留有效不动**：T2（`buildPublishedCountQuery` SQL 断言）、T4/T5（package.json 契约）。

## §3 红队自审（开工前逐条自查，PR 列结论）

1. T1 冷/热缓存构造：用 `s.pubCache.clear()` 或直接预置 `set()`。
2. T3/T8 依赖 mock：`WorkspaceContext`/`useWsAI`/fetch——照 `src/test/components.test.tsx` 先例。
3. T6/T7 抽函数后 `go test ./internal/connect/user/` 全绿，行为零变化。
4. 每个重做测试**自己先做删行实验**：删保护行 → 必须红，结果写进 PR。

## §4 门禁（全部必绿再提交）

```bash
go -C backend build ./...
go -C backend run ./tools/check-file-lines --strict
npx tsc --noEmit -p frontend/tsconfig.app.json
cd frontend && npx vitest run   # 存量 146 + 新增全绿
cd frontend && npm run build
```

## §5 回填纪律（不做 = 任务判失败）

1. registry POST-1「2026-08-11 复审块」追加补强记录：每项重做测试的删行验证结果（实测真红）。
2. `docs/audits/handover-audit-plan.md` 变更日志加一行（hash + 重做项 + 验证结果）。
3. **完成即 commit**；hash 报审计方。
4. 不自宣完成——审计方独立删行复测后才 🟦→✅。

## §6 验收标准（审计方将实测）

- 5 项重做测试：审计方独立删行 → **必须实测红**（非声明）。
- §4 门禁全绿 + 回填齐 + 工作树干净。
