# 施工方交接提示词（2026-08-10）

> **依据**：CLAUDE.md「Deployment (强制 — 禁止手动)」+ `tech-debt-registry.md` EXEC-PARAMS 验收行（✅ 权威 done，2026-08-10 审计方实测）+ `handover-audit-plan.md` 验收条目。
> **强制遵守** `docs/audits/builder-sop.md` 三铁律 + 验收 checklist（§200）：① one task = one scope（不扩大）；② 对抗证明（删关键一行→测试必红）；③ **红队自审**（完工后对着 §200 checklist + 验收 5 维过一遍，带着债提交=失败）；④ 不自行宣告 ✅，等审计方实测；⑤ 完工回填三层；⑥ REUSE preflight（cap.sh）。
> **节奏**：一次一个 task，提示词 1（部署）完成后做提示词 2（可选补测），完工回填 + 自审后再交付，不并行多任务。
> **当前状态**：EXEC-PARAMS（a1c88f33）已由审计方实测验收通过——5 项修订全部落地、9 测试全绿、对抗证明实测（删 `SetBarPrice(bar.Open)` → close 测试红；强制 spread → bar_close 测试红）、go build/tsc/vitest 140/140/vite build/check-file-lines 全绿。⚠️待Claude复审已解除，**部署已解锁**。工作区仅剩审计方 2 行验收 doc 未提交（handover + registry），代码零未提交，migrations 空。

---

## 提示词 1【🔴 部署 · 唯一方式，禁止手动替代】EXEC-PARAMS 部署 + 冒烟验证

```
依据 CLAUDE.md「Deployment (强制 — 禁止手动)」段 + registry EXEC-PARAMS 验收行（先完整读）。

背景：EXEC-PARAMS（signal_timing/fill_rule/simulation_mode 端到端接线 + StrictMode 引擎 bug 修复，
commit a1c88f33）已审计方实测验收通过，部署解锁。本次把该提交上线：后端 docker compose + 前端 docker cp。

范围（one task = one scope）：仅部署 + 冒烟验证。不改任何代码（含顺手修复/格式化）、不补测试
（另有提示词 2）、不动 docker-compose.yml/迁移/环境变量/其他服务。

步骤：
1. 先读 CLAUDE.md「Deployment」段 + builder-sop.md + registry EXEC-PARAMS 验收行。
2. 部署前检查（必做，任一不过停下确认）：
   - git status：确认零未提交代码文件；先 commit 审计方 2 行验收 doc（handover + registry，
     message: docs: EXEC-PARAMS 审计方验收记录）——使 build 上下文与 HEAD 一致。
   - git status backend/migrations/ 必须为空（未提交 .up.sql 会随 Docker build 自动执行；有 WIP 先移走）。
   - git log -1 确认 HEAD 含 a1c88f33（或其后的文档 commit）。
   - 记录当前运行中 backend 镜像 tag（rtk proxy docker compose images backend 或 docker inspect），
     写入回滚预案。
3. 后端部署（唯一方式）：rtk proxy docker compose build backend && rtk proxy docker compose up -d backend。
   确认容器 Healthy（rtk proxy docker compose ps）。
4. 前端部署（唯一方式）：cd frontend && npm run build（产出 dist/）→
   rtk proxy docker cp frontend/dist/. alphaforge-frontend:/usr/share/nginx/html/ →
   rtk proxy docker exec alphaforge-frontend nginx -s reload。
   ❌ 禁止宿主机 go build → docker cp；❌ 禁止在运行中容器内 go build / apk add build-base。
5. 冒烟验证（部署后，逐项记录结果）：
   - curl 容器 /healthz 与 /readyz 均健康。
   - 【对抗证明·validate 闸上线】API 级：startBacktestRun 带 executionConfig{simulation_mode:"DATASET"}
     → 预期 InvalidArgument（422，报 not yet implemented）；executionConfig{fill_rule:"limit"} 同样拒绝。
     若旧容器还在跑，这两个请求会被静默接受——拒绝 = 新代码真实上线。
   - 边界（防闸太紧误杀）：带合法请求 executionConfig{signal_timing:"next_bar_open", fill_rule:"bar_close",
     simulation_mode:"KLINE_RANGE"} → 不得误拒（验证不走到 validate 的 market-data 检查也没关系，
     只要不是 InvalidArgument 即可）。
   - 后端日志：docker logs backend 无 panic；启动日志正常。
   - 前端：浏览器硬刷新（Ctrl+Shift+R）打开策略工作区 → 回测参数 modal 出现 Execution Assumptions
     三选择器（Simulation Mode / Signal Timing / Fill Rule，DATASET 与 limit 选项置灰）。
     若无法连接 MT 账户做全流程，降级为产物验证：grep -c "next_bar_open" frontend/dist/assets/*.js > 0
     （旧 dist 无此产物），并在回填中注明降级。
   - 若条件具备（有连接账户+品种数据）：跑一次回测，ExecutionAssumptions 面板显示用户选择的
     signal_timing（非硬编码 next_bar_open）——诚实性验证。
6. 回滚预案（冒烟任一失败即回滚，不现场修代码）：
   - 后端：docker compose up -d 步骤 2 记录的旧镜像 tag。
   - 前端：docker cp 回旧 dist（若无备份，checkout 旧 commit 重新 build 再 cp）。
   - 回滚后报告审计方，不自行诊断修复。

对抗证明（部署场景，缺 = 任务判失败）：
- validate 闸生效：部署后 API 拒绝 DATASET/limit（旧行为是静默接受）——这是部署任务的"关键行"。
- 响应诚实性：回测 ExecutionAssumptions 显示用户选择值，非硬编码 next_bar_open。
- 前端新 UI 上线：dist grep "next_bar_open" 命中（旧 dist 无此产物）。

红队自审（部署任务级 edge cases，任一不过回去处理，不带债交付）：
- 部署前工作区零未提交代码文件（只允许刚 commit 的 2 行 doc）；多一个未提交文件=停下确认，不静默 build。
- build 失败 / pull 超时：先 rtk proxy docker compose config 验证配置；重试≤3 次，仍失败进回滚预案。
- 容器起不来（NATS/PG 依赖问题）：看 rtk proxy docker compose ps 依赖服务状态；不擅自重启/改依赖
  容器配置（超范围），报告等指示。
- 前端 dist 与 HEAD 匹配：build 前确认 HEAD 含 a1c88f33，避免用旧 dist 部署。
- 浏览器缓存：用户看到旧 UI 不是部署失败——提示硬刷新；不为此改 nginx 配置（超范围）。
- 克制：整个部署过程零代码改动；发现任何新问题只记录报告，不顺手修；不动 compose 文件/迁移/环境变量。
- 可演进性：回滚预案可复用（记录镜像 tag 成习惯），不写死单次操作。

Gate：部署 + 冒烟命令见步骤 5；无生产代码改动（doc-only commit），不需 go build/go test。

完工回填（不做 = 任务判失败）：
1. tech-debt-registry.md EXEC-PARAMS 条目追加部署行：部署时间 / 镜像 tag / 冒烟结果
   （validate 闸实测 + ExecutionAssumptions 实测或降级说明）。不改 ✅ 状态（已权威）。
2. handover-audit-plan.md 变更日志加一行（EXEC-PARAMS 部署完成 + 冒烟结果）。
3. commit（doc-only；message: docs: EXEC-PARAMS deployed + smoke results）。
4. 不自宣告完成——状态标 ⚠️待 Claude 复审，等审计方核对。
```

