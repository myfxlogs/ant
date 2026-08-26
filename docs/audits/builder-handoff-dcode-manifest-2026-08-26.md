# 施工交接：D-CODE-HYGIENE-001 逐文件 manifest 补齐（验收收口）

> 审计方（Claude）2026-08-26 派工，施工方（GLM-5.2）任务：**纯文档任务，只做本文件 S1–S4**。不重新审计、不自由发挥、不扩大范围。
> **基线 HEAD=`34e983a6`，工作树干净**。开工前先核验 SSOT 指纹（见下），再整读本文件 + `AGENTS.md §0` + registry「D-CODE-HYGIENE-001」BEGIN/END 区块（H1/H2/H3）+ 交付回填 + GPT-5.6 两次复审 + 当前问题总览。
> **SSOT 核验**：本文件第一部分是 registry `D-CODE-HYGIENE-001-MANIFEST:BEGIN/END` 区块（SSOT 权威文本）的只读副本。对 registry 区块执行
> `sed -n '/^<!-- D-CODE-HYGIENE-001-MANIFEST:BEGIN -->$/,/^<!-- D-CODE-HYGIENE-001-MANIFEST:END -->$/p' docs/audits/tech-debt-registry.md | sed '1d;$d' | sha256sum`
> 必须等于 `86b89c74d3871249501b822d0ce10c909aa71c48e07331adf52d1c60224269d3`。不匹配立即停止返回 Claude。**任何修订只改 registry 区块并重算指纹，本文件不单独修订。**

---

<!-- D-CODE-HYGIENE-001-MANIFEST:BEGIN -->
## D-CODE-HYGIENE-001-MANIFEST 施工提示词（逐文件 manifest 补齐；施工者 GLM-5.2，设计/验收者 Claude）

> **先整读** `AGENTS.md §0`、D-CODE-HYGIENE-001 BEGIN/END 区块（H1/H2/H3）、交付回填、GPT-5.6 两次独立复审、当前问题总览、本节和开工包 `docs/audits/builder-handoff-dcode-manifest-2026-08-26.md`，再动手。只做本节 S1–S4。

### 立项背景（证据链）

GPT-5.6 两次独立复审均 ❌未验收（见 registry D-CODE 交付回填之后）。第二轮后**唯一剩余阻断**（registry 原文）：

> H2 明确要求 registry 对每一个新文件注明"来源文件、抽出的责任、REUSE/NEW、行为回归命令"；当前回填仅有 `120 新文件`总数和 `pipeline_state_callbacks.go` 个别说明，没有完整 manifest。`grep` 检索 registry/audits 也未找到其余逐文件映射。该文档缺口属于 T5 的"文档失配"，不是可由 build/test 代替的项目。

**下一步（registry 原文）**：仅补充逐文件 manifest 并逐项对应 H1 来源、语义责任、REUSE/NEW 和 package 回归命令；不得借机修改代码、AGENTS/CLAUDE、proto、schema 或 VM。补齐后停手等待 Claude 再次复审；不提交、不部署。

本任务 = **纯文档任务**：产出 H2 要求的逐文件 manifest，把 D-CODE 交付证据补完整，让验收可以从 registry 独立确认每个新文件都是 H1 的纯语义拆分。

### 🔴 绝对边界（违反 = 直接判失败）

1. **只允许修改** `docs/audits/tech-debt-registry.md`（追加 manifest 子区块）与 `docs/audits/handover-audit-plan.md`（顶部追加一行）。**禁止改动任何 `backend/**` 代码、`AGENTS.md`、`CLAUDE.md`、proto、schema、VM、check-file-lines 工具**；发现工作树有非本任务的改动，报告 Claude，不得处理。
2. **禁止 commit / push / deploy**——manifest 只落在工作树，由 Claude 复审验收后统一提交。禁止 `--no-verify`。
3. 禁止为对齐数字改写已有回填/复审内容；实测与回填声明的数字差异必须在 manifest 中**逐文件披露原因**（append-only，不覆盖历史）。
4. 收工只显式 `git add` 本任务两个文档；禁止 `git add -A`／`git add .`（本仓多 agent 并发）。

### 施工步骤（目标 + 精确坐标）

