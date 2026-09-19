# 施工派工单 — LOWPRI-SWEEP-1（低优清扫批：CQ-10 + CQ-5 + MDGATEWAY-5 + TUNING-OVERFIT-2）

**日期**: 2026-09-19 · **设计方**: Devin CLI · **施工**: 施工方（ANT_ROLE=builder）
**立项背景**: i18n/VM 两大债系收官后清 open 队列尾巴。四个低优条目打包一批：死代码×2、硬编码/陈旧数据×3、静默惰性闸×1、缺注释豁免×33。共同主题 = 平台"诚实"红线（fail-closed/如实暴露/不伪造）。
**设计实查（本批全部坐标已逐项核实，2026-09-19）**：

| 条目 | 实证 |
|---|---|
| CQ-10 | `ai_strategy_templates` 全 backend Go 引用 = 仅 `repository/ai_strategy_templates_repository.go` 自身（零生产调用方）；表/migration 保留不删 |
| CQ-5 | `grep -rn "eslint-disable" frontend/src` 排除 `src/gen/` = **33 处**（登记时 11 已漂移），几乎全为 `react-hooks/exhaustive-deps` 无理由豁免 |
| MDG-5a | `brokersearch/search.go:97-105`：mtapi error **或** 空结果 → `staticBrokerFilter` 回退；`staticBrokers:22` 硬编码 Exness IP `18.163.85.196:443` + 4 家空 Access——**资金路径上发陈旧/编造数据**（绑定向导选 server） |
| MDG-5b | `mt4grpc3.mtapi.io:443`/`mt5grpc3.mtapi.io:443` 字面量散落 6 处：search.go:58/61/72/75 + mt4/connection.go:97 + mt5/connection.go:96 |
| MDG-5c | `session_clock.go` **生产零消费**——`SessionClock` 类型全仓仅 `market_state_test.go`/`pure_test.go` 引用；外汇 session 硬编码（closeUTC=22）属死代码带错误交易假设 |
| MDG-5d | adapters GetError 已查 85 处，但"多处次要漏查"需施工方枚举补漏（模式：调用后无 `resp.GetError()` 判定的 RPC 站点） |
| TUNING-OVERFIT-2 | `quality.go:302` `MaxIsOosDegradation.IsPositive() && snap.OosSharpeRatio != ""` — 门**已配 0.5 生效中**（migration 213/260 `true,true`），但全仓 5 处 `BacktestSnapshot{}` 生产点**零填充 Oos 字段**（`AgentBacktestResult` 无 OOS 字段）→ 门恒空转=静默 fail-open |

---

## S1【CQ-10】删死仓库 `ai_strategy_templates_repository.go`

- **目标**：删 `backend/internal/repository/ai_strategy_templates_repository.go`（5 func 全死）。
- **约束**：**表不删**（migration 130 + 种子数据保留——registry 明示"勿连带删表"）；migration 文件一律不动。
- **前置复核**：`grep -rn "AIStrategyTemplatesRepository\|ai_strategy_templates" backend/ --include="*.go" | grep -v migrations` 应仅剩自身文件；若发现新调用方 → 停手回报。
- **文档同步**：`docs/spec/purchase-to-live-link-spec.md:93` 的"保留（AI 生成骨架库仍有用）"行后追加一句订正——"（2026-09-19 订正：ARCH-3 后该 repo 生产零调用，已删；表+种子保留）"，原文不改。
- **对抗证明**：删除后 `go build ./...` 必须绿（若红=漏判调用方，恢复并回报）。

## S2【CQ-5】33 处 eslint-disable 补正当理由

- **目标**：`frontend/src`（**排除 `src/gen/`、`**/i18n/**`——已在 eslint.config.js globalIgnores**）全部 `eslint-disable` 注释补 `-- <理由>`。
- **纪律**：**逐处读上下文写真理由**（如 `mount-only fetch`、`deps intentionally excludes X: re-run loop`），禁批量统一文案。无法给出正当理由的 → 该 disable 很可能掩盖真 bug → 修代码消 disable；修不动的列清单报回复审。
- **语法**：ESLint 9 flat config 支持 `// eslint-disable-next-line <rule> -- <reason>` / `// eslint-disable-line <rule> -- <reason>`。
- **门禁**：`cd frontend && npx eslint src --max-warnings 0`（或项目 lint 脚本）绿、tsc 0、vitest 不回归。

## S3【MDG-5a】staticBrokers 回退 → fail-closed

