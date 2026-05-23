# ADR-0010 错误码体系

## Status
**Accepted** (2026-05-23) — 按推荐方向 A (自建 internal/errs/ 包)；M0.3 建包 + 映射 Top-50 错误

## Context
DESIGN-REVIEW MR-04 指出：仓库内 `errors.New("...")` / `fmt.Errorf("...")` 裸字符串 632 处；前端拿到的 `message` 中英混杂；错误无法 i18n、无法程序化判别；不符合 AGENT.md「errs 包，禁裸字符串」。

## Options
- **A. 自建 `internal/errs/` 包**（推荐）：定义错误码 enum（如 `ERR_INVALID_SYMBOL`）+ 中/英 message map；后端返回 `errs.E(code, args...)`；Connect interceptor 把 errs 转成 `connect.Error` + structured details；前端按 code 渲染 i18n message
- B. 用 Connect Code 原生：仅用 `connect.CodeInvalidArgument` 等 16 个 grpc-status code；颗粒度太粗
- C. 混合：业务错用 errs 包，传输层用 Connect Code 装载；最终落地仍是 A

## Decision
**A (自建 internal/errs/ 包)**：自建错误码表 + i18n 映射；Top-50 高频错误首批迁移；M0.3 建包。

## Consequences（按 A 推演）
- **+** 错误程序化判别；前端按 code 渲染；运维按 code 聚合告警
- **+** 与 ADR-0011 trace_id 联动后定位极快
- **−** 引入新包；Top-50 高频错误首批迁移；存量裸字符串打 baseline 缓迁
- **−** Connect 层需 interceptor 适配（已有 `internal/interceptor/`）

## 实施约束（M0.3）
- 错误码命名：`ERR_<DOMAIN>_<NAME>`，全大写 + 下划线
- 中文 message 必填，英文 message 可选（fallback 到中文）
- 错误码加 `severity`: info / warn / error / fatal
- 业务码段位约定：`1000-1999` auth；`2000-2999` account；`3000-3999` strategy；...

## Related
- DESIGN-REVIEW MR-04
- AGENT.md 工程纪律「错误集中」
- ADR-0001（Connect RPC 错误传输层）
- ADR-0011（trace_id 关联）

## History
- 2026-05-23 Proposed

- 2026-05-23 Accepted（按推荐方向落地，详见 Decision）