- **S1（清单核对）**：运行
  `git show acaa86db --diff-filter=A --name-only --format="" | grep '\.go$'`
  得到 acaa86db 新增 Go 文件清单（审计方实测 **121 个：70 实现 + 51 测试**；另有非 Go 的 `docs/audits/vm-adversarial-proofs.md` 1 个）。与交付回填声称的 **120（68 实现 + 52 测试）** 逐项核对，**定位全部差异文件**（多 2 个实现、少 1 个测试），并在 manifest 开头披露差异结论——哪些文件**不属于** D-CODE 拆分产物（候选：round4/round5 VM 对抗测试、D-VM-LIVE-001 相关文件、pre-existing 工作树遗留），其归属是什么。
- **S2（逐文件 manifest）**：对每个**判定为 D-CODE 拆分新增**的文件（来源为 H1 65 文件清单内文件的直接拆分/改名产物）填写 H2 四项，逐行一条：
  1. **来源文件**——H1 清单中的文件路径（必须可定位；改名/拆分的给出对应关系证据，不得写"由多个文件合并"之类不可核验描述）；
  2. **抽出的责任**——从来源文件移出的具体语义（函数名/测试名列表，非泛泛描述）；
  3. **REUSE/NEW**——纯拆分移动既有符号 = `REUSE:`（列出符号）；因拆分新建的 helper/测试 = `NEW:`（说明理由）；
  4. **行为回归命令**——该文件所属 package 的回归测试命令（如 `go test ./internal/connect/strategy -count=1`），与拆分前该 package 的测试集对齐。
- **S3（非 D-CODE 文件披露）**：S1 清单中判定不属于 D-CODE 的文件（来源不在 H1 且非 H1 拆分产物），单独一小节列出并注明归属（如"round5 VM 对抗测试，属 D-VM-LIVE-001 范围"、"pre-existing 工作树遗留随 acaa86db 一并入库"），**不得混入 manifest 主体冒充拆分产物**。
- **S4（回填）**：manifest 以子区块追加到 registry「D-CODE-HYGIENE-001 交付回填」之后（append-only，不改动已有内容）；`handover-audit-plan.md` 变更日志顶部追加一行；任务状态保持 `⚠️待Claude复审`，不得自标 ✅done。

### 红队自审（施工后切换怀疑者视角，逐条书面回答）

1. manifest 是否覆盖了全部 D-CODE 拆分新增文件？实测 121 vs 声称 120 的差异是否逐文件解释清楚、结论闭合？
2. 每一条"来源文件"是否真的可定位到 H1 65 文件清单（独立核验者只看 registry 也能验证）？
3. 每一条"行为回归命令"是否真实覆盖该文件全部测试？拆分前后该 package 测试集是否等价？
4. 有没有把非 D-CODE 文件（round4/5 对抗测试、pb.go churn、docs）混入 manifest 主体？S3 披露是否完整？
5. 有没有为了"看起来完整"编造来源/责任或复制粘贴其他条目的内容？每一条的"抽出的责任"必须能与 git show 实际内容对上。

### 验收门禁（Claude 复审时逐条独立核验）

- 文档差异：`git diff --stat docs/audits/` 仅 manifest 子区块 + 两个变更日志行；`git diff --stat` 无任何 `backend/` 改动。
- 数字自洽：manifest 条目数 = 披露的 D-CODE 新增文件数；与 120/121 的差异结论闭合（每个差异文件都有交代）。
- 抽检：Claude 随机抽 5 条 manifest，独立 `git show acaa86db -- <文件>` / 与 H1 来源文件 diff，核验来源与责任属实。
- 无代码改动、无 commit：工作树中 manifest 未提交（Claude 验收后统一提交）。

### 回填与收尾

manifest 追加 + handover 一行；**状态填 `⚠️待Claude复审`，不得自标 ✅done**。

> **勿部署、勿 push、勿 commit，停手等 Claude 复审。禁止 `--no-verify`。收工只显式 `git add` 本任务涉及的两个文档，禁止 `git add -A`／`git add .`（本仓多 agent 并发）。**
<!-- D-CODE-HYGIENE-001-MANIFEST:END -->

---

# 附：工作材料（非 SSOT，供施工使用，不作为验收依据）

## A. H1 65 文件清单（registry「D-CODE-HYGIENE-001」BEGIN/END 区块原文，2026-08-25 baseline 实测 warning 文件）

