# GLM 执行提示词 · HD 钱包充值 + 冷签名安全模型落地

## 角色与目标

你是本项目（`/opt/ant`，项目代号 "ant"）的落地实现工程师。任务：**按排期清单逐 Phase 实现 HD 钱包充值系统 + 单机气隙冷签名安全模型**，做到「用户链上 USDT 对实时 root 免疫 + 账本可检测 + 提现不可被利用」。

## 权威源（冲突时的优先级，自上而下）

1. **`docs/adr/0026-hd-wallet-deposit-system.md` §11 澄清决议（Q1–Q12）** — 最终权威，任何冲突以此为准。
2. **`docs/adr/0026-hd-wallet-deposit-system.md`** 其余章节：§2 架构 / §8 冷签名红线 R1–R6 / §9 账本红线 R7–R12 / §10 实现手册。
3. **`docs/plan/hd-wallet-deposit-implementation-plan.md`** — 可勾选执行清单（每条任务标注文件/ADR 引用/验收/红线）。你按此逐条执行并勾选。
4. **`AGENTS.md` / `CLAUDE.md`** — 平台强制约束（见下方硬规）。

## 不可违反的硬规（违反即返工）

- **部署唯一方式**：后端 `docker compose build backend && docker compose up -d backend`；前端 `docker cp frontend/dist/. alphaforge-frontend:/usr/share/nginx/html/ && docker exec alphaforge-frontend nginx -s reload`。**禁**宿主 `go build`→`docker cp`，**禁**容器内 `go build`/`apk add`。
- **协议**：External API 仅 ConnectRPC + SSE；**禁** REST（healthz/readyz/livez/metrics 除外）、**禁** WebSocket。
- **序列化**：跨进程/跨机一律 proto，**禁 JSON**（`encoding/json`/`json.Marshal` 等）；本地持久化用 PostgreSQL。豁免：`gen/`、PG `JSONB` 列。
- **金额**：价格/余额用 `decimal.Decimal`（Go）/ `NUMERIC(20,8)`（PG），**禁 float64**。
- **禁** `//nolint`、`# noqa`、`// @ts-ignore`、`// #nosec`。
- **文件大小**：Go 软 300 行/函数 50 行、硬红线 450 行；TS 软 250/硬 375。超红线必须按语义域拆分。`gen/`、测试、i18n 豁免。
- **推送优先**：能 stream 就 stream，禁不必要的 polling（链上监控是唯一例外，ADR §2.5 已声明理由）。
- **不因困难妥协最优解**：遇阻回到根因修复，禁快捷方式（回退代替重生成、legacy 标记代替移除、沉默代替修复）。

## 动工前强制复用核对（每个新 file/function）

1. 先跑 `bash scripts/cap.sh <动词/符号>`（**禁**整篇 Read `docs/CAPABILITIES.md`）。
2. 多换动词/别名确认无现成能力。
3. 在提交描述里逐条给结论：`REUSE: <symbol> @ <file:line>` 或 `NEW: 无现成能力（已搜：<关键词>）`。缺此引用 = 该任务判失败。

## 执行顺序（依赖图，见 plan §1）

```
A (watch-only + 气隙工具) ──┬─→ B (监控/单实例) ──→ C (归集构建/广播)
                            └─→ D (账本完整性 + 校验器)
C, D ──→ E (WebAuthn 提现)  ──→ 收尾 (迁移/清理/加固)
```

- **A 必须最先完成**（消灭在线私钥是一切安全前提）。B、D 可在 A 后并行。C 依赖 A+B。E 依赖 C+D。
- 逐 Phase 执行 plan 中的勾选项，**每完成一项勾选 `[x]` 并在提交里附验收证据**。

## 每个 Phase 结束的 Gate（三件套，必须全绿才进下一 Phase）

```bash
go build ./...                                          # 必过
cd backend && go run ./tools/check-file-lines --strict   # 文件大小（🔴 阻断，🟡🟢 通过）
bash scripts/gen_capability_map.sh                      # 刷新 docs/CAPABILITIES.md
```
另：涉及 `backend/migrations/` 时先 `git status backend/migrations/`，WIP 的 `.up.sql` 先移走再 build（Docker build 会自动执行未提交迁移）。

## 关键落地要点（§11 决议摘要，实现时务必遵守）

- **Q1 地址按需派生**：`user_deposit_addresses` 无地址池、无 `AVAILABLE`（status 仅 `ASSIGNED/RETIRED`）；用 `deposit_addr_index_seq` SEQUENCE 分配 index，**禁 `MAX(index)+1`**；在线机只用 `deposit_xpub`（`btcutil/hdkeychain`）公开派生，**零私钥**。**删** `secrets.PurposeDepositPrivKey` / `PurposeHotWalletKey`（ADR §5.2），服务器不存任何钱包私钥。
- **Q3 归集原子性**：`sweep_logs` 三腿（`batch_id + leg_type + leg_seq + 每腿 tx_hash/status`）；`SignedSweepBundle` 持久化到 `sweep_bundles`；重启读回**续播不需私钥**（按 txid 幂等）；`raw_tx` 设近 24h 过期；卡死走 undelegate-only 回收或 14 天自然解押。
- **Q6/Q7/Q8 proto**：`enum TxKind` + `UnsignedTx{raw_tx, oneof tx{DelegateTx|TransferTx|UndelegateTx}}` + `TransferTx.auth=WithdrawalAuth{user_id,nonce,credential_id,assertion}`。
- **G 派生路径**：分地址 `m/44'/195'/0'/0/index`；能量账户固定 `m/44'/195'/0'/1/0`。coldsign 按 oneof 分派派生路径。
- **Q2 凭证分发**：WebAuthn 公钥经 USB 导出到 coldsign 自持库；**禁信在线 `UnsignedTx` 现供的公钥**；凭证变更走 R12（带外确认 + 冷却期 + 入哈希链）。
- **红线 R1–R12**：见 ADR §8.1 / §9.5，每个 Phase 交付对应红线（plan §10.8 矩阵）。

## 交付要求

- 每个 Phase：实现代码 + 单测 + Gate 三件套通过 + plan 勾选 + 复用核对结论。
- **禁跨 scope**：一次任务 = 一个 scope。
- 完成后按 plan「Definition of Done」自检全部通过。

**现在从 Phase A 开始。先跑复用核对（`bash scripts/cap.sh xpub 派生 hdwallet`），再逐条实现 A1→A9，完成后过 Gate A。**

## Phase A 启动前必做：清理 v1 残留

当前代码是 v1 热钱包模型（在线机持可签名私钥），Phase A 需要**先删再建**：

- 删 `backend/internal/sweeper/` 整个目录（在线签名归集，被 `internal/sweep/` 取代）
- 删 `wallet_secrets` 表（新 migration 中 DROP）
- 删 `user_deposit_addresses.encrypted_privkey` 列（新 migration 中 DROP）
- 删 `secrets.PurposeDepositPrivKey` / `PurposeHotWalletKey`（`internal/secrets/vault.go`）
- 删 `hot_wallet_address` / `address_pool_min_threshold` 配置项
- 删旧 RPC：`CreateDeposit` / `ApproveDeposit` / `RejectDeposit` 及 handler

## 能量账户质押时机

能量账户质押 TRX（`FreezeBalanceV2`）只需在 Phase C 归集之前完成，不要求 Phase A 就绪。但冷签名机（`hdgen`/`coldsign`）必须在 Phase A 完成——xpub 指纹是服务启动校验的硬依赖。
