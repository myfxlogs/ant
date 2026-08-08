# 施工方交接提示词（2026-08-08）

> **依据 specs**：`docs/spec/copy-leakage-protection-spec.md`（跟单外泄防护）+ `docs/spec/strategy-review-followup-spec.md`（migration/压测/UX/清理 batch）
> **强制遵守** `docs/audits/builder-sop.md`：① one task = one scope（不扩大）；② 对抗证明（删关键一行→测试必红）；③ **不自行宣告 ✅**，等审计方实测；④ 完工回填三层（`tech-debt-registry.md` 🟦→✅ + 对抗证明 + `handover-audit-plan.md` 变更日志）；⑤ REUSE preflight（`bash scripts/cap.sh <关键词>`，禁止重复造轮子）。
> **节奏**：按优先级一次一个 task，完工回填后再领下一个。讨论与施工并行，互不阻塞。

---

## 提示词 1【🔴 P0 · ops · 先做】运行实例 migration 追平

```
依据 docs/spec/strategy-review-followup-spec.md Part A。
现状：运行库停 migration 265，缺 266(ARCH-4 magic_number)/267(FEAT-5 decay_status)，测试者在跑旧 build，测试结论失真。
任务：
1. git status backend/migrations/ 确认 266/267 .up.sql 在仓库；未提交先提交。
2. 唯一部署方式：docker compose build backend && docker compose up -d backend（自动跑 pending migrations）。禁止宿主 go build→docker cp。
3. 验证三连：schema_migrations 含 266/267 + marketplace_strategies.decay_status 列存在 + strategy_schedules.magic_number 列存在 + /healthz 200 + 后端日志无 migration error。
对抗证明：追平前 decay_status 查询失败（审计方已实测 "column does not exist"），追平后必存在——留输出截图。
完工回填：handover 变更日志加一行（registry 无对应条目则新建 MIG-3）。不自行宣告 ✅，等审计方实测。
```

## 提示词 2【🔴 P0 · 最高战略价值】跟单外泄防护 Phase 1 — 账号绑定强制

```
依据 docs/spec/copy-leakage-protection-spec.md Phase 1（§4）。
范围（one task=one scope）：仅 Phase 1 账号绑定，不碰 Phase 2 检测。
步骤：
1. 读 spec §4 + builder-sop.md。REUSE preflight：bash scripts/cap.sh subscribe；cap.sh schedule；cap.sh gate。
2. 任务 1.1 先核验：当前 purchase/subscribe→schedule 是否已强制"free 层 1 账号"。如实回填（真根因与 spec 假设不同→如实写）。
3. 1.2 绑定模型：migration（subscription_bound_accounts 或 strategy_schedules 约束）+ tier 账号上限入 system_config（subscription.account_limit.free=1 / .pro=5 / .enterprise=N），admin 面板可改（非硬编码，复用现有 system_config + admin config UI 模式）。
4. 1.3 执行闸：在 Gate（risk-gate 单一 chokepoint D6-A）校验"绑定账号数 < tier_limit 且当前 account 已绑定"，超额→拒绝（错误："超出档位账号上限，升级 Pro"）。
5. 1.4 前端：购买/启动显示档位额度 + 绑定管理 UI + admin 档位配置控件。
对抗证明：free 用户绑第 2 个账号→必被拒；移除 1.3 校验→测试必红。
Gate：go build ./... + go test ./... + cd backend && go run ./tools/check-file-lines --strict + cd frontend && npm run build。
完工回填：registry 新建 LEAKAGE-1 条目 🟦→✅ + 对抗证明 + handover 变更日志。不自行宣告 ✅。
```

## 提示词 3【🟠 P1】跟单外泄 Phase 2 — 跟单检测（Phase 1 完成后）

