# 施工派工单：VM-API-TRUTH-1（批次2d：timeseries 无源/handle 假实现 7 API 重分类）

> **角色纪律**：你是施工方，不是验收方。完成后保持 `🟦open（施工完成，待独立复审）`，由 Devin CLI 独立复审后才可变更为 `✅done`。禁止扩 scope、提交、push、部署。commit 用 `ANT_ROLE=builder` 前缀。

## 立项背景

**触发**：`docs/audits/tech-debt-registry.md:126` VM-API-TRUTH-1（P1）。批次1 ✅done（commit e97a43b8，MQL5 order/deal/history 22 API）。批次2a ✅done（commit 8f946579，platform checkup 12 API）。批次2b 派工单 `@67c0ed8a`（Account 全假 + Symbol by-reference 5 API）。批次2c 派工单已落档 `@64c6efc3`（AccountInfo* 假分支+枚举编号）。本批 2d 重分类 timeseries 无源/handle 假实现。

**本批范围**：7 个 API——均为"无权威数据源却返回固定值/假成功值"。重分类 `StatusUnsupported`，编译期拒绝。

**证据链**（HEAD 实拍，2026-09-17 Devin CLI 设计复审）：

| API名 | 文件:行 | 实际返回 | 真实数据源 | 裁定 |
|---|---|---|---|---|
| `CopyBuffer` | `vm_builtin_mql5_ts.go:283-296` | `count` 假成功值，**不填 by-reference 数组**；且 `argI(args,4)` 把数组参数误当 count 读（真签名 count 在 args[3]） | 无（VM 无 indicator handle 子系统，裁定不扩展） | 重分类（handle API + 假成功值） |
| `CopyRates` | `vm_builtin_mql5_ts.go:215-226` | close 价 `DecimalVal` 冒充 `MqlRates` 结构数组 | 无（VM 无 MqlRates 结构概念；per-bar spread/real_volume 无源，无法完整实现） | 重分类 |
| `iSpread` | `vm_builtin_mql5_ts.go:142-145` | `0`（固定） | 无（BarSeries 无 per-bar spread 通道；forex 每根 bar 有真实 spread，0 是伪造） | 重分类 |
| `CopySpread` | `vm_builtin_mql5_ts.go:313-315` | `0`（固定） | 同上 | 重分类 |
| `CopyTicks` | `vm_builtin_mql5_ts.go:317-319` | `0`（固定） | 无（无 tick 级数据源） | 重分类 |
| `BarsCalculated` | `vm_builtin_mql5_ts.go:321-323` | `Bars().Len()` **忽略 handle 参数** | 无（handle API：真语义是"指定指标的已计算 bar 数"，非主序列长度） | 重分类 |
| `SeriesInfoInteger` | `vm_builtin_mql5_ts.go:325-327` | `0`（全 prop 固定） | 无 | 重分类 |

**保留（不误伤）**：
- `iRealVolume`/`CopyRealVolume`（`:137-140`/`:309-311`）——**venue 事实语义**：回测模拟非交易所品种（forex/CFD），real_volume 本就为 0（无集中成交量源）；同 `SymbolSelect=1`/`IsTesting=true` 的环境语义类。若未来支持交易所品种需实源，另立债。
- `Bars`/`iBarShift`/`iHighest`/`iLowest`/`iTickVolume`/`iVolume`/`CopyClose`/`CopyHigh`/`CopyLow`/`CopyOpen`/`CopyTime`/`CopyTickVolume`——有真实 BarSeries/Volume 通道。
- `resolveSeries`/`copyBarData`/`valueAt`/`extremeIndex`/`resolveBarSeries` helper——真实实现共用，保留。

**依赖核查**：`grep -r` 全 backend 仅 mql2go 包内文件引用（builtins.go/builtin_registry.go/vm_builtin_wiring.go/vm_builtin_mql5_ts.go/api_registry.go）。无测试/策略/其他包依赖。

## 设计 SSOT 声明

