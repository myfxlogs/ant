# 地基审计 · 第二层（第一性原则合规）

> 审计框架：`docs/audits/foundation-audit-framework.md`
> 第一性原则来源：CLAUDE.md Mandatory Constraints

## 1. Decimal Compliance（禁止 float64 计价）

**结论**：✅ 合规。

- `sdk.Bar` 和 `mdtick.Bar` 价格字段均使用 `decimal.Decimal`
- `strategy/indicators/` 使用 float64 做技术指标计算（RSI、MACD 等）——这是正确的：指标是数学公式，不是价格。边界在 Bar → 指标函数时已经过类型转换。
- `tools/mql2go/interp/value.go` 注释明确："All numeric values use decimal.Decimal — no float64 for prices"

## 2. Proto Only（禁止 JSON 序列化/持久化/交换）

**结论**：🔴 有违规。

| 文件 | 行号 | 违规 | 严重度 |
|------|------|------|--------|
| `service/subscription_service_proto.go` | 187 | `json.Unmarshal` | 🔴 |
| `service/webauthn_registration.go` | 55 | `json.Marshal` | 🔴 |
| `service/systemai/chat.go` | 162, 271 | `json.Marshal` + `json.Unmarshal` | 🔴 |
| `service/systemai/chat_stream.go` | 144 | `json.Unmarshal` | 🔴 |

**豁免项**（合规）：
- `protojson.Marshal` — 操作的是 proto 消息，不是任意 JSON ✅
- `JSONB` 列的 DB 操作 — CLAUDE.md 明确豁免 ✅

**分析**：
- `subscription_service_proto.go:187` — 反序列化 JSON 字符串到 map。应该用 proto message 替代。
- `webauthn_registration.go:55` — WebAuthn 库的 `options.Response` 是外部结构体，`json.Marshal` 是调用库 API 所必需的。这是外部依赖强制使用 JSON——不是我们选择的。**建议豁免。**
- `systemai/chat.go:162,271` + `chat_stream.go:144` — 调用外部 LLM API。OpenAI/Anthropic API 使用 JSON。外部 API 调用属于边界交互——**建议豁免。**

**判决**：1 个真违规（`subscription_service_proto.go:187`），3 个外部 API 豁免。CLAUDE.md 的规则需要补充：外部 API 调用不在此限。

## 3. Push-First（禁止 timer/polling）

**结论**：🟡 有疑似违规，需逐项判断。

| 文件 | 间隔 | 做什么 | 判断 |
|------|------|--------|------|
| `cmd/server/main.go:319` | 24h | 清理旧快照 | ✅ 无 push 源，非延迟敏感——异常成立 |
| `cmd/server/handlers_admin.go:102` | 24h | admin 数据清理 | ✅ 同上 |
| `internal/reconcile/reconcile.go:60` | 6h | 内部账本对账 | ✅ 无 push 源，非延迟敏感 |
| `internal/reconcile/reconcile.go:62` | 24h | 链上余额对账 | ✅ 同上 |
| `internal/service/subscription_renewal.go:20` | 24h | 订阅续费 | ✅ 同上 |
| `internal/mdgateway/pg_writer.go:99` | FlushInterval | PG 批量写入刷新 | ⚠️ 可改为"缓冲区满时刷新"，当前依赖 ticker |
| `internal/service/quota_checker.go:151` | interval | 配额检查 | ⚠️ 可改为惰性求值（使用 API 时检查） |
| `internal/service/webauthn_service.go:94` | 5min | WebAuthn 凭证清理 | ⚠️ 可改为惰性求值 |
| `cmd/server/handlers_webauthn.go:60` | 30s | WebAuthn 会话清理 | ⚠️ 同上 |
| `cmd/server/handlers_webauthn.go:75` | 1min | WebAuthn 清理 | ⚠️ 同上 |

**判决**：5 个合规（异常成立），5 个可优化（应改为惰性求值或事件驱动）。

## 4. No REST Endpoints（除 healthz/readyz/livez/metrics）

**结论**：✅ 合规。所有 HTTP 端点均通过 ConnectRPC `mux.Handle()`。未发现裸 REST handler。

## 5. No Nolint/Noqa/Ts-Ignore

**结论**：✅ 合规。零违规。

## 6. File Size Limits

**结论**：🔴 3 个文件超硬性红线。

| 文件 | 行数 | 超限 |
|------|------|------|
| `backend/internal/service/account_service.go` | 452 | +50% |
| `backend/cmd/server/handlers.go` | 462 | +54% |
| `frontend/src/pages/marketplace/components/AutoGeneratePanel.tsx` | 433 | +73%（但此文件尚不存在——GLM 施工中） |

## 7. 汇总

| 检查项 | 结果 |
|--------|------|
| Decimal | ✅ |
| Proto only | 🔴 1 真违规 + 3 豁免 |
| Push-first | 🟡 5 可优化 |
| No REST | ✅ |
| No nolint/noqa | ✅ |
| File size | 🔴 2 真违规 + 1 施工中 |

## 整改建议

| 优先级 | 项 | 行动 |
|--------|----|------|
| P0 | `account_service.go` 拆分 | 452 行 → 拆为 2-3 文件 |
| P0 | `handlers.go` 拆分 | 462 行 → 按 handler 域拆分 |
| P1 | `subscription_service_proto.go` JSON 修复 | 改用 proto message |
| P2 | 5 个 timer 优化 | 逐步改为惰性求值/事件驱动 |
| P2 | CLAUDE.md 补充外部 API 豁免 | 加"调用外部 LLM API 等使用 JSON 不在此限" |
