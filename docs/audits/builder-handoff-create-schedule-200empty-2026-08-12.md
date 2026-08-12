# 施工交接：CREATE-SCHEDULE-200EMPTY（CreateSchedule 200 空 body 修复）

> 审计方（Claude Code）2026-08-12 根因定论完成，三层文档已回填（registry CREATE-SCHEDULE-200EMPTY 条目 / handover 变更日志 / memory open-items-registry）。
> 施工方（Windsurf）任务：按本文档执行修复 + 回填 + 对抗证明。**one task = one scope：只做本文档范围内的事。**
> ⚠️ 首轮交接文档（"部署漂移"结论）**作废**，以本文档为准。

---

## 一、症状（用户报告 + 已复现）

Deploy modal 填完整表单点 Create → `CreateSchedule` API 返回 **HTTP 200 + 0 字节 body**，`strategy_schedules` 表无新记录 → 前端 `created.id` undefined → 不跳转 `/strategy/live?tab=schedules&scheduleId=xxx` → Live 页活跃运行为空。
用户已纠偏：**"调度成功"只是页面 toast 提示（`message.success` 无条件执行），不是真实创建成功**。

## 二、根因（已定论，勿重新排查 — 真根因 = 接线 bug，非部署漂移）

**证据链（完整，审计方逐环实证）**：

1. **接线 bug（git HEAD 就存在）**：`backend/cmd/server/handlers.go:191` 调用 `setupStrategyAndTrading` 时**漏传 `BoundSvc`**（对照 `:126` `registerPostAccountDeps` 有传）。→ `backend/cmd/server/handlers_strategy_runtime.go:81` `boundSvc := p.BoundSvc`（nil）→ `:87` `strategyServer.SetBoundSvc(boundSvc)` → `StrategyServer.boundSvc`（`BoundAccountChecker` 接口，`backend/internal/connect/strategy/strategy_handler.go:35`）接收 **typed-nil** `*service.BoundAccountService` → 接口 type 非空 → `strategy_schedules.go:74` `s.boundSvc != nil` 判断为 **true**。
2. **panic**：`strategy_schedules.go:75` 调 `s.boundSvc.EnsureBoundAccount(...)` → `backend/internal/service/bound_account_svc.go:41` `s.boundRepo.IsAccountBound(...)` **nil 接收者解引用** → `panic: runtime error: invalid memory address or nil pointer dereference`。INSERT 从未执行 → DB 0 条。
3. **200 空机制**：`backend/cmd/server/main.go:266` `sentryhttp.New(sentryhttp.Options{Repanic: false, WaitForDelivery: true})` recover 吞掉 panic → 不 re-panic → net/http 未写响应 → **HTTP 200 + Content-Length: 0**。Sentry DSN 未设置 → panic 不上报，无痕。
4. **前端假象**：`frontend/src/pages/strategy/components/DeployScheduleModal.tsx` handleSubmit 里 `message.success(...)` **无条件执行** → 即使响应空/失败也显示"调度成功"。

**引入历史**：LEAKAGE-1 `be831d5d`（2026-08-08 10:36:46）加 EnsureBoundAccount 检查 + SetBoundSvc 接口注入，但 **strategy runtime handler 接线漏传**。08-08 前无此检查 → 用户能创建调度。对照组 `backend/cmd/server/handlers_strategy.go:65-67` 有 `if d.boundSvc != nil` 保护，`handlers_strategy_runtime.go:87` **无此保护**——漏传 + 无保护两处缺失叠加。

**排除清单（审计方实测，勿重复）**：nginx（ListAccounts 200+1232B 完整透传）/ 前端（请求构造正常）/ DB（表结构正常，0 条因 INSERT 未到达）/ 其他 handler（DeleteSchedule 500 JSON 正常）/ auth（401 正常）。

**⚠️ windsurf 已做的掩盖（必须清理，不是修复）**：
- `backend/internal/service/bound_account_svc.go:35-40`：`defer recover()` 把 panic 转 500 error（`"ensure bound: panic: ..."`）——只掩盖症状，根因（接线）未修。**恢复为无 recover 的干净实现**。
- `strategy_schedules.go` 已清理干净（审计方核对），无需再动。

## 三、修复步骤（按序执行）

1. **一行接线修复**：`backend/cmd/server/handlers.go:191` 的 `strategyTradingParams{...}` 加字段：
   ```go
   BoundSvc: boundSvc,
   ```
   （`boundSvc` 已在 `handlers.go:90` 由 `setupSubscription` 组装，`:126` 已用同源变量传给 `registerPostAccountDeps`，直接用同名变量即可。）
2. **移除掩盖**：`backend/internal/service/bound_account_svc.go` 删除 `EnsureBoundAccount` 的 `(retErr error)` 命名返回值 + `defer recover()` 块（35-40 行），恢复原签名 `func (s *BoundAccountService) EnsureBoundAccount(ctx context.Context, userID, accountID uuid.UUID) error`。
3. **门禁检查**：`cd backend && go build ./...` + `go test ./internal/...`（若时间允许全量）。**不需要动前端**（前端 JS 无改动）。
4. **重新构建部署后端**（唯一合法方式，禁止宿主机 go build → docker cp）：
   ```
   docker compose build backend && docker compose up -d backend
   ```