- 设计文档：本派工单（唯一真相源）
- 相关契约：`AGENTS.md` §0 fail-closed 红线、§7.2 无死代码
- 裁定：`docs/audits/tech-debt-registry.md:126` 2026-09-16 Devin CLI 裁定（StatusUnsupported 方向；MQL5 handle 子系统不扩展）+ 本单保留裁定（Devin CLI 设计复审 2026-09-17）

## 约束与目标

- **目标**：7 个 API 从 implemented 重分类为 StatusUnsupported，编译期拒绝。删除假实现函数 + 注册 + 绑定（无死代码）。
- **范围**：仅以下 5 文件：
  - `backend/tools/mql2go/interp/api_registry.go`（unsupportedSymbols 添加 7 + 新 reason 常量）
  - `backend/tools/mql2go/interp/builtin_registry.go`（implementedPlatform 移除 7）
  - `backend/tools/mql2go/builtins.go`（删除 7 个 nil 注册）
  - `backend/tools/mql2go/vm_builtin_wiring.go`（删除 7 个 fn 绑定）
  - `backend/tools/mql2go/vm_builtin_mql5_ts.go`（删除 7 个假实现函数）
- **测试**：`backend/tools/mql2go/vm_api_truth_test.go`（追加 S6q/r/s 测试到现有文件，不新建文件）

## 边界 / 不做

- **不改** `iRealVolume`/`CopyRealVolume`（venue 事实语义保留）——`LookupAPI` 保持 `StatusImplemented`。
- **不改** `Bars`/`iBarShift`/`iHighest`/`iLowest`/`iTickVolume`/`iVolume`/`CopyClose`/`CopyHigh`/`CopyLow`/`CopyOpen`/`CopyTime`/`CopyTickVolume`（真实实现）。
- **不改** `resolveSeries`/`copyBarData`/`valueAt`/`extremeIndex`/`resolveBarSeries`（真实实现共用 helper，删除函数后不得残留未用函数/import）。
- **不改** `SymbolInfoDouble`/`SymbolInfoInteger`/`SymbolInfoString`/`MarketInfo`（真实实现读 `Broker().SymbolInfo()`；其 prop 编号疑似同类枚举偏移，另立债 `VM-ENUM-NUMBERING-1`，不在本批）。
- **不改** `analyze.go looksLikeMQLBuiltin`、`compile_py_mapping.go`、既有测试逻辑。
- **坐标漂移**：本单坐标按 HEAD 实拍；批次2b/2c 施工会漂移行号——以符号锚点为准，勿按死行号改。

## 施工指令

### S1 — unsupportedSymbols 添加 7 API + 新 reason 常量

- **坐标**：`backend/tools/mql2go/interp/api_registry.go`（reason 常量块 `reasonAccountSymbolStub` 行后 + unsupportedSymbols 列表批次2b 块后）。
- **落点**：
  - reason 常量追加 `reasonTimeseriesNoSource = "timeseries functions returning fixed values or fake success without authoritative data in the backtest VM"`。
  - unsupportedSymbols 追加 7 行（注释块标明批次2d）：
    ```go
    // VM-API-TRUTH-1 batch 2d: timeseries stubs returning fixed values,
    // fake success counts, or handle semantics without authoritative data.
    // Reclassified StatusUnsupported.
    {Name: "CopyBuffer", Status: StatusUnsupported, Category: CatFunction, Reason: reasonTimeseriesNoSource},
    {Name: "CopyRates", Status: StatusUnsupported, Category: CatFunction, Reason: reasonTimeseriesNoSource},
    {Name: "iSpread", Status: StatusUnsupported, Category: CatFunction, Reason: reasonTimeseriesNoSource},
    {Name: "CopySpread", Status: StatusUnsupported, Category: CatFunction, Reason: reasonTimeseriesNoSource},
    {Name: "CopyTicks", Status: StatusUnsupported, Category: CatFunction, Reason: reasonTimeseriesNoSource},
    {Name: "BarsCalculated", Status: StatusUnsupported, Category: CatFunction, Reason: reasonTimeseriesNoSource},
    {Name: "SeriesInfoInteger", Status: StatusUnsupported, Category: CatFunction, Reason: reasonTimeseriesNoSource},
    ```
