# BACKLOG（v2）

> 已知缺陷、待办、未立项的工作项。卡片成熟后晋升到 `ROADMAP.md`。

## M8 候选（业务层渐进重构）

| 候选 | 描述 | 估时 |
|---|---|---|
| M8-X1 | 全部 service/*.go 拆到 ≤ 400 行 | 5d |
| M8-X2 | 业务表 sqlc 覆盖 ≥ 80% | 8d |
| M8-X3 | errs 包替换所有裸字符串错误 | 6d |
| M8-X4 | trace_id 全链路（含 SSE）| 3d |
| M8-X5 | golangci-lint baseline 削减到 < 50 处 | 4d |
| M8-X6 | sandbox_scan 接入 strategy-service 沙箱（M5 残留）| 2d |
| M8-X7 | SQL 注入修复（log_repo + admin_repo）| 3d |
| M8-X8 | risk_control JSONB 残留删除 | 2d |
| M8-X9 | i18n en/ja/zh-tw 补齐前端 | 8d |

## M9 候选（清理老包）

| 候选 | 描述 | 估时 |
|---|---|---|
| M9-X1 | DROP `internal/mt4client/` `internal/mt5client/` | 1d（验证 0 引用即可）|
| M9-X2 | DROP TABLE `kline_data` `tick_data` 等老行情表 | 0.5d |
| M9-X3 | 删除 `service/kline_service*.go` 7 个文件 | 0.5d |
| M9-X4 | docs.old/ 归档外部存档（迁出仓库）| 0.5d |

## 已知缺陷（M7 完成后修）

| ID | 描述 | 优先级 |
|---|---|---|
| BUG-1 | 前端 K 线组件按 broker 切换时偶发 stale state | P2 |
| BUG-2 | AI 助手生成 DSL 字符串时未做 lint | P1 |
| BUG-3 | strategy-service 沙箱内存上限 256MB 偏低 | P2 |

## 暗坑追踪（持续录入）

每条新发现的 mtapi 暗坑 → `docs/spec/16-mtapi-quirks-register.md` 追加，并在此 BACKLOG 记录修复卡片。

| Quirk | 修复卡片 | 状态 |
|---|---|---|
| Q-001 OnQuote.Time 不实时 | M7.1-9 | 🅒 |
| Q-002 mtapi metadata Bearer | M7.1-15 M7.1-16 | 🅒 |
| Q-003 TradeMode=0 阻塞 | M7.1-17 + M7.1-18 | 🅒 |
| Q-004 跨 broker symbol 混合 | M7.1-18 | 🅒 |

## 设计复查（2026-05-23 第二轮）— 待人类决策的过度/缺陷

| ID | 类别 | 描述 | 决策待入 |
|---|---|---|---|
| RV-C1 | 过度设计 | CircuitBreaker per-account 过细，可改 per-broker（同 broker 多账户共享 breaker），简化状态 | M7.1-13 开工前 |
| RV-C2 | 过度设计 | 5σ MAD 离群检查在高频 pip 跳行情下易误杀；考虑改为更保守阈值或 boolean 仅 bid>ask | M7.1-7 开工前 |
| RV-C3 | 缺乏取证 | 100-window xxhash dedup 是为应对 broker 重发，但需 anttrader 真实日志取证；无证据则可砍 | M7.1-8 开工前 |
| RV-C4 | 过度设计 | `/livez/account/{id}` per-account 健康端点；运维实际只看 metric。改 metric-only | M7.1-14 开工前 |
| RV-C5 | 过度设计 | Spill 旋转 size + age 双触发可简化为 size-only（age 限制由 fluentbit 等外部 ship 解决）| M7.1-12 开工前 |
| RV-C6 | 缺陷 | 缺 telemetry 完整性测试（spec/15 列出指标，但无单测验证全部指标确实暴露）| M7.6 加卡 |
| RV-B5 | 缺陷 | vault.Client 接口未 spec（spec/11 §3.1 提到 `Vault` 但无文件定义 `Encrypt/Decrypt/RotateKey` 契约）→ 需新建 `docs/spec/17-secrets.md` | M7.1-2 开工前 |
| RV-B6 | 缺陷 | errs 包接口未 spec（AGENT.md §3.6 强约束 `errs.Code + 中文 user_message` 但无定义）→ 现有 `backend/internal/errs/` 是否复用？需 spec/17 同档 | M8 |
| RV-B7 | 缺陷 | M7.0-7 后端 import 路径变更（`anttrader/gen/proto/ant/v1/...`），需在 spec/10 §3 import 示例增加 v2 路径示例 | M7.0-7 实施时 |

## 文档债

- [ ] `docs/spec/dep-allowlist.md` 依赖白名单（M8 立）
- [ ] `docs/spec/20-factorsvc.md` factorsvc 详细规范（M7.3 完成后回填）
- [ ] `docs/spec/21-quantengine.md` quantengine 详细规范（M7.4 完成后回填）
- [ ] `docs/spec/22-oms.md` 订单状态机（M8 立）
- [ ] `docs/runbook/incidents-general.md` 通用故障手册（M8）
