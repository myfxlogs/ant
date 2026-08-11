# Builder Handoff · UX-1~8 修复（2026-08-11）

> **角色**：审计方（Claude Code）出提示词；施工方（Windsurf）实现 + 回填。
> **依据**：registry POST-1 UX 审计发现清单（2026-08-10 审计方 3 agent 静态扫描 + 逐条验证）。
> **完工回填纪律**：registry 🟦→✅（标日期）+ 真实根因/对抗证明/测试结果；handover 变更日志一行；**不自行宣告完成**，等审计方核对状态 + 实测。
> **Scope 铁律**：只做下面 8 项。不做其他清理/重构/优化（one task = one scope）。

---

## 总览

| # | 项 | 严重度 | 范围 | 修复指引 |
|---|----|--------|------|----------|
| UX-1 | 衰减徽章买家不可见 | 🔴 产品核心 | 后端 share + 前端 3 处 | **独立 spec**：`docs/spec/ux1-decay-badge-spec.md` |
| UX-2 | 实盘战绩失败静默"无数据" | 🔴 | 前端 1 文件 | 本文件 §2 |
| UX-3 | 客户端筛选+服务端分页空页 | 🔴 | 后端 proto/handler + 前端 | 本文件 §3（**根治方案已定：后端过滤**） |
| UX-4 | 移动端回测结果不可见 | 🔴 | 前端 1-2 文件 | 本文件 §4 |
| UX-5 | AI Fix 空 strategyId 静默失败 | 🔴 | 前端 1 文件 | 本文件 §5 |
| UX-6 | 实盘 SSE 断流伪装"无策略" | 🔴 | 前端 1 文件 | 本文件 §6 |
| UX-7 | 4 公开路由无 ErrorBoundary | 🔴 | 前端 1 文件 | 本文件 §7 |
| UX-8 | build 无类型检查（657 存量 TS 错误） | 🔴 机制根因 | 前端全部 | 本文件 §8（**量大，拆批执行**） |

**执行顺序建议**：UX-8 的 `tsc --noEmit` 排查放在最后（先在无 tsc 环境修完 UX-1~7 的代码改动，再统一清 657 存量——否则 tsc 存量错误淹没新增错误难分）。**UX-7 可先做**（5 分钟，防下一个 UX-P0）。

---

## §2 UX-2 实盘战绩接口失败静默伪装"无数据"

**位置**：`frontend/src/pages/marketplace/components/LivePerformanceTab.tsx:66-68`

**问题**：`catch { setPoints([]); setSummary(null); }` —— 接口失败与"确实无实盘"渲染同一空态，用户分不清。

**修复**：
1. 加独立 `error` 状态（如 `const [error, setError] = useState<string | null>(null)`），catch 里 `setError(...)`（用 i18n 友好文案，别直出原始 message——同 registry 🟡"原始报错直出"问题，此处顺带避免）。
2. 渲染分支区分四态：`loading` / `error`（错误提示 + 重试按钮，调原 fetch 函数）/ 空（Empty，原逻辑）/ 有数据（原逻辑）。
3. 重试按钮复用现有 refetch 或原 fetch 函数。

**对抗证明**：组件测试——mock 该查询失败 → 断言 error 分支渲染（提示 + 重试按钮）；删 `setError` → 必红。mock 成功但空 → 空态渲染（不误伤）。

## §3 UX-3 客户端筛选+服务端分页 → 空页 + total 错

**位置**：`useMarketplace.ts:47-68`（客户端过滤 + total 估算）+ `MarketTab.tsx:48`（`empty={length===0}`）

**根因（2026-08-11 审计方核实）**：`ListPublishedRequest`（marketplace_service.proto:180）**无价格过滤字段**；`ListPublishedResponse` **无 total 字段**。前端在服务端分页（limit/offset）之上做客户端价格过滤 → 第 2 页数据可能全是 paid（free 过滤后为空）+ `total = ... : page*pageSize+1` 纯估算。

**修复（根治：过滤下沉后端）**：
1. **后端** `proto/ant/v1/marketplace_service.proto`：
   - `ListPublishedRequest` 加 `string price_filter = 7;`（`all|free|paid`，空=all；注释标注取值）
   - `ListPublishedResponse` 加 `int32 total = 2;`
   - `buf generate` 重新生成。
