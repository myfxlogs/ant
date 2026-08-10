# Builder Handoff · Part D 残留清理（2026-08-10）

> **角色**：审计方（Claude Code）出提示词；施工方（Windsurf）实现 + 回填。
> **依据 spec**：`docs/spec/strategy-review-followup-spec.md` **Part D**（D.1 runbook 实写 / D.2 CQ-2 knip / D.3 CQ-5 eslint-disable 核验）。
> **完工回填纪律**：registry 🟦→✅（标日期）+ 真实根因/对抗证明/测试结果；handover 变更日志一行；**不自行宣告完成**，等审计方核对状态 + 实测。
> **Scope 铁律**：只做下面 3 个子任务。不做其他清理/重构/优化（one task = one scope）。

---

## Phase 1 — D.1 runbook 实写（12 个占位文件）

**背景**：`deploy/prometheus/alerts.yml` 12 条告警的 `runbook:` 链接指向的文件全是占位（~500-700B，开头"占位（post-launch 补全）"）。Alertmanager 不受影响，但 oncall 无手册。

**文件清单（12 个，与 alerts.yml 链接一一对应，已核对）**：

| 组 | 文件 |
|---|---|
| mthub（4） | `docs/runbook/mthub-order-error.md` / `mthub-order-reject.md` / `mthub-place-latency.md` / `mthub-session-disconnect.md` |
| md（5） | `docs/runbook/md-circuit-open.md` / `md-dlq-spike.md` / `md-clock-skew.md` / `md-tick-latency.md` / `md-normalizer-fallback.md` |
| platform（3） | `docs/runbook/backend-down.md` / `backend-high-memory.md` / `pg-pool-exhausted.md` |

**任务**：每份补全为 5 段结构：
```
## 症状
## 影响
## 诊断步骤
## 应急处置
## 常见根因
```

**内容来源（禁止凭空编造）**：
- 该告警在 `alerts.yml` 中的 `expr` / `severity` / `labels` / `annotations`（从中推导触发条件与症状）
- 代码真实路径（如 `md-circuit-open` → mdgateway circuit breaker；`mthub-*` → `internal/mthub/`；`pg-pool-exhausted` → pgxpool 配置）
- 现有完整 runbook 范式：`docs/runbook/mt-incidents.md`（真实事故复盘）、`mql2go-known-pitfalls.md`、`server-resource-limits.md`

**对抗证明**：`grep 'runbook:' deploy/prometheus/alerts.yml` 取 12 个链接 → 每个文件存在、>1KB、**无"占位"字样**、含全部 5 个段标题。删掉任一文件的任一段 → 对抗证明必失败。

**红队自审（Phase 1 必过）**：
- [ ] 告警名 ↔ runbook 文件名对齐（链接 404 = 失败）
- [ ] 诊断步骤给**可执行命令**（psql/curl/docker logs/grep），不是空话
- [ ] 常见根因写**真根因**（从代码/历史事故推），不写"可能是网络问题"式废话

---

## Phase 2 — D.2 CQ-2 前端 knip 死代码清理

**背景**：registry CQ-2：前端死代码 80 文件 + 96 exports（2026-08-09 核验量，以本次扫描为准）。knip 已配置（`frontend/knip.json`：ignore `src/gen/**` / `public/**` / `App.css` / `i18n/resources/**` + 1 个 ignoreExports），`package.json` 有 `knip ^6.31.0`。

**任务**：
1. `npx knip` 出报告（记录基线数量：多少文件/多少 exports/多少 dependencies）。
2. **分类处置**，逐条进清理清单（留档）：
   - **真死代码**（全仓无引用）→ 删。
   - **误报**（动态引用 / `React.lazy` 路由懒加载 / 模板字符串 import / 测试辅助 / story 文件 / 被 `vitest.config.ts` exclude 的 e2e 辅助）→ **保留**，清单标注理由。
3. **CQ-9 前端部分**（后端已删 ✅ 2026-08-10，产品决策已定）：`pages/market/Market.tsx` / `pages/trading/Trading.tsx` / `components/chart/{PriceChart,useChartData}.tsx` 及其独有依赖 → 直接删（后端对应链已清，前端这步是 CQ-9 收尾）。
4. 删完后 `npx knip` 再跑 → **0 issue**（或仅剩 ignoreExports 白名单项）。

