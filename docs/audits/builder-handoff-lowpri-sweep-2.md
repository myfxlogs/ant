# 施工派工单 — LOWPRI-SWEEP-2（复审副产清扫：CQ-11 死簇 + CQ-12 lint 红）

**日期**: 2026-09-19 · **设计方**: Devin CLI · **施工**: 施工方（ANT_ROLE=builder）
**立项背景**: LOWPRI-SWEEP-1 复审副产两登记——CQ-11（internal/ai 传递性死代码簇）+ CQ-12（`npm run lint` 主枝存量红：WorkspaceCenterColumn.tsx 1 error+2 warnings，修后 lint 全绿）。

---

## S1【CQ-11】internal/ai 传递性死簇删除

**死簇边界（2026-09-19 已逐项复核，施工前重跑复核门）**：

| 删 | 证据 |
|---|---|
| `backend/internal/ai/template_library.go` **整文件** | TemplateLibrary/NewTemplateLibrary/MatchByFamily/Match/matchCategory/topByCategory/matchByName/countMatches 全仓零调用方（含测试） |
| `backend/internal/repository/ai_strategy_templates_repository.go` **整文件** | CQ-10 已删 repo 方法，残留 `AIStrategyTemplate` 类型+`ParameterSlotsString` 仅被 strategy_prompt.go 死链语法引用 |
| `strategy_prompt.go` 中 `PromptParams` type | 生产零构造点 |
| `strategy_prompt.go` 中 `BuildSystemPrompt`+`BuildUserPrompt` | 全仓零调用方（含 Template 渲染块 :178-182 随之死） |

**保留（活）**：`StrategyPromptBuilder`/`NewStrategyPromptBuilder`/`BuildFeedbackPrompt`/`FeedbackPromptParams`（auto_fix.go:66-68 消费）/`DetectCodeStrategyType`（auto_fix.go:67 消费）/`IntentResult`（clarification.go 自有链，PromptParams 死后仍活）。

**复核门（先跑再删）**：
```bash
grep -rn "TemplateLibrary\|PromptParams{\|BuildUserPrompt\|BuildSystemPrompt\|AIStrategyTemplate" backend/ --include="*.go" | grep -v "internal/ai/template_library.go\|internal/ai/strategy_prompt.go\|internal/repository/ai_strategy_templates_repository.go"
```
必须零命中（`PromptParams{` 的 FeedbackPromptParams 子串不算）——发现新引用即停手回报。
**注意**：删 PromptParams 后 `strategy_prompt.go` 的 `repository` import 若无剩余引用须一并删（`FeedbackPromptParams` 不引用 repository——检查）；`Intent` 字段随 PromptParams 死。禁动 `ai_strategy_templates` 表/migration/种子。

## S2【CQ-12】WorkspaceCenterColumn.tsx 拆修使 lint 归零

- **修 warning×2**：`onNewSource`(:147)/`onSectionChange`(:159) 包 `useCallback`——这俩每渲染重建使 :193 useMemo 失效，是 warning 也是真问题。
- **修 error×1**：文件 313 行（eslint 计 281>250）——抽取 ≥1 个内联渲染块为独立组件文件（候选：`:166+` `backtestHistoryPanel` 等 JSX 块），使 eslint 计行 ≤250。抽取时 props 显式化、沿用同目录组件惯例。
- **门禁**：`cd frontend && ./node_modules/.bin/eslint src` exit 0（当前 3 problems 全消）+ `npm run lint` 绿 + tsc 0 + vitest 不回归。

## 边界（不做）

- 不删 `ai_strategy_templates` 表/migration/种子数据。
- 不动 FeedbackPrompt 链、IntentAnalyzer/clarification、auto_fix.go。
- CQ-12 只做这一文件（其余 lint 项为零）。
- 不部署、不 push。

## 门禁

```bash
cd backend && go build ./... && go vet ./internal/ai/... ./internal/repository/... && go test ./internal/ai/... ./internal/mdgateway/... ./internal/repository/... && go run ./tools/check-file-lines --strict
cd frontend && ./node_modules/.bin/eslint src && npm run lint && npx tsc --noEmit && npx vitest run
git diff --check
```

- 对抗证明：S1 删后 build 绿即引用清零证明（如实 N/A 于断言级 mutation——删除型债）。S2 eslint 归零即证明。
- 串行，完成报证据等复审。
