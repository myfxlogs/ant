# 施工交接批次 5：对抗测试补强（LIVE-4 + LIVE-7b + Live UI）

> 审计方（Claude Code）2026-08-13 发出。合并三项"实现已验收 ✅、对抗证明缺失/无效"的补强，性质统一为对抗测试补强，一次交接。
> **one task = one scope**：只做本文件三项。**不重新审计、不自由发挥、不扩大范围、不改实现代码**（除非本文件明确要求）。
> **背景铁律**：删了还绿 = 测试无效 = 未完成。本批次每一项都是上批"实现 ✅ 但对抗缺失/无效"的遗留。

---

## 范围

| # | 任务 | 来源 | 级别 | 涉及文件 |
|---|------|------|------|----------|
| 1 | LIVE-4：gate fail-closed 对抗测试无效（NilGate 测错分支）| 批次3 遗留 | P2 | `mthub/service_orders_unit_test.go` + `mthub/service_coverage_test.go` |
| 2 | LIVE-7b：NOTIFY->SSE 闭环零测试 | 批次3 遗留 | P3 | `strategy/strategy_schedules.go`（冒烟，不改实现）|
| 3 | Live UI：3 项对抗测试缺失 | 批次4 遗留 | P1 配套 | `frontend/.../LiveSchedulesTab.tsx` + `ScheduleTable.tsx` + `LiveStrategyPage.tsx` |

---

## 任务 1：LIVE-4 gate fail-closed 对抗补强（P2）

### 根因（审计方 2026-08-13 独立删行实测，证据确凿）

实现没问题：`service_orders.go:131` `if s.gate == nil { return error("gate not configured...") }` + `service_orders_close.go:104` 同理（fail-closed 正确）。

**问题在测试**：6 个 gate 测试里，**NilGate 类 4 个全部无效**（审计方删 `s.gate == nil` 分支后 4 个仍绿）：

| 测试 | 文件:行 | 构造 | 为何无效 |
|------|---------|------|----------|
| `TestEvaluatePlaceGate_NilGate` | `service_orders_unit_test.go:430` | `newTestService()`（gate=NewDefaultGate **非 nil**）| gate 非 nil，根本不触发 nil 分支；测试名"NilGate"误导 |
| `TestEvaluateCloseGate_NilGate` | `service_orders_unit_test.go:439` | 同上 | 同上 |
| `TestEvaluatePlaceGate_NilGate_FailClosed` | `service_coverage_test.go:1968` | `newTestServiceNoGate()`（gate+provider **皆 nil**）| 删 gate 分支后 err 来自 provider 分支（:134），仍 `!= nil` 仍绿 |
| `TestEvaluateCloseGate_NilGate_FailClosed` | `service_coverage_test.go:1999` | 同上 | 同上 |

有效的只有 `NilStateProvider_FailClosed`（:1983/:2010，gate 非 nil + provider nil）2 个--但那测的是 **provider 分支**，不是 gate 分支。

**结论**：`s.gate == nil` fail-closed 分支**零有效覆盖**。

### 修复方向（审计方定，照做）

**核心：让 NilGate 测试在删 `s.gate == nil` 分支后必然 RED。** 两条路径任选其一（推荐 A）：

**路径 A（推荐，最简）— 断言 error 消息区分分支**：
- `service_coverage_test.go:1968/1999` 的两个 `NilGate_FailClosed` 测试：把断言从 `if err == nil` 改为**断言 err 消息包含 `"gate not configured"`**：
  ```go
  if err == nil || !strings.Contains(err.Error(), "gate not configured") {
      t.Fatalf("expected 'gate not configured' fail-closed error, got %v", err)
  }
  ```
  - 有 gate 分支：err = "gate not configured: order rejected (fail-closed)" -> 消息匹配 -> GREEN
  - 删 gate 分支：err = "account state provider not configured..."（provider 也 nil）-> 消息不匹配 -> **RED** ✓
- `service_orders_unit_test.go:430/439` 的两个误导性 `NilGate` 测试（gate 非 nil）：**删除**（coverage 的 `_FailClosed` 已覆盖 nil gate，这俩用 newTestService() 名不副实，留着误导且无效）。

**路径 B — 改构造隔离分支**：
- `NilGate_FailClosed` 改用 `newTestServiceNoGate()` + `svc.SetAccountStateProvider(...)`（注入 provider 使其非 nil，gate 仍 nil）。删 gate 分支 -> 继续 `s.gate.Evaluate(...)` -> nil 解引用 panic -> RED。需 `defer recover` 断言 panic。不如 A 干净。

### 对抗证明（必做，审计方将独立删行复测）

1. 删 `service_orders.go:131-133` 的 `if s.gate == nil { return ... }` 块 -> `TestEvaluatePlaceGate_NilGate_FailClosed` **RED**（断言消息不匹配 / panic）
2. 删 `service_orders_close.go:104-106` 的 `if s.gate == nil { return ... }` 块 -> `TestEvaluateCloseGate_NilGate_FailClosed` **RED**
3. 还原 -> 全 GREEN
4. `NilStateProvider_FailClosed`（已有效）保持不动，删 provider 分支 -> RED（回归确认）
5. **不许动实现代码**（`service_orders.go`/`service_orders_close.go` 的 fail-closed 分支已验收正确，只改测试）