**误删守卫（强制）**：清理后必须 `npm run build` exit 0 + `vitest run` 全绿。任何编译错误/测试红 = 误删，立即还原。

**对抗证明**：清理后 `npx knip` 0 issue；把任一已删导出加回 → 报告必重新出现该条（证明删的是真死代码，非误删）。

**红队自审（Phase 2 必过）**：
- [ ] 动态 import（`import(...)`）/ 路由懒加载引用 —— 勿删
- [ ] i18n 资源键 / 语言文件 —— 已在 ignore，勿手删
- [ ] `src/gen/**` —— 已在 ignore，**勿手改**（重建即覆盖）
- [ ] CSS 类名拼接 / tailwind 动态类 —— knip 不查 css，人工留意勿删样式文件
- [ ] 被 `/* eslint-disable */` 全文件标记的（如 `PriceChart.tsx` 有行级 immutability disable）—— 按引用判断，不因有 disable 而保留

---

## Phase 3 — D.3 CQ-5 eslint-disable 核验

**背景**：registry CQ-5：35 处 `react-hooks/exhaustive-deps` disable（多数带 REF 注释，非零容忍硬违例，清理属增量优化）。全仓 `eslint-disable` 共 188 处（含 `src/gen/**` 全文件 `/* eslint-disable */` 自动生成豁免）。

**任务**：全量核验（**排除 `src/gen/**`**，自动生成豁免勿动），每处分类：
- **带 REF 注释且理由成立**（如 `react-hooks/immutability -- ref mutation in callback is safe (not during render)`）→ **保留**，核验表记"核验通过"。
- **带 REF 注释但理由不成立 / 无注释** → 删 disable 或补真实理由注释。
- 规则名含 `react-hooks/exhaustive-deps` 的：若 deps 数组确实无法满足（外部回调/ref/闭包语义）→ 保留 + 确保注释解释了 why；若其实可满足 → 修复 deps 并删 disable。

**输出**：核验表入 registry（CQ-5 条目下），每条 = `文件:行 / 规则 / 处置(保留|修复|删) / 理由`。

**对抗证明**：核验表覆盖**全部**非 gen 的 eslint-disable（数量对上）；修复删掉的 disable，把注释还原 → eslint 必重新报错（证明该 disable 确实压着真实 lint 错误，非冗余）。

**红队自审（Phase 3 必过）**：
- [ ] 勿为清零而删出 bug（exhaustive-deps 的 disable 多数正当，保留 ≠ 偷懒）
- [ ] 行级 `// eslint-disable-next-line` vs 块级 vs 全文件 `/* eslint-disable */` —— 处置粒度不同，勿混淆
- [ ] gen 文件**一行不改**（重建即覆盖，改了也是白改）
- [ ] 修复 deps 后必须 `npm run build` + `vitest run` 全绿（deps 修复可能改行为——有测试覆盖则安全）

---

## 完工回填纪律（不做 = 任务失败）

1. `docs/audits/tech-debt-registry.md`：
   - **CQ-2** 🟦→✅（标日期 + knip 前后数量 + 删除清单摘要 + 对抗证明）
   - **CQ-5** 🟦→✅（标日期 + 核验表摘要 + 对抗证明）
   - **CQ-9** 条目追加"前端清理完成"（后端已 ✅，前端这步收尾）
   - **POST-3**（runbook 占位）销账 ✅
   - 只改状态列 + 追加，不删条目、不改审计方事实陈述
2. `docs/audits/handover-audit-plan.md` 变更日志加一行（日期 + 任务 + commit + 关键数字）。
3. **不自行宣告完成**——等审计方核对状态 + 实测。

## 验收标准（审计方实测）

- 12 个 runbook 无占位 + 5 段齐全 + 对抗证明成立
- `npx knip` 0 issue + 清理清单齐全
- CQ-5 核验表齐全 + 对抗证明成立
- `npm run build` / `vitest run` / `go build ./...` / `check-file-lines --strict` 全绿