- **验证**：`go build ./tools/mql2go/` 通过；`LookupAPI("CopyBuffer")` 返回 `StatusUnsupported`。

### S2 — implementedPlatform 移除 7 API

- **坐标**：`backend/tools/mql2go/interp/builtin_registry.go`（implementedPlatform 列表）。
- **落点**：删 `"iSpread"`（`"iTickVolume", "iRealVolume", "iVolume", "iSpread",` 行）、`"CopyRates"`、`"CopyBuffer"`、`"CopySpread"`、`"CopyTicks"`、`"BarsCalculated"`、`"SeriesInfoInteger",` 七处。**保留** `iTickVolume`/`iRealVolume`/`iVolume`/`CopyClose`/`CopyHigh`/`CopyLow`/`CopyOpen`/`CopyTime`/`CopyTickVolume`/`CopyRealVolume`/`Bars`/`iBarShift`/`iHighest`/`iLowest`。
- **验证**：`TestImplementedNamesHaveVMHandlers` 不再校验这 7 个名字。

### S3 — builtins.go 删除 7 个 nil 注册

- **坐标**：`backend/tools/mql2go/builtins.go`（iSpread:382 / CopyRates:383 / CopyBuffer:389 / CopySpread:392 / CopyTicks:393 / BarsCalculated:394 / SeriesInfoInteger:395 附近，行号随 2b 漂移以符号为准）。
- **落点**：删除 7 行 nil 注册，保留 `iVolume`/`iTickVolume`/`iRealVolume`/`CopyRealVolume`/`CopyClose`/`CopyHigh`/`CopyLow`/`CopyOpen`/`CopyTime`/`CopyTickVolume`。
- **验证**：`TestNoDuplicateBuiltins` 通过。

### S4 — vm_builtin_wiring.go 删除 7 个 fn 绑定

- **坐标**：`backend/tools/mql2go/vm_builtin_wiring.go`（`:142-155` timeseries 块附近）。
- **落点**：删 `id("iSpread")`、`id("CopyRates")`、`id("CopyBuffer")`、`id("CopySpread")`、`id("CopyTicks")`、`id("BarsCalculated")`、`id("SeriesInfoInteger")` 七行绑定。保留 `iTickVolume`/`iRealVolume`/`iVolume`/`CopyClose`/`CopyHigh`/`CopyLow`/`CopyOpen`/`CopyTime`/`CopyTickVolume`/`CopyRealVolume` 绑定。
- **验证**：`TestAllBuiltinsWired` 通过。

### S5 — vm_builtin_mql5_ts.go 删除 7 个假实现函数

- **坐标**：`backend/tools/mql2go/vm_builtin_mql5_ts.go`。
- **落点**：删 `builtinCopyRates`（`:215-226`）、`builtinCopyBuffer`（`:283-296`）、`builtinISpread`（`:142-145`）、`builtinCopySpread`（`:313-315`）、`builtinCopyTicks`（`:317-319`）、`builtinBarsCalculated`（`:321-323`）、`builtinSeriesInfoInteger`（`:325-327`）。**保留** `builtinIRealVolume`/`builtinCopyRealVolume`/`builtinBars`/`builtinIBarShift`/`builtinIHighest`/`builtinILowest`/`builtinITickVolume`/`builtinIVolume`/`builtinCopyClose`/`builtinCopyHigh`/`builtinCopyLow`/`builtinCopyOpen`/`builtinCopyTime`/`builtinCopyTickVolume` + 全部 helper。删除后无未用函数/import（`fmt`/`decimal`/`sdk` 仍被保留代码用）。
- **验证**：`go build ./tools/mql2go/` 通过（无未使用符号）。

### S6 — 追加对抗测试（`vm_api_truth_test.go` 现有文件末尾）