2. **后端 handler**（`backend/internal/connect/marketplace/` 下 listPublished 实现）：
   - `price_filter == "free"` → `AND (price_amount IS NULL OR price_amount = '0')`
   - `price_filter == "paid"` → `AND price_amount IS NOT NULL AND price_amount > '0'`（price_amount 是 text 列，数值比较注意类型转换或 `price_amount::numeric > 0`——以现有 SQL 风格为准）
   - 同条件 `COUNT(*)` 返回 total。
3. **前端** `useMarketplace.ts`：
   - 删客户端过滤（strategies 直接用 allStrategies，删 `priceFilter` 相关 filter 逻辑——但**保留 priceFilter state**，因为筛选 UI 还在）
   - `listPublished` 请求加 `priceFilter` 参数（`priceFilter === 'all' ? undefined : priceFilter`）
   - `total` 从响应 `resp.total` 取（注意：myPublished 调用也要更新——它不传 priceFilter 则 total 是全量，检查调用处是否用 total；**myPublished 的 total 语义不要被影响**）
   - useRpcQuery 的 queryKey 加 priceFilter（否则切筛选不重拉）。
4. **回归守卫**：author tab（myPublished）传 `priceFilter: 'all'` 或省略——行为不变。

**对抗证明**：
- 后端 test：`price_filter="free"` → 响应只含免费策略 + total == 免费数（构造混合数据集）；删 WHERE 条件 → 必红。
- 前端：组件测试 MarketTab——mock 2 页数据，free 过滤 → 第 2 页非空且 total 正确；删 priceFilter 传参 → 必红（或后端 test 覆盖，前端 UI 测试选一个写）。

## §4 UX-4 移动端回测结果完全不可见

**位置**：`frontend/src/pages/strategy/components/workspace/BottomPanelSection.tsx:30`（`if (isMobile) return null`）

**问题**：手机跑完回测只剩 toast，结果面板只在桌面渲染。

**修复**：
1. 移动端不 return null——改渲染 antd `Drawer`（或可复用现有 Drawer 容器），内容复用 `BacktestResultsTab`（与桌面同组件，保持一致性）。
2. 触发入口：回测完成时（或点结果按钮）自动开 Drawer。
3. 桌面行为**零变化**（isMobile 分支只影响移动端渲染路径）。

**对抗证明**：组件测试——`isMobile=true` → 断言结果内容可渲染（Drawer 打开状态或内容存在）；把修复回退为 return null → 必红。

## §5 UX-5 AI Fix 主路径静默失败

**位置**：`frontend/src/pages/strategy/components/workspace/useAIFix.tsx:77-81`（strategyId 空时 `setDiffOpen(false)` 静默丢弃 diff）

**问题**：手写代码+保存的新策略恰无 strategyId（只有 ImportEA 路径会设）→ AI Fix 点了没反应。

**修复**（二选一，取 A；若 A 实施中遇阻再取 B 并说明）：
- **A（推荐）**：strategyId 空时禁用 Apply/运行按钮 + tooltip 提示"请先保存策略"（i18n）。保存成功后 strategyId 有了，按钮恢复。
- B：回退"仅更新编辑器"（不弹 diff，直接把 AI 修复写进编辑器）。

**对抗证明**：组件测试——空 strategyId → Apply 按钮 disabled + 提示可见；保存后 → 可用。删禁用逻辑 → 必红。

## §6 UX-6 实盘 SSE 断流伪装"无策略"

**位置**：`frontend/src/pages/strategy/LiveStrategyPage.tsx:39-49`（catch 清空列表无重连）

**问题**：SSE 断流 → 列表清空 → 用户以为没策略在跑。对比 `LiveSchedulesTab` 有 2s 重连。

**修复**：
1. catch **不清空旧数据**（保留上次成功数据）。
2. 加断线横幅（antd Alert："连接中断，正在重试…"），数据恢复后消失。
3. 重连机制：复用 LiveSchedulesTab 的 2s 重试 pattern（先读它怎么写的，REUSE）。

**对抗证明**：组件测试——首次成功数据 → 模拟断流 → 旧数据仍在 + 横幅渲染；恢复 → 横幅消失。删"保留旧数据" → 必红。

## §7 UX-7 4 公开路由无 ErrorBoundary/Suspense

**位置**：`frontend/src/AppRoutes.tsx:173-188`（SharePerformancePage/LandingPage/BrokersPage/StrategySharePage 裸 lazy 挂载）