5. **部署后冒烟验证**（必须，防再犯核心）：
   ```
   # 登录拿 token（e2e 凭据）
   curl -s -X POST http://localhost:8022/ant.v1.AuthService/Login \
     -H "Content-Type: application/json" \
     -d '{"login":"xianhua.chan@gmail.com","password":"Abc123456..."}'
   # 合法 CreateSchedule（用户模板 8403ffab-5840-4825-acb3-7b042f41db59 + 账户 904d14e6-8d67-4541-80f9-f3b7f9587a00）
   curl -s -X POST http://localhost:8022/ant.v1.StrategyService/CreateSchedule \
     -H "Content-Type: application/json" -H "Authorization: Bearer $TOKEN" \
     -d '{"templateId":"8403ffab-5840-4825-acb3-7b042f41db59","accountId":"904d14e6-8d67-4541-80f9-f3b7f9587a00","name":"SMOKE TEST","symbol":"BTCUSDm","timeframe":"15m","scheduleType":"event","scheduleConfig":{"triggerMode":"stable_kline"}}'
   # 断言：HTTP 200 + body 含 "id" + strategy_schedules 表新增记录
   docker exec alphaforge-postgres psql -U ant -d ant -c "SELECT count(*) FROM strategy_schedules"
   # 清理：DeleteSchedule 删掉冒烟记录（保留冒烟输出截图/文本）
   ```

## 四、验收标准

| # | 标准 | 验证方式 |
|---|------|---------|
| 1 | 合法 CreateSchedule → HTTP 200 + JSON body 含 `id` | curl（见上） |
| 2 | DB 新增 schedule 记录 | psql count > 0（对比冒烟前） |
| 3 | 空 symbol → 400 `symbol is required`（回归不破） | curl 空 symbol |
| 4 | 容器二进制时间戳更新 | `docker exec alphaforge-backend ls -la /app/alphaforge` |
| 5 | `bound_account_svc.go` 无 recover/DEBUG 残留 | git diff 干净 |
| 6 | e2e `tests/e2e/deploy-schedule.spec.ts` test 5 手动流程 → 跳转 + 行高亮 | Playwright 手动/自动 |

## 五、对抗证明（必做，删了不红 = 未完成）

1. **修复前基线**：当前容器（windsurf recover 版本）合法请求 → 500 `"ensure bound: panic: ..."`（红）。
2. **修复后**：同一请求 → 200 + JSON 含 `id` + DB 记录（绿）。
3. **删行实验（回归级，必做）**：临时删掉 `handlers.go:191` 的 `BoundSvc: boundSvc` 一行 → 重新构建 → 合法请求 → **必须复现 panic**（无 recover 时 = 200 空；或至少 500 带 panic 字样）→ 还原 + 重建 → 200 + JSON。**实测记录红/绿各一行输出**。
4. **前端 toast 逻辑（如顺手）**：`message.success` 改为 `if (created?.id) { message.success(...); navigate(...) } else { message.error(...) }`——但**此改动超出本任务 scope**，不做也行；如做需补前端 build + 部署，在回填中注明。

## 六、红队自审（任务级 edge cases，必须逐条给出结论）

- [ ] 部署后 `docker ps` 确认 backend 重启（Up 时间重置）且 healthy
- [ ] 冒烟请求的 templateId 是用户模板（8403ffab… 是用户 'E2E 复刻' 模板）——系统模板会 403，注意区分
- [ ] 账户 904d14e6… 状态（reconnecting）——EnsureBoundAccount 只查 `mt_accounts` 归属 + 绑定，不查连接状态，不应因此失败；若失败如实记录
- [ ] 冒烟后清理记录（DeleteSchedule），不留脏数据
- [ ] 部署时是否有未提交 migration？（`git status backend/migrations/` —— 若有 WIP .up.sql 先移走，避免随 build 自动执行）
- [ ] 提交内容核对：registry/handover 变更日志**只追加不删**（pre-commit 钩子会拦删，被拦 = 改好文档再提交，禁 `--no-verify`）；`bound_account_svc.go` 的 recover 删除属于本任务范围，可提交
- [ ] 本次是**接线 bug 不是部署漂移**——不要以"重建容器"为唯一修复动作，必须有 `handlers.go` 代码变更

## 七、回填（不做 = 任务判失败）

1. `docs/audits/tech-debt-registry.md`：CREATE-SCHEDULE-200EMPTY 条目状态 `🟦open → ✅done`（标日期）+ 追加真实修复记录（commit、冒烟输出、对抗证明红绿各一行）。若真实根因与审计方不同，如实写明。
2. `docs/audits/handover-audit-plan.md` 变更日志加一行。
3. 本文件完成后移到 `docs/audits/archive/` 或标注完成（审计方验收后处理）。

## 八、防再犯（审计方建议，低优 follow-up，可在回填里注明不做）

- `main.go:266` sentryhttp `Repanic: false` 静默吞 panic = **"静默错"**（无痕 200 空，无 Sentry DSN 时连日志都没有）→ 建议改 `Repanic: true`（panic 传播 → 连接中断 → nginx 502 可检测）。
- `DeployScheduleModal.tsx` `message.success` 无条件 → 建议按 `created?.id` 条件化。
- 部署后冒烟脚本进部署文档。

## 九、沟通

- 完成后在共享会话（或交接文件）报告：修复动作 + 冒烟输出 + 对抗证明红绿记录 + 回填位置。**不自行宣告完成**，等审计方核对状态 + 实测后 ✅ 才权威。