```text
backend/internal/repository/wallet_repo.go
backend/tools/mql2go/vm_builtin_string.go
backend/internal/connect/user/share_service.go
backend/internal/mdgateway/adapter/mt4/profit.go
backend/internal/repository/ai_gateway_repository.go
backend/cmd/coldsign-gui/main.go
backend/internal/risk/gate_test.go
backend/internal/connect/strategy/backtest_worker_vm.go
backend/internal/connect/strategy/live_runner.go
backend/tools/mql2go/compile_interp.go
backend/tools/mql2go/rule_engine.go
backend/internal/marketplace/decay_detector.go
backend/internal/mthub/service_orders_unit_test.go
backend/internal/connect/strategy/strategy_execution_handler.go
backend/internal/mdgateway/adapter/mt5/orders.go
backend/internal/marketplace/publish.go
backend/internal/sweep/sweep_test.go
backend/internal/execalgo/algo_test.go
backend/internal/service/subscription_service.go
backend/tools/mql2go/bytecode_cache_unmarshal.go
backend/tools/mql2go/header_parser.go
backend/tools/mql2go/honesty_audit_test.go
backend/internal/marketplace/service_subscription.go
backend/tools/mql2go/compile.go
backend/internal/connect/strategy/trade_fields_invariant_test.go
backend/internal/marketplace/quality.go
backend/internal/mdgateway/adapter/mt4/orders.go
backend/internal/connect/strategy/schedule_event_test.go
backend/internal/marketplace/live_performance.go
backend/internal/risk/rules.go
backend/tools/mql2go/compile_interp_helpers.go
backend/internal/connect/marketplace/marketplace_test.go
backend/internal/connect/strategy/live_context.go
backend/internal/marketplace/strategy_optimizer.go
backend/cmd/server/pipeline.go
backend/internal/marketplace/money_flow_integration_test.go
backend/internal/sweep/worker.go
backend/tools/mql2go/vm_builtin_trade.go
backend/internal/chain/tron_grid.go
backend/internal/mthub/service_orders.go
backend/internal/connect/strategy/session_registry.go
backend/internal/service/systemai/chat_stream.go
backend/internal/connect/strategy/strategy_experiment_worker.go
backend/internal/connect/system/mthub_service_integration_test.go
backend/tools/mql2go/vm_execute.go
backend/internal/connect/strategy/live_dispatch.go
backend/internal/connect/strategy/mutation_coordinator_test.go
backend/internal/connect/strategy/strategy_schedules.go
backend/internal/mthub/service_coverage_test.go
backend/tools/mql2go/compile_py_expr.go
backend/internal/mdgateway/adapter/mt4/mt4_test.go
backend/internal/mthub/service.go
backend/tools/mql2go/compile_py_test.go
backend/internal/connect/gateway/ai_gateway_handler.go
backend/internal/connect/strategy/live_diag_truth_test.go
backend/internal/knowledgebase/service.go
backend/internal/connect/strategy/schedule_execute.go
backend/internal/connect/strategy/trade_barrier.go
backend/internal/mdgateway/adapter/mt5/mt5_test.go
backend/internal/mdgateway/pure_test.go
backend/internal/sweep/builder.go
backend/tools/mql2go/vm_audit_test.go
backend/internal/connect/ai/code_assist_handler.go
backend/internal/connect/strategy/schedule_hotloop_test.go
backend/tools/mql2go/builtins_registry.go
```

## B. acaa86db 新增文件实测清单（审计方 2026-08-26 提取：`git show acaa86db --diff-filter=A`）

**121 个 Go 文件（70 实现 + 51 测试）+ 1 个非 Go（docs/audits/vm-adversarial-proofs.md）**：

