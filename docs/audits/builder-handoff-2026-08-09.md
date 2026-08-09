# 施工方交接提示词（2026-08-09）

> **依据 spec**：`docs/spec/frontend-zero-trust-share-page-spec.md`（FE-TRUST-1 分享业绩页——后端正确回撤 + 衍生统计迁移 + 修真 bug）
> **强制遵守** `docs/audits/builder-sop.md` 三铁律 + 验收 checklist（§200）：① one task = one scope（不扩大）；② 对抗证明（删关键一行→测试必红）；③ **红队自审**（完工后对着 §200 checklist + 验收 5 维过一遍，带着债提交=失败）；④ 不自行宣告 ✅，等审计方实测；⑤ 完工回填三层；⑥ REUSE preflight（cap.sh）。
> **节奏**：一次一个 task，完工回填 + 自审后再交付，不并行多任务。

---

## 提示词 1【🟠 中优 · 含真 bug 修复】FE-TRUST-1 分享页——后端正确回撤 + 零信任迁移

```
依据 docs/spec/frontend-zero-trust-share-page-spec.md（权威设计，T1-T4 + 验收 + 对抗证明已写齐，先完整读再动手）。

背景：/share/:token 公开页（面向潜在买家）前端在 TS 算回撤/交易统计 = 零信任违背；核验连带发现后端 share_service.go:summarizeTrades 的 maxDD(:177-179) 是"单笔最负 Profit"非真回撤，经 perf.MaxDrawdown 显示在公开 OG 社交预览图(share_og_image.go:97)。同一"最大回撤"在 OG 图(美元·最差单笔) vs 分享页 KPI(百分比·真回撤) 显示完全不同的数。

范围（one task = one scope）：仅 share 子系统——share_service.go / share.proto / share_handler.go / 分享页 SharePerformancePage*.ts(x) + gen 产物。不动 analytics/summary/marketplace/admin（已合规，spec §一已确认）。

步骤：
1. 先读 spec 全文 + builder-sop.md。
2. REUSE preflight（T1 动工前必做）：bash scripts/cap.sh drawdown；cap.sh aggregate。后端已有正确回撤算法 analytics_rolling.go:103 computeDrawdownEvents(runningMax + (runningMax-eq)/runningMax*100) + live_performance.go:208-222(decimal peak-trough)。抽共享 computeMaxDrawdownPct(equityPoints) decimal.Decimal，share + analytics 共用（可演进性：单一真相源，勿 inline 各写一份）。品种聚合查 analytics_compute.go:228/258 复用。PR 描述逐条给 REUSE:/NEW:。
3. T1 修真回撤（根因）：删 summarizeTrades 的 maxDD=单笔最负错误逻辑；在 BuildSharePerformance（已有 equityPoints，:62）算真正 peak-to-trough 回撤，decimal，百分比口径（与分享页现有显示 + marketplace quality.go MaxDrawdownPct 闸一致）。
4. T1 补交易级统计：tradeSummary 增 best/worst/avgWin/avgLoss/symbolStats（全 decimal）。
5. T2 proto 加 ShareTradeStats + ShareSymbolStat（decimal 用 string 传输）；share_handler.go:107-115 填充。MaxDrawdown 字段复用（T1 已让它变正确，语义=真回撤百分比，无需新字段）。
6. T3 前端删 computeMaxDrawdownPct/computeTradeStats/aggregateBySymbol 三函数，改读后端 proto 纯渲染；ShareData 接口同步加 tradeStats/symbolStats 字段（对齐 gen 类型）。

对抗证明（缺 = 任务判失败）：
- 回撤正确性：构造 equity [100,120,90,110] → 真回撤 25%(peak=120→trough=90)，后端 maxDrawdownStr 返回该值。故意退回旧"maxDD=单笔最负"逻辑 → 测试必红（证修了真 bug，非空操作）。
- 零信任迁移：删前端三函数后 grep computeMaxDrawdownPct|computeTradeStats|aggregateBySymbol src/ 零命中 + tsc 无残留引用。
- OG 一致性：OG 预览图与分享页 KPI 用同一 perf.MaxDrawdown → 同值。

红队自审（完工后汇报前，对着过一遍，任一不过回去改，不要带着债提交）：
- 防御性：equity<2 点 / peak=0(除零！live_performance.go 的 decimal Div 对零除数会 panic，必须 guard，参考 analytics_rolling.go 的 if runningMax>0) / 空 trades / 单笔 / 全盈或全亏(avgWin/avgLoss 分母为零) / 空 symbol name —— 全部 graceful 返回 "0"/"-" 不 panic。
- 可演进性：回撤算法抽成共享 helper，share 与 analytics 共用，非 inline 重复。
- 克制：没动 spec 范围外文件；没新建并行文档；没顺手重构 share_service。
- 测试质量：正/负/边界齐全（含上述除零/空集合边界）。
- 意图理解：解决的是"公开面显示错误回撤语义 + 前端零信任"真问题，非字面搬运。

Gate：go build ./... + go test ./internal/connect/user/... + cd backend && go run ./tools/check-file-lines --strict(0🔴) + cd frontend && npm run build && npm test。

完工回填（不做 = 任务判失败）：
1. tech-debt-registry.md FE-TRUST-1 🟦open → ✅done（标日期）+ 追加【真实根因】（重点写清后端 maxDD=单笔最负、被误标为回撤、OG 预览公开面显示）+ 修复方式 + 对抗证明 + 自审结论。如实写，真根因与 spec 不同就纠偏（高价值）。
2. handover-audit-plan.md 变更日志加一行。
3. CLAUDE.md「零信任」相关段补一句经验（公开面衍生统计必须后端算；回撤须用 equity 算真值，勿用单笔最差冒充）。
4. commit（doc + code 一并，message 写清"修分享页回撤 bug + 零信任迁移"）。
5. 不自宣告完成——状态标 ⚠️待 Claude 复审，等审计方实测翻 ✅。
```