### 门禁
`go build ./...` / `go test ./internal/mthub/...` 全绿 / `check-file-lines --strict`（若改了测试文件行数）。

---

## 任务 2：LIVE-7b NOTIFY->SSE 闭环对抗补强（P3）

### 现状（审计方代码核对，实现完整）

- 4 写路径都调 `s.notifyScheduleChange(ctx)`（-> `pglisten.Notify(ctx, s.svc.DB(), "schedule_change", "")`）：Create `:106` / Update `:153` / Delete `:201` / Toggle `:231`
- `WatchSchedules`（`:251`）`s.pgListen.Listen(ctx, "schedule_change")` 收 notifCh -> ListSchedules -> hash 比较 -> 变化则 `stream.Send`
- **实现完整 ✅，零测试零冒烟**。

### 验证路径（冒烟实测，与 batch2/batch4 同模式）

`pgListen` 是具体类型 `*pglisten.Listener`（非接口）+ `Notify` 是包级函数 -> 单测 mock 困难，需真实 PG。故**走冒烟实测**（live 8022）。

**冒烟契约**（施工方写脚本，参考 `/tmp/smoke_batch2.py` / `/tmp/smoke_batch4.py` 的登录+RPC 模式，凭据从 smoke_batch2.py 提取勿明文重写）：

1. 登录拿 token
2. **开 WatchSchedules SSE 流**（ConnectRPC server streaming，`POST /ant.v1.StrategyService/WatchSchedules`，streaming response）-- 用一个线程/协程持续读流
3. 并发 `CreateSchedule`（event/interval 任一）
4. 断言流在 **5s 内收到至少 1 个 `WatchSchedulesEvent`**（含新建 schedule）
5. `ToggleSchedule` disable -> 断言流再收到事件（isActive 变化）
6. `DeleteSchedule` -> 断言流收到事件（schedule 消失）
7. cleanup

**ConnectRPC streaming 冒烟提示**：ConnectRPC server streaming over HTTP，response 是 enveloped JSON 流（5 字节 flags+length header + JSON payload，重复）。Python 可用 `http.client` 或 `requests` 的 streaming response 逐块解析 envelope；或用 `connect-go` 客户端（若环境允许）。若 streaming 解析成本高，**允许改用 curl `--no-buffer` 观察**：开流后并发 CRUD，断言响应体有新 chunk 输出（非空流 = NOTIFY 触发了 push）。

**对抗证明（必做）**：
- 正路径：CRUD 后流收到事件（GREEN）
- **反路径（删行红）**：临时注释 `notifyScheduleChange` 的 `pglisten.Notify(...)` 调用（strategy_schedules.go:237）-> CRUD 后流 **30s 内无事件**（只剩 ticker 兜底）-> 冒烟断言"5s 内收到"**失败 RED** -> 还原 -> GREEN。**此删行实验必做并记录**（证明事件确实来自 NOTIFY 而非 ticker）。

### 门禁
冒烟脚本输出（正路径 GREEN + 反路径删行 RED）+ `go build ./...`（反路径还原后）。

---

## 任务 3：Live UI 对抗测试补强（P1 配套，原批次4b）

> 原提示词 `builder-handoff-batch4b-livepage-ui-antitest-2026-08-13.md` 内容并入本任务，保留作历史痕迹。

### 现状（批次4 验收，实现 ✅ 对抗零存在）

`LiveSchedulesTab.tsx` / `ScheduleTable.tsx` / `LiveStrategyPage.tsx` **无任何 test 文件**，`tests/e2e/deploy-schedule.spec.ts` 无新断言。5 点契约代码核对全符合，但 spec 必做 3 项对抗测试零存在。

### 三项对抗测试（必做，每项删行必红）

| # | 场景 | 删行（红）| 绿 | 位置 |
|---|------|-----------|-----|------|
| 1 | Enable 成功自动跳 tab1 | 删 `if (next) navigate('/strategy/live?tab=active')` -> 不跳转 | navigate 被调用 path=`/strategy/live?tab=active` | `LiveSchedulesTab.tsx:196-197` |
| 2 | last_error 红色错误显示 | 删 `{row?.lastError && (...)}` 渲染块 -> 错误不可见 | ⚠ + Tooltip + danger 样式显示 lastError | `ScheduleTable.tsx:153-157` |
| 3 | 日志/健康按钮 scheduleId 空 disabled | 删 `disabled={!record.scheduleId}` -> 空 scheduleId 仍可点 | 空->disabled / 非空->enabled | `LiveStrategyPage.tsx:120-127` |
| 4（低优）| healthId modal 关闭清 URL | 删清理行 -> 刷新残留 | 关闭 modal 清 healthId 参数 | `LiveSchedulesTab.tsx` health effect |

### 测试要求（铁律）