```text
# 实现（70）：
backend/cmd/coldsign-gui/sign_dialog.go
backend/cmd/server/pipeline_state_callbacks.go
backend/internal/chain/tron_grid_queries.go
backend/internal/connect/ai/code_assist_handler_extract.go
backend/internal/connect/gateway/ai_gateway_handler_models.go
backend/internal/connect/gateway/ai_gateway_handler_usage.go
backend/internal/connect/strategy/backtest_worker_vm_response.go
backend/internal/connect/strategy/live_context_build.go
backend/internal/connect/strategy/live_context_enums.go
backend/internal/connect/strategy/live_dispatch_paper.go
backend/internal/connect/strategy/live_runner_loop.go
backend/internal/connect/strategy/schedule_execute_build.go
backend/internal/connect/strategy/session_registry_active.go
backend/internal/connect/strategy/session_registry_queries.go
backend/internal/connect/strategy/strategy_execution_handlers.go
backend/internal/connect/strategy/strategy_experiment_worker_validation.go
backend/internal/connect/strategy/strategy_schedules_validation.go
backend/internal/connect/strategy/trade_barrier_wait.go
backend/internal/connect/strategy/vm_live_helpers.go
backend/internal/connect/strategy/vm_live_validators.go
backend/internal/connect/user/share_service_metrics.go
backend/internal/knowledgebase/service_loader.go
backend/internal/marketplace/decay_detector_batch.go
backend/internal/marketplace/live_performance_collector.go
backend/internal/marketplace/live_performance_recompute.go
backend/internal/marketplace/publish_cache.go
backend/internal/marketplace/publish_query.go
backend/internal/marketplace/quality_validation.go
backend/internal/marketplace/service_subscription_loop.go
backend/internal/marketplace/service_subscription_renewal.go
backend/internal/marketplace/strategy_optimizer_publish.go
backend/internal/marketplace/strategy_optimizer_query.go
backend/internal/mdgateway/adapter/mt4/orders_events.go
backend/internal/mdgateway/adapter/mt4/orders_queries.go
backend/internal/mdgateway/adapter/mt4/profit_fetch.go
backend/internal/mdgateway/adapter/mt5/orders_queries.go
backend/internal/mthub/service_brokers.go
backend/internal/mthub/service_orders_helpers.go
backend/internal/mthub/service_orders_oms.go
backend/internal/mthub/service_queries.go
backend/internal/repository/ai_gateway_repository_usage.go
backend/internal/repository/wallet_repo_tx.go
backend/internal/risk/rules_advanced.go
backend/internal/service/subscription_service_billing.go
backend/internal/service/systemai/chat_stream_helpers.go
backend/internal/sweep/builder_bundle.go
backend/internal/sweep/worker_export.go
backend/tools/mql2go/builtins_registry.go
backend/tools/mql2go/builtins_registry_ext.go
backend/tools/mql2go/bytecode_cache_unmarshal.go
backend/tools/mql2go/bytecode_cache_unmarshal_io.go
backend/tools/mql2go/bytecode_validate.go
backend/tools/mql2go/compile_expr_helpers.go
backend/tools/mql2go/compile_helpers.go
backend/tools/mql2go/compile_interp_decls.go
backend/tools/mql2go/compile_interp_expr_helpers2.go
backend/tools/mql2go/compile_interp_funcs.go
backend/tools/mql2go/compile_interp_helpers.go
backend/tools/mql2go/compile_interp_stmts.go
backend/tools/mql2go/compile_py_expr_ops.go
backend/tools/mql2go/header_parser_extract.go
backend/tools/mql2go/interp/analyze_walk.go
backend/tools/mql2go/interp/constants_colors.go
backend/tools/mql2go/interp_runner_events.go
backend/tools/mql2go/rule_engine_rules.go
backend/tools/mql2go/vm_builtin_array_ops.go
backend/tools/mql2go/vm_builtin_math_basic.go
backend/tools/mql2go/vm_builtin_trade_mql5.go
backend/tools/mql2go/vm_builtin_trade_props.go
backend/tools/mql2go/vm_execute_handlers.go

# 测试（51）：
backend/internal/connect/marketplace/marketplace_edge_test.go
backend/internal/connect/strategy/live_diag_truth_lifecycle_test.go
backend/internal/connect/strategy/mutation_coordinator_labels_test.go
backend/internal/connect/strategy/mutation_coordinator_recovery_test.go
backend/internal/connect/strategy/schedule_event_launch_test.go
backend/internal/connect/strategy/schedule_hotloop_cache_test.go
backend/internal/connect/strategy/trade_fields_build_test.go
backend/internal/connect/strategy/trade_fields_helpers_test.go
backend/internal/connect/strategy/trade_fields_side_test.go
backend/internal/connect/strategy/vm_api_truth3_round4_test.go
backend/internal/connect/strategy/vm_api_truth3_round5_test.go
backend/internal/connect/strategy/vm_api_truth3_test.go
backend/internal/connect/strategy/vm_trade_context3_test.go
backend/internal/connect/strategy/vm_trade_context6_round4_test.go
backend/internal/connect/strategy/vm_trade_context6_round5_test.go
backend/internal/connect/strategy/vm_trade_context6_test.go
backend/internal/connect/system/mthub_service_integration_events_test.go
backend/internal/execalgo/algo_helpers_test.go
backend/internal/execalgo/algo_schedule_test.go
backend/internal/marketplace/money_flow_integration_lifecycle_test.go
backend/internal/mdgateway/adapter/mt4/mt4_bars_test.go
backend/internal/mdgateway/adapter/mt4/mt4_connection_test.go
backend/internal/mdgateway/adapter/mt4/mt4_streams_test.go
backend/internal/mdgateway/adapter/mt4/mt4_trading_test.go
backend/internal/mdgateway/adapter/mt5/mt5_trading_test.go
backend/internal/mdgateway/pure_metrics_test.go
backend/internal/mdgateway/pure_session_test.go
backend/internal/mthub/service_coverage_gate_test.go
backend/internal/mthub/service_coverage_orders_test.go
backend/internal/mthub/service_coverage_session_test.go
backend/internal/mthub/service_orders_events_test.go
backend/internal/risk/gate_helpers_test.go
backend/internal/risk/gate_margin_test.go
backend/internal/sweep/sweep_reconfirm_test.go
backend/tools/mql2go/compile_py_features_test.go
backend/tools/mql2go/compile_py_mapping_test.go
backend/tools/mql2go/compile_py_operators_test.go
backend/tools/mql2go/compile_py_rejection_test.go
backend/tools/mql2go/honesty_audit_probes_test.go
backend/tools/mql2go/vm_audit_builtins_test.go
backend/tools/mql2go/vm_audit_cache_test.go
backend/tools/mql2go/vm_audit_control_flow_test.go
backend/tools/mql2go/vm_audit_failclosed_test.go
backend/tools/mql2go/vm_audit_semantics_test.go
backend/tools/mql2go/vm_audit_test.go
backend/tools/mql2go/vm_audit_timeseries_test.go
backend/tools/mql2go/vm_audit_trade_context_test.go
backend/tools/mql2go/vm_audit_trade_test.go
backend/tools/mql2go/vm_cache_integrity5_test.go
backend/tools/mql2go/vm_compiler_semantics4_round4_test.go
backend/tools/mql2go/vm_compiler_semantics4_test.go

# 非 Go（1）：
docs/audits/vm-adversarial-proofs.md
```