- **根因**：`Search` 在 mtapi error/空结果时发硬编码静态表（含过期 Exness IP）——绑定向导按陈旧地址连真实交易服务器，违反"服务器实拍优先"红线。且 Connect 本身也要走 mtapi——静态表在 mtapi 宕机时**根本帮不上忙**，只提供假希望+错数据。
- **修复**：删 `staticBrokers` var + `staticBrokerFilter` + `Search` 回退分支。mtapi error → `return nil, err`（错误如实上浮到 UI）；空结果 → 返回空列表（合法"无匹配"）。
- **落点**：`backend/internal/mdgateway/adapter/brokersearch/search.go:20-41`（staticBrokers）、`:97-105`（回退分支）、`staticBrokerFilter` 函数。
- **对抗证明 T3**：新测试 `TestSearch_MtapiError_NoStaleFallback`——gateway 指向不可达端口 → `Search` 返回 error 非空（**不**返回 Exness 等静态行）；mutation 恢复 `staticBrokerFilter` 回退 → RED（error=nil 且结果含 Exness）。

## S4【MDG-5b】mtapi host 字面量收敛为命名常量

- search.go：4 处字面量 → 文件级 `const defaultMT4Gateway/defaultMT5Gateway`（`New`/`NewFromConfig` 共用）。
- `adapter/mt4/connection.go:97` + `adapter/mt5/connection.go:96`：各自文件级命名常量（adapter 间不互引——brokersearch 不 import adapter，方向保持）。
- 纯命名收敛，零行为变化；build+现有测试绿即证。

## S5【MDG-5c】删死代码 session_clock.go

- **前置复核**（硬门）：`grep -rn "SessionClock" backend/ --include="*.go" | grep -v session_clock.go | grep -v _test` 必须为零；发现任何生产引用 → 停手回报（改判为"修默认值"而非删除）。
- 删 `backend/internal/mdgateway/session_clock.go` + 其专属测试文件/用例（`market_state_test.go`、`pure_test.go` 中 SessionClock 相关段——**只删引用它的测试**，文件内其他测试保留）。
- **对抗证明**：删后 build+全测试绿。

## S6【TUNING-OVERFIT-2】惰性闸 → fail-visible（如实暴露，不扩功能）

- **裁定**：完整闭环（publish 路径产 OOS 回测）是 feature 级改动，**不在本批**——本批只做"静默→可见"的诚实化。
- **修复**：`quality.go:302` 块前加分支——`gates.MaxIsOosDegradation.IsPositive() && snap.OosSharpeRatio == ""` → `s.log.Warn("OOS degradation gate armed but snapshot lacks OOS data; gate skipped", zap.String("strategy_id", strategyID), zap.String("gate", "max_is_oos_degradation"))`（`s.log` 已存在，`types.go:30`）。
- **对抗证明 T6**：zap observer core 注入 `New(pg, nil, observedLogger)` → `ValidateBacktestQuality` 喂无 OOS 的 snapshot → 断言恰好 1 条 warn 含 gate 名；mutation 删 warn 行 → RED。
- **登记说明**：registry 条目追加"完整闭环=publish 时 OOS 回测（feature，另立）"。

## S7【MDG-5d】次要 GetError 漏查枚举+补漏

- **枚举法**：mt4/mt5 adapter 全部 RPC 站点（`client.Xxx(` / `resp, err :=`）逐一核对——`err==nil` 后必须查 `resp.GetError()`（code!=0 → return error）。漏站点补齐为 fail-closed error return（错误信息带 RPC 名+code+msg，仿 `:37`/`:107` 惯例）。
- **产出**：报告中列出漏查站点清单（file:line + RPC 名），每处一个修复；无漏则如实报"枚举全覆盖无新增"。
- **对抗证明**：挑 1 处补的站点加测试——stub 返回 `Error{Code:5}` → 断言 error 非 nil；mutation 删该 GetError 检查 → RED。（若全包已覆盖则本项产出=枚举报告）

---

## 边界（不做）

- 不删 `ai_strategy_templates` 表/任何 migration。
- 不做 publish 路径 OOS 回测（feature，另立）。
- 不动 SessionClock 之外的其他 mdgateway 文件（除 S3/S4/S7 指定处）。
- 不动 CQ-5 范围外的 eslint 规则配置（不新增 lint 规则、不改 config——除非一处 disable 实在无法正当化而需删代码，单独列出）。
- 不部署、不 push。

## 门禁（完成报告必须实报以下全部）

```bash
cd backend && go build ./... && go vet ./...
cd backend && go test ./internal/mdgateway/... ./internal/repository/... ./internal/marketplace/... 2>&1 | tail -5
cd backend && go test -race ./internal/mdgateway/adapter/brokersearch/... ./internal/marketplace/...
cd backend && go run ./tools/check-file-lines --strict
cd frontend && npx eslint src --max-warnings 0 && npx tsc --noEmit && npx vitest run
git diff --check
```

- mutation 证据：T3（S3）、T6（S6）、S7 补漏站点各 1 例 RED→restore→GREEN。
- 范围：预期触碰 `backend/internal/repository/ai_strategy_templates_repository.go`（删）、`session_clock.go`（删）+其测试、`brokersearch/search.go`、`adapter/mt4|mt5/connection.go`、`marketplace/quality.go`+测试、`docs/spec/purchase-to-live-link-spec.md`（一行订正）、frontend ~10 文件（disable 注释）。越界先报。
- 串行施工，完成后报证据等复审。