- **必须真实组件渲染**（`render(<ScheduleTable schedules={[...]}/>)`），❌ 禁手写 div 拷贝冒充组件（POST-1 教训：手写 div 测拷贝不测真代码）
- **每项真实执行删行实验**：删关键行 -> RED -> 还原 -> GREEN，记录输出
- mock 最小化：只 mock 被测组件实际依赖（`useNavigate` / api client / i18n / 数据 hooks），❌ 不 mock 被测逻辑本身
- 若依赖链 mock 过深，允许抽可测薄函数（行为不变 + 注释说明动因），但**必须走真实渲染路径**
- 抽函数例外（任务 2 可抽 `onToggleActive` 的 navigate 决策；任务 3 activeColumns 可抽为可测函数），实现行为不变

### 门禁
`tsc --noEmit -p tsconfig.app.json` 0err / `vitest run` 全量过（存量 144 + 新增）/ `npm run build` / `go build`（未动后端也应绿）。

---

## 对抗证明汇总

| # | 场景 | 红（删行）| 绿 |
|---|------|-----------|-----|
| 1 | LIVE-4 删 `s.gate==nil` 块 | NilGate 测试 RED（消息不匹配/panic）| GREEN |
| 2 | LIVE-7b 删 `pglisten.Notify` 调用 | 冒烟 5s 内无事件 RED | CRUD 后流收事件 |
| 3 | UI 删 Enable->tab1 联动 | 不跳转 RED | 自动跳 active |
| 4 | UI 删 last_error 渲染 | 错误不可见 RED | 红显 ⚠ |
| 5 | UI 删 disabled 守卫 | 空 scheduleId 可点 RED | disabled |
| 6 | 回归全量门禁 | - | 全绿 |

---

## 红队自审（逐条给出结论）

- [ ] **LIVE-4**：NilGate 测试删 `s.gate==nil` 分支后是否真 RED（实测，非只写断言）？误导性 unit_test:430/439 是否已删除或改 NoGate？`NilStateProvider` 是否保持有效（删 provider 分支仍 RED）？
- [ ] **LIVE-4**：是否误改了实现代码（`service_orders.go`/`service_orders_close.go` fail-closed 分支禁止动，只改测试）？
- [ ] **LIVE-7b**：冒烟反路径（删 `pglisten.Notify`）是否实测 RED？事件是否确实来自 NOTIFY 而非 30s ticker？（删 NOTIFY 后应 30s 内无事件，断言"5s 内收到"才 RED）
- [ ] **LIVE-7b**：是否改了实现代码？（禁止；若坚持改 pgListen 为接口做单测，**先报审计方讨论**，属可演进性改动超 scope）
- [ ] **UI**：每项测试是否真实组件渲染（非手写 div）？删行实验是否每项真实执行？
- [ ] **UI**：现有 144 测试是否全绿（无回归）？是否意外改实现（任务 1-3 实现已验收，除任务 4 URL 清理外禁止改实现；抽函数例外需行为不变）？
- [ ] 门禁全绿实测后记录输出（go build / go test mthub / tsc / vitest / npm build）
- [ ] 部署：本批次纯测试补强，**一般不需重新部署**（除非改了实现代码--任务 4 URL 清理若改实现需重新 docker cp 前端）
- [ ] **⚠️ 上批教训（必读）**：回填 `registry/handover` 时**只追加新行，绝不改/替换审计方既有记录行**。批次4 施工方把审计方批3验收行替换为自身声明 = 铁律违反。pre-commit 钩子仅拦"删行"不拦"改行"，**改行同样违规**。追加你的施工记录行即可。
- [ ] 提交核对：禁 `--no-verify` 绕过 pre-commit 钩子

---

## 回填（不做 = 任务判失败）

1. `docs/audits/tech-debt-registry.md`：每项追加施工记录（测试文件/断言 + 删行红绿输出 + commit）。**只追加，不改审计方验收行。**
   - LIVE-4：更新 DEPLOY-LIVE-4 表行（"对抗 2/4 无效"->"对抗补强 ✅"，标日期 + commit）+ 追加明细行
   - LIVE-7b：更新 DEPLOY-LIVE-7 表行（补"对抗补强 ✅"+冒烟输出）+ 追加明细行
   - UI：追加明细行（指向批次5b 验收）
2. `docs/audits/handover-audit-plan.md` 变更日志加一行（三项补强完成 + 冒烟输出）。
3. 完成后报告：每项测试文件/断言 + 删行红绿记录 + 冒烟输出 + 回填位置。**不自行宣告完成**，等审计方独立删行复测后 ✅ 才权威。

---

## 沟通

- 完成后一句话报告：`批次5 完成：LIVE-4 NilGate 对抗修复（删 gate 分支 RED）+ LIVE-7b 冒烟（删 NOTIFY RED）+ UI 3 项（删行 RED）+ 回填位置`。
- 聊天一句话（定位+铁律）："批次5 补强：三项对抗测试删行必红实测；只改测试不改实现（LIVE-7b 冒烟除外）；回填只追加不改审计方行。"