## C. 数字核对表（S1 必做，逐文件披露差异）

| 口径 | 实现 | 测试 | 合计 | 备注 |
|---|---|---|---|---|
| 交付回填声称（registry :322） | 68 | 52 | 120 | "全部为 H1 清单内文件或直接从 H1 文件抽出的新文件" |
| 审计方实测 acaa86db 新增 Go | 70 | 51 | 121 | 本文件 B 节；+1 非 Go docs |
| 差异 | +2 | -1 | +1 | **每个差异文件都要定位并披露归属** |

已知参考（不构成结论，需施工方独立核实）：
- B 节实现中有 `vm_live_validators.go` / `strategy_execution_handlers.go`——H1 清单无同名文件；前者为 D-VM-LIVE-001-P1 的实现载体（registry 记录其 validateExecuteLiveRequestMode 为 P1 产物），后者由 H1 的 `strategy_execution_handler.go` 改名/拆分，两者归属需按 `git log --follow` / 与 H1 来源文件 diff 判定。
- B 节测试中有 round4/round5 命名文件（vm_trade_context6_round5_test.go 等）——D-VM-LIVE-001 范围重定（registry 范围重定段）将 round5 VM 从 D-CODE 划出；P1B 还删除了该文件内的 2 个测死代码测试。这些文件是否计入 D-CODE manifest 由 S1 核对结论决定，S3 必须披露归属。
- `docs/audits/vm-adversarial-proofs.md` 为非 Go 审计文档，不计入 manifest 主体，S3 一行披露即可。

## D. manifest 条目模板（S2 输出格式）

追加到 registry「D-CODE-HYGIENE-001 交付回填」之后：

```markdown
### D-CODE-HYGIENE-001 逐文件 manifest（2026-08-26 补齐；施工方 GLM-5.2）

**S1 清单核对结论**：acaa86db 新增 Go 文件实测 121（70 实现 + 51 测试），回填声称 120（68+52）；差异 = <逐文件列出>；判定 D-CODE 拆分新增 = <N> 个，非 D-CODE = <M> 个（归属见 S3）。

| 新文件 | 来源 H1 文件 | 抽出的责任（函数/测试名） | REUSE/NEW | 行为回归命令 |
|---|---|---|---|---|
| ... | ... | ... | ... | ... |

### 非 D-CODE 新增文件披露（S3）
| 文件 | 归属 | 依据 |
|---|---|---|
| ... | ... | ... |
```

规则提醒：每行必须可独立核验；"来源 H1 文件"不得写多个文件合并；"抽出的责任"列出具体符号；"行为回归命令"必须是真实可运行的 `go test` 命令。