- **S6q — 编译期拒绝**：`TestVM_API_TRUTH_1_TimeseriesNoSourceRejected`：表驱动 `unsupportedTimeseriesNoSource = []string{"CopyBuffer","CopyRates","iSpread","CopySpread","CopyTicks","BarsCalculated","SeriesInfoInteger"}`。每 API 最小源码 `void OnTick() { API(); }`，断言 `CompileMQL` 返回 error 且消息含 "unsupported" 或 API 名。复用 S6a/S6d/S6g 断言模式。
- **S6r — registry 一致性**：`TestVM_API_TRUTH_1_TimeseriesNoSourceRegistryConsistency`：每 API `LookupAPI=StatusUnsupported` + `Reason` 非空 + `IsAPIImplemented=false` + `IsAPIUnsupported=true`。
- **S6s — 真实/venue 语义未误伤**：`TestVM_API_TRUTH_1_TimeseriesRealStillImplemented`：对 `iTickVolume`/`iVolume`/`iRealVolume`/`CopyRealVolume`/`CopyClose`/`CopyHigh`/`CopyLow`/`CopyOpen`/`CopyTime`/`CopyTickVolume`/`Bars`/`iBarShift`/`iHighest`/`iLowest`，`IsAPIImplemented=true` + `IsAPIUnsupported=false`（`iRealVolume`/`CopyRealVolume` 保留为 venue 事实语义，不得误删）。
- **验证**：`go test ./tools/mql2go/ -run TestVM_API_TRUTH_1` 全绿。

## 验收标准

- [ ] `go build ./...` 通过
- [ ] `go test ./tools/mql2go/` 全过（含批次1/2a/2b/2c + 2d S6q/r/s）
- [ ] `cd backend && go run ./tools/check-file-lines --strict` 零错误
- [ ] `gofmt` / `go vet` 零警告
- [ ] `go test -race -count=3 ./tools/mql2go/` 通过
- [ ] 对抗证明：注释 unsupportedSymbols 7 行 → S6q/S6r RED → 恢复 GREEN（附命令与输出）
- [ ] diff 通读无死代码 / TODO / 调试残留 / 范围外改动

## 施工完成自审（强制，D-012）

交付自报前必须完成并随报提交：
- [ ] 逐项重跑上方验收标准并贴真实输出
- [ ] 红队自审 diff 三问：更简等价方案 / 边界·nil·并发 / 逆向依赖或重复基础设施
- [ ] 自审发现的缺陷已修复至全绿（自报列出发现项+修复项）
- [ ] 无自审记录 = 复审直接退回

## 交付格式

自报必须按以下六段顺序（D-016，缺一 = 复审直接退回）：
1. **变更文件清单**：列出本任务改动的所有文件。
2. **S1-Sn 实现摘要**：每步落点对码（改了什么、在哪、是否符合派工单坐标）。
3. **对抗证明**：mutation RED → restore → GREEN 的命令与关键输出。
4. **机检门禁**：build / test / race×3 / vet / gofmt / check-lines / diff --check 逐项真实输出。
5. **范围确认**：仅改派工单列出的文件，无范围外改动。
6. **结束语**：`[施工完成:VM-API-TRUTH-1-批次2d] @<commit-hash>`（D-014，无此行 = 未交付，复审不启动）。

**停手等 Devin CLI 复审；不达标将由决策方开缺陷清单退回。勿部署，禁 `--no-verify`。**

## 复审注记（Devin CLI 设计复审记录，非施工内容）

- **VM-ENUM-NUMBERING-1 候选债（P2）**：prop-switch 假实现函数族的"自造紧凑编号 vs 真枚举"是跨文件模式——已证实 `AccountInfoInteger`(32/35/36 vs 2/5/6)、`AccountInfoString`(顺序错位)、`constants.go`(SO_*/INITIAL 互换)；疑似同类：`SymbolInfoDouble`/`SymbolInfoInteger`（真 ENUM_SYMBOL_INFO_* 是大枚举非紧凑 0-7）、`MarketInfo` MODE_*（与 MQL4 编号亦不符）。本债只做一次性全枚举对齐审计，勿分散到各批。
- **同族剩余假 API 队列**：account noop 整 API 8 个（`vm_builtin_impls.go:143-151`，归 2e/2f）+ `SymbolInfoDouble/Integer/String`/`MarketInfo` 的 prop 编号审计（VM-ENUM-NUMBERING-1）。