```
依据 docs/spec/copy-leakage-protection-spec.md Phase 2（§5）。前置：Phase 1 完成。
范围：仅 Phase 2 检测。warn-not-block（不硬阻断，防误伤合法用户）。
步骤：
1. 任务 2.1 先做 mtapi runtime 探查：调 Account/AccountSummary，确认是否暴露"账号是否 signal provider"。回填结论（决定后续信号集；不暴露则降级 2.2+2.4）。
2. 2.2 PlacedType_Signal 监控：bound 账户 order/deal 流出现 PlacedType=Signal→标记（账号在接收跟单，异常）。
3. 2.3（若 2.1 暴露）signal-provider 状态检测→admin 告警 + 通知作者。
4. 2.4 多会话/多终端异常→标记。
5. 2.5 告警通道复用 deploy/prometheus/alerts.yml + 通知 SSE。
对抗证明：模拟 bound 账户出现 PlacedType_Signal 交易→检测必触发告警；移除监控→不触发→红。
完工回填：registry LEAKAGE-2 条目 + handover。不自行宣告 ✅。
注意：Non-goal = 检测外部被动跟单者（不可观测），别过承诺。
```

## 提示词 4【🟠 P1 · 审计方主导】前端 UX 系统审计

```
依据 docs/spec/strategy-review-followup-spec.md Part C。
性质：审计任务（产 UX 问题清单），非直接修复。
分工：UX 判断层 = 审计方(Claude Code)或引入第三方；机械扫描层 = 施工方先做。
施工方先做机械扫描：遍历 5 大流（marketplace / purchase-subscribe / strategy workspace / account 管理 / auth-landing-share），输出：
- i18n 缺失项（5 语言 en/zh-cn/zh-tw/ja/vi 全用户可见字符串）
- 响应式断点问题（移动端）
- 空态/错误态/加载态缺失清单
进 registry POST-1。判断层（任务完成度/新手可懂性/"战绩不可骗"对散户是否易懂）由审计方接手。
对抗证明（机械）：i18n 扫描覆盖 5 语言 × 全可见字符串，0 遗漏。
完工回填：registry POST-1 更新 + handover。不自行宣告 ✅。
```

## 提示词 5【🟡 P2】性能冒烟压测（migration 追平后）

```
依据 docs/spec/strategy-review-followup-spec.md Part B。前置：Part A 完成（否则压旧 build 无意义）。
工具 k6。50 并发虚拟用户 × 关键路径（ListPublished / Subscribe+Purchase / RunBacktest SSE / StartStrategy / SSE 订阅流）× 2 分钟。
采集 p50/p99 延迟、错误率、PG/Redis/NATS 连接池水位、goroutine、/metrics。输出瓶颈清单进 registry。
对抗证明：调小 PG pool_max_conns=5 重跑→必现连接错误（证明压测能发现问题）。
不在 prod 跑，测试环境隔离。完工回填 registry POST-2 + handover。不自行宣告 ✅。
```

## 提示词 6【🟢 P2】残留清理

```
依据 docs/spec/strategy-review-followup-spec.md Part D。
D.1 runbook 实写：12 个 docs/runbook/*.md 占位补全（症状/影响/诊断步骤/应急处置/常见根因），对齐 deploy/prometheus/alerts.yml 规则名。参考已有 docs/runbook/mql2go-known-pitfalls.md 范式。
D.2 CQ-2：cd frontend && npx knip → 清理未引用导出/组件。
D.3 CQ-5：全量核验 eslint-disable 用法，非硬违例则保留+注释理由，否则清理。
对抗证明（D.1）：grep alerts.yml 的 runbook: 链接 → 每个指向的文件非空 + 含 5 段。
Gate：npm run build + check-file-lines。完工回填 registry CQ-2/CQ-5/POST-3 + handover。不自行宣告 ✅。
```

---

## 队列总览

| # | 任务 | 优先级 | 前置 |
|---|---|---|---|
| 1 | migration 追平 | 🔴 P0 | 无（先做，阻断测试）|
| 2 | 跟单 Phase 1 账号绑定 | 🔴 P0 | 无（可与 1 并行，不同 scope）|
| 3 | 跟单 Phase 2 检测 | 🟠 P1 | 2 完成 |
| 4 | 前端 UX 审计（机械层）| 🟠 P1 | 无 |
| 5 | 性能冒烟压测 | 🟡 P2 | 1 完成 |
| 6 | 残留清理 | 🟢 P2 | 无 |

施工方从 1+2 起步（可并行），完工回填后领 3/4/5/6。
