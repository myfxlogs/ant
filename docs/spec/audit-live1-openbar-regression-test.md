# 施工 Spec：LIVE-1 回归测试——实盘策略只在收盘 bar 执行

> **状态：✅ 已由审计方自行实现**（2026-08-07）。小任务不值当外部 round-trip；spec 留作 LIVE-1 推理 + 测试设计记录。落地：`shouldRunOnBar` + `live_runner_bar_filter_test.go`（6 例全绿）。后续大特性仍走外部 agent 施工（0028 模式）。

> 来源：接手审计 #6 实盘调度，发现 LIVE-1（open bar 泄漏进策略执行流）。**修复已落地并验证绿**（见「当前代码状态」）；本 spec 是配套**回归测试**，锁住"open bar 不触发 OnBar"这一行为，防日后重构悄悄删守卫导致 LIVE-1 复活。

- **负责人（设计 + 验收）**：Claude（审计方，第一责任人）
- **施工者**：外部 agent（只按 spec 施工，无设计权、无决策权）
- **关联**：`docs/audits/tech-debt-registry.md` LIVE-1；`backend/internal/connect/strategy/live_runner.go`

## 背景（为什么要有这个测试）

实盘策略 runner 通过 `barBroker` 订阅 bar。该 broker 同时被图表 SSE 订阅——图表要"正在形成的 K 线"(open/in-progress bar, 每 500ms 一帧),策略却只要**收盘 bar**(每周期一次)。LIVE-1 的根因是两者共享一条流、runner 不过滤 `bar.Closed`,导致:

1. 策略每 500ms 在同一根未定型 bar 上重复执行 OnBar;
2. bar 窗口(500 根)被 open 快照淹没,数分钟内真实历史 bar 被挤出,指标重复计数;
3. OnBar 策略在未定型 bar 发信号 → 实盘与收盘-bar 回测发散(动摇"实盘战绩可信")。

修复 = runner 只在 `bar.Closed` 时执行。**本测试锁住这个语义。**

## 当前代码状态（修复已落地，施工方勿重复修复）

`backend/internal/connect/strategy/live_runner.go` 的 `runLiveEventLoop`,bar 分支现状(先 Read 确认):

```go
case bar, ok := <-p.barCh:
    if !ok {
        s.log.Warn("LiveStrategyRunner: bar channel closed, exiting")
        return
    }
    // LIVE-1: execute on finalized bars only. ...(6 行说明注释)
    if !bar.Closed {
        continue
    }
    if p.extraSymbolSet[bar.Symbol] && bar.Period == p.cfg.Timeframe {
        handleExtraSymbolBar(bar, p.extraBars)
        continue
    }
    if bar.Symbol != p.cfg.Symbol || bar.Period != p.cfg.Timeframe {
        continue
    }
    s.handleBar(p.runCtx, p.cfg, bar, p.bars, p.session, p.firstBar, p.activeSess, p.extraBars)
```

## 施工任务

### 1. 抽纯函数 `shouldRunOnBar`（同文件 `live_runner.go`，放 `runLiveEventLoop` 附近）

```go
// shouldRunOnBar reports whether a finalized bar for the strategy's primary
// symbol/timeframe should trigger OnBar. Open (in-progress) bars are excluded
// (LIVE-1): they are chart-feed snapshots, not strategy events — feeding them
// re-triggers OnBar mid-formation, corrupts the bar window, and diverges live
// from closed-bar backtest. Intra-bar updates belong on the tick channel.
func shouldRunOnBar(bar *mthub.BarUpdate, symbol, timeframe string) bool {
	return bar.Closed && bar.Symbol == symbol && bar.Period == timeframe
}
```

### 2. event loop 主 symbol 分支改用谓词

把 bar 分支里「LIVE-1 长注释 + `if !bar.Closed { continue }` + `if bar.Symbol != ... || bar.Period != ...`」三块,替换为:

```go
    // LIVE-1: extra-symbol context windows also use finalized bars only.
    if bar.Closed && p.extraSymbolSet[bar.Symbol] && bar.Period == p.cfg.Timeframe {
        handleExtraSymbolBar(bar, p.extraBars)
        continue
    }
    if !shouldRunOnBar(bar, p.cfg.Symbol, p.cfg.Timeframe) {
        continue
    }
```

（LIVE-1 的详细说明移到 `shouldRunOnBar` 的 doc comment 里。语义不变：open bar 仍被跳过，主 symbol 仍需 symbol+timeframe 匹配。）

### 3. 表驱动单测（新文件 `backend/internal/connect/strategy/live_runner_bar_filter_test.go`）

**验证链路**：bar → `shouldRunOnBar` → 只有「收盘 + 主 symbol + 主 timeframe 匹配」返回 true。

```go
func TestShouldRunOnBar(t *testing.T) {
	const sym, tf = "EURUSD", "1h"
	cases := []struct {
		name string
		bar  *mthub.BarUpdate
		want bool
	}{
		{"closed matching", &mthub.BarUpdate{Symbol: sym, Period: tf, Closed: true}, true},
		{"open matching — LIVE-1 must skip", &mthub.BarUpdate{Symbol: sym, Period: tf, Closed: false}, false},
		{"closed wrong symbol", &mthub.BarUpdate{Symbol: "GBPUSD", Period: tf, Closed: true}, false},
		{"closed wrong timeframe", &mthub.BarUpdate{Symbol: sym, Period: "15m", Closed: true}, false},
		{"open wrong symbol", &mthub.BarUpdate{Symbol: "GBPUSD", Period: tf, Closed: false}, false},
		{"zero value bar (nothing set)", &mthub.BarUpdate{}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := shouldRunOnBar(c.bar, sym, tf); got != c.want {
				t.Errorf("shouldRunOnBar(%+v) = %v, want %v", c.bar, got, c.want)
			}
		})
	}
}
```

**对抗 case（必覆盖，不预先公布的也要测）**：
- `open + 匹配 → false`（**LIVE-1 回归关键**；若有人删 `bar.Closed` 检查,此项必红）
- `closed + 匹配 → true`（正路）
- symbol/timeframe 任一不匹配 → false（即使 closed）
- 零值 bar → false

## 约束

- 后端 Go 测试。`go build ./...` 与 `go test ./internal/connect/strategy/...` **全绿**（从 `backend/` 目录跑）。
- `live_runner.go` 现约 384 行，**不得超过 450 行红线**（CLAUDE.md）。只加一个短函数 + 一个测试文件，别扩太多。
- 最小改动：只动 `live_runner.go`（抽函数 + 改分支）+ 新增测试文件；**不改其他生产代码**。
- 禁裸 `grep`/`find`/`cat`；用内置工具。先 Read 文件拿精确文本再 Edit，避免空白字符不匹配。

## 硬性质量要求（不只"能跑"，必须最优）

1. **实现是最优解**——抽纯函数让 LIVE-1 守卫可测、可命名、可复用；event loop 调用它。
2. **代码干净无冗余**——无死代码、无未用 import、无重复的 Closed 判断（主 symbol 路径 Closed 只在谓词里判一次）。
3. **无技术债**——无 TODO/FIXME/nolint。
4. **第一性**——测试名表达"open bar 不触发 OnBar"的链路意图，不是测某个函数内部。

## 验收标准（我会查）

- [ ] `shouldRunOnBar` 存在且 event loop 主 symbol 分支调用它；open bar 仍被跳过。
- [ ] `go build ./...` 绿；`go test ./internal/connect/strategy/...` 绿，`TestShouldRunOnBar` 6 个子用例全过。
- [ ] 删掉 `shouldRunOnBar` 里的 `bar.Closed &&`(模拟回退) → `open + 匹配` 用例**必红**(证明测试有效)。
- [ ] file-lines 无新 🔴。
