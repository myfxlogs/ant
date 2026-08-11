# UX-1~8 返工单（2026-08-11 审计方复审后）

> 前提：上一批 `990aa947`/`a371ce23` 的 TS 清零声明已被审计方实测验收 ✅；但 UX-1~8 因 3 缺陷 + 零对抗测试 + 零回填维持 🟦open。本单只做返工，**one task = one scope**：不重构、不顺手改别处、不扩大范围。

## §1 必改项（4 项）

### 1. UX-3 🔴 缓存命中 total=-1（`backend/internal/marketplace/publish.go:289-290`）
- 现状：`if cached, ok := s.pubCache.get(cacheKey); ok { return cached, -1, nil }` → 前端分页 total=-1。
- 修复：`publishedCacheEntry`（`:23`）加 `total` 字段，`set`（`:336`）存入 COUNT 结果，命中路径 `return cached, entry.total, nil`。**不得**只在非缓存路径修。

### 2. UX-4 🔴 修错面板（移动端回测结果仍不可见）
- 现状：`frontend/src/pages/strategy/components/workspace/BottomPanelSection.tsx:46-61` 移动端 Drawer 装的是 `ChartBottomPanel`（持仓/历史）；回测结果 `BacktestResultsTab`（`frontend/src/components/backtest/BacktestResultsTab.tsx:99` 默认导出，桌面路径在 `BacktestPanel.tsx:113` 用）只在 `!isMobile` 的 WorkspaceAIPanel 渲染。
- 修复：移动端 Drawer 改为渲染回测结果面板。**复用 `BacktestResultsTab`**（props 来源对齐 `BacktestPanel.tsx` 的接线方式），**禁止复制组件**；Drawer 内 `ChartBottomPanel` 可保留在二级入口（不删桌面行为）。
- 验收：mobile viewport 跑完回测 → 结果（指标/净值）可见；桌面零回归。

### 3. UX-8 🔴 tsc 门禁空操作（3 处）
- `frontend/package.json`：build script 改为 `tsc --noEmit -p tsconfig.app.json && vite build`（或 `tsc -b`，选哪种都行，须真的检查 src）。
- `.github/workflows/ci.yml:253-254`：`npx tsc --noEmit` → `npx tsc --noEmit -p tsconfig.app.json`（root solution-style config 是 0 文件空操作，实测过）。
- **披露 erasableSyntaxOnly 移除动因**（handoff 铁律 §8 设计判断类）：PR 描述写清"恢复 flag → src/gen/ 11×TS1294 enum 错误（被 import 的 gen 不受 exclude 挡），故移除"。禁止静默。

### 4. UX-1 ⚠️ share 查询不确定性（`backend/internal/connect/user/share_handler.go:77-82`）
- 现状：`_ =` 吞错 + 无 ORDER BY/LIMIT 1 → 同账户多条 published 时任意行；ErrNoRows 时 decayStatus 空串。
- 修复：加 `ORDER BY updated_at DESC LIMIT 1`；ErrNoRows（pgx.ErrNoRows）→ 落 `'none'`，其余错误记日志（禁 `_ =` 吞）。

## §2 对抗证明测试（8 项，删关键行必红，否则未完成）

| # | 修项 | 测试 | 删除哪行必红 |
|---|---|---|---|
| T1 | UX-3 | 缓存命中（同请求 2 次）→ total > 0 且 = 过滤后真实数 | 删 `entry.total` 返回 → 红 |
| T2 | UX-3 | price_filter≠all 时 SQL 过滤生效（后端行为测试，stub 签名不算） | 删 filter 进 SQL → 红 |
| T3 | UX-4 | mobile viewport 跑完回测 → 结果面板渲染（组件测试） | 删 Drawer 内结果面板 → 红 |
| T4 | UX-8 | `npx tsc --noEmit -p tsconfig.app.json` 对含错误文件必红（本地注入临时错误验证） | 去掉该命令 → 红（注：旧 CI 命令 0 文件，删了也绿——正是缺陷） |
| T5 | UX-8 | build script 含 tsc（脚本字符串断言或实测 build 对错误文件失败） | 去掉 tsc → 红 |
| T6 | UX-1 | 同账户 2 条 published（decay_status 不同）→ 2 次调用返回同一值 | 删 ORDER BY/LIMIT 1 → 红（或 5 次抽样判定不稳） |
| T7 | UX-1 | 账户无 published → 返回 'none' 不报错 | 删 ErrNoRows 分支 → 红 |
| T8 | UX-2 | LivePerformanceTab：error 态渲染 Alert + 重试按钮（此前零测试） | 删 error 态 → 红 |

## §3 红队自审（任务级 edge cases，开工前逐条自查并在 PR 列结论）
1. UX-3：cacheKey 是否已含 priceFilter？（`:288` 已含——确认没动 key 语义）；myPublished 路径不走 total，确认不回归。
2. UX-4：Drawer 高度/滚动；BacktestResultsTab 16 个 props 全有值；空态（未跑回测）显示什么。
3. UX-8：CI 里 `npm ci` 后 tsc 可用（node_modules 就绪）；`tsc -b` vs `--noEmit` 与现有 tsconfig（noEmit:true）不冲突。
4. UX-1：share 账户有策略但全下架/decay_status NULL → COALESCE 兜底 'none'。
5. i18n：本单不新增 key（沿用上批已加的 4key×5 语言）；若被迫新增必须 5 语言齐。

## §4 门禁（全部必绿再提交）
```bash
go -C backend build ./...
go -C backend run ./tools/check-file-lines --strict
npx tsc --noEmit -p frontend/tsconfig.app.json
cd frontend && npx vitest run   # 存量 140 + 新增 ≥8 全绿
cd frontend && npm run build
```

## §5 回填纪律（不做 = 任务判失败）
1. `docs/audits/tech-debt-registry.md` POST-1：UX-1~8 条目追加「返工记录」（日期 + 真实根因 + 每项对抗测试的删行验证结果）；状态改 ✅ **等审计方实测验收后才权威**。
2. `docs/audits/handover-audit-plan.md` 变更日志加一行（提交 hash + 修了什么 + 测试数）。
3. **完成即 commit**（上批 132 文件未提交的教训）；commit 后把 hash 报给审计方。
4. 不自宣完成——审计方核对 + 实测后才 🟦→✅。

## §6 验收标准（审计方将实测）
- 4 缺陷逐项复现修复：UX-3 缓存命中 total 正确 / UX-4 手机回测结果可见 / UX-8 CI tsc 真检查（`--listFilesOnly` 非 0）+ build 含 tsc + erasableSyntaxOnly 动因披露在 PR / UX-1 查询确定 + 无吞错。
- 8 项对抗测试逐个删行验证红（审计方独立复测 ≥2 项）。
- §4 门禁全绿 + 三层回填齐 + 工作树干净。
