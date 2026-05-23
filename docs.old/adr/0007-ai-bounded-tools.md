# ADR-0007 AI 策略生成 bounded tools

## Status
**Accepted** (2026-05-23) — 按推荐方向 B (中等白名单)；M4 实施

## Context
ant 的核心特色是「自然语言 → Python 策略」。LLM 调用什么工具决定了：① 生成质量；② 安全边界；③ 成本。当前 `debate_v2` 已有多模型辩论，但工具集合未文档化。

## Options
- **A. 极简白名单**：LLM 仅可调用 ① `get_canonical_symbol_info(canonical)` ② `get_user_strategy_symbols()` ③ `get_indicator_doc(name)`；不可触发回测、不可下单
- B. 中等白名单：A + ④ `dry_run_backtest(strategy_code, range)`（后端代跑）
- C. 强工具：B + ⑤ `submit_order(...)`（直接下单，需 user 确认）

## Decision
**B (中等白名单)**：让 LLM 在生成阶段能"感知"策略历史表现并自我修正；下单仍走人工 confirm。M5 策略市场上线前不开 C。

## Consequences（按 B 推演）
- **+** 闭环更短，生成质量提高
- **+** 下单仍由 user 显式触发，合规风险可控
- **−** dry_run 成本（CPU + LLM token）；需要回测 quota 限额
- **−** 工具调用日志须可审计（接 ADR-0011 trace_id）

## Related
- AlfQ 迁移计划 §M4
- DESIGN-REVIEW UX-CR-02（AI 进度可见性）

## History
- 2026-05-23 Proposed

- 2026-05-23 Accepted（按推荐方向落地，详见 Decision）