**问题**：UX-P0 崩溃（分享页白屏）即此缺口后果——公开路由任何渲染错误 = 整页白屏无兜底。

**修复**：
1. 统一 wrap：公开路由也走 `PageWrapper` 或共享 `ErrorBoundary` + `Suspense`（先看 private/admin 路由的 wrap 是什么，REUSE 同一组件）。
2. ErrorBoundary 兜底 UI：友好错误页 + 刷新按钮（i18n）。

**对抗证明**：手动/组件测试——构造渲染错误 → 显示兜底 UI 而非白屏（如渲染测试中断言 ErrorBoundary fallback）；移除 wrap → 必红。

## §8 UX-8 build 无类型检查（机制根因，防 UX-P0 类回归）

**位置**：`frontend/package.json`（build=`vite build`）+ `tsconfig.app.json` + 全 src/

**问题**：vite build 不做类型检查，657 个存量 TS 错误全绕过——UX-P0（未声明 `trades` 引用）即因此上线。

**修复（分两步）**：
1. **先清存量**：`npx tsc -p tsconfig.app.json --noEmit 2>&1 | grep "error TS" | wc -l` 确认当前存量（2026-08-10 时 657）。**逐条修**（机械任务，按错误分组批量处理）：
   - **机械类**（绝大多数）：缺导入/未使用变量/未使用参数（noUnusedLocals/noUnusedParameters）/类型不匹配可安全标注的（`as` 断言最小化，每个 `as` 带一行理由注释）/可选链空值处理。
   - **需要设计判断类**（接口不匹配/跨层类型冲突）：标记 `// TODO(UX-8): ...` + 记录清单，**不要**用 `any` 塞掉（除非该处本就是动态边界且有理由注释）——这类单独列出等审计方复核。
   - **纪律**：修类型**不得改变运行时行为**（只加类型不换逻辑；纯类型问题用类型手段解决）。
2. **再入阻断**：存量清零后，`package.json` build 改为 `"build": "tsc -p tsconfig.app.json --noEmit && vite build"`（本地 build 即拦截）；CI frontend job 跑 `npm run build` 自动覆盖（核对 ci.yml 确认）。
3. `src/gen/**` 自动生成文件如有错误：不改，`tsconfig.app.json` 的 exclude/ignore 处理（先看现在怎么排除的——knip.json 已 ignore gen，tsconfig 也应有对应处理；gen 若报错属于生成器问题，报审计方）。

**对抗证明**：
- 清零后 `npx tsc -p tsconfig.app.json --noEmit` exit 0（**这是 UX-P0 的机制性修复**：UX-P0 的 `(228,32) TS2304` 若重现必红）。
- 把任一已修类型错误还原 → tsc 必红。
- 存量清零 + build 加 tsc 后：`npm run build` 全绿。

**执行建议**：清 657 错误按目录拆批（如每次 `tsc` 输出只处理前 N 条，修完重跑看下一个），每批 build 一次确认无运行时回归（vitest 140 全绿是底线）。**如果 657 数字对不上（存量已变）以实测为准**。

---

## 门禁（每项修完都跑）

```bash
# 前端（每项后）
npx tsc -p tsconfig.app.json --noEmit   # UX-8 落地前存量错误不要求清零，但新增错误必须为 0
npm run build
npm test                                # vitest run，140 全绿底线

# 后端（UX-1/UX-3 改动后）
go build ./...
go test ./internal/marketplace/ ./internal/connect/marketplace/ ./internal/connect/user/
cd backend && go run ./tools/check-file-lines --strict   # 0🔴
```

## 完工回填纪律（不做 = 任务失败）

1. `docs/audits/tech-debt-registry.md` POST-1 下：UX-1~8 每条 🟦→✅（标日期 + 真实根因/修复方式/对抗证明/测试结果）。UX-8 附：修前存量数 → 修后 0 + 需要设计判断的残留清单（若有）。
2. `docs/audits/handover-audit-plan.md` 变更日志一行。
3. **不自行宣告完成**——等审计方核对状态 + 实测。

## 验收标准（审计方实测）

- UX-1~7 各自对抗证明成立（删关键行必红）+ 代码走查（意图理解/可演进性/防御性/克制）
- UX-8：tsc exit 0 + build 含 tsc + CI 覆盖 + 657 存量清零（以实测数为准）
- `go build` / `tsc` / `npm run build` / `vitest run` / `check-file-lines --strict` 全绿