---

## 提示词 2【🟢 低优可选 · 非阻断】EXEC-PARAMS 补 2 个非阻断单测

```
依据 tech-debt-registry.md EXEC-PARAMS 验收行「非阻断观察 a」：worker 层回退派生
（strictMode→signalTiming）与 validate 拒绝逻辑无单测。提示词 1 部署完成后做。

范围（one task = one scope）：仅测试——backend/internal/connect/strategy/ 下新建/追加
backtest_worker_vm_test.go + strategy_backtest_validate_test.go。不动生产代码。

步骤：
1. REUSE preflight（动工前）：bash scripts/cap.sh validate；cap.sh backtestParams；
   查同包已有测试的 server 构造模式（如 backtest_execution_test.go 的 StrategyExecutionServer 零字段
   构造 + mock BarSource），复用不重造。PR 描述给 REUSE:/NEW:。
2. Test 1 TestBuildBacktestConfig_SignalTimingFallback（backtest_worker_vm_test.go）：
   - params{strictMode:false, signalTiming:""} + 最小 repository.BacktestRun → cfg.SignalTiming=="same_bar_close"。
   - params{strictMode:true, signalTiming:""} → cfg.SignalTiming=="next_bar_open"。
   - params{strictMode:true, signalTiming:"same_bar_close"} → 显式值优先，=="same_bar_close"（不回归）。
   - 顺带断言 cfg.FillRule=="bar_close" / cfg.SimulationMode=="KLINE_RANGE" 空值默认。
3. Test 2 TestValidateBacktestRequest_RejectsUnimplemented（strategy_backtest_validate_test.go）：
   - executionConfig{fill_rule:"limit"} → connect.CodeInvalidArgument。
   - executionConfig{simulation_mode:"DATASET"} → CodeInvalidArgument。
   - executionConfig{fill_rule:"bar_close", simulation_mode:"KLINE_RANGE"} → 不误拒。
   - req.Msg.GetExecutionConfig()==nil → 不 panic，正常走后续（marketDataRepo nil 时提前 return nil
     的短路点可隔离本测试，无需 mock 数据）。
   - 大小写精确性：fill_rule:"LIMIT" 不误拒（按当前实现精确匹配语义，不做模糊宽容）。

对抗证明（缺 = 任务判失败）：
- 删 validate 中 limit 拒绝分支 → Test 2 必红。
- 删 buildBacktestConfig 中 signalTiming 派生行 → Test 1 必红。

红队自审（任务级 edge cases）：
- nil executionConfig / 空字符串值合法不拒绝 / 大小写边界（"LIMIT"≠"limit"）/ 显式值优先于派生 /
  测试数据确定性（固定 epoch，禁 time.Now）/ 不 mock 过度（真实 server struct 零字段即可）/
  克制（不动生产代码，不顺手重构）。
- 测试质量：覆盖"防闸太紧"（合法值不误拒）与"防闸太松"（非法值必拒）双向。

Gate：go build ./... + go test ./internal/connect/strategy/... + go run ./tools/check-file-lines --strict。

完工回填（不做 = 任务判失败）：
1. tech-debt-registry.md EXEC-PARAMS 条目追加补测行（测试名 + 对抗证明结果）。
2. handover-audit-plan.md 变更日志加一行。
3. commit（test + doc 一并）。
4. 不自宣告完成——标 ⚠️待 Claude 复审，等审计方实测。
```
