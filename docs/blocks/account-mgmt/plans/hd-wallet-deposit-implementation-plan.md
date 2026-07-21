# HD 钱包充值 + 系统钱包 · 落地排期清单

> 权威依据：`docs/adr/0026-hd-wallet-deposit-system.md`（§2 架构 / §8 冷签名红线 R1–R6 / §9 账本红线 R7–R12 / §10 实现手册 / **§11 澄清决议 Q1–Q12**）。
> 冲突时以 ADR §11 为最终权威。
> 本清单是可勾选执行版；每条任务标注 **文件**、**ADR 引用**、**验收**、**红线**。GLM 逐 Phase 执行，Phase 内可并行，Phase 间有依赖。

## 0. 目标与边界（执行前必读）

- **达成即最优（单机约束下）**：在线机零资金私钥 → 用户链上 USDT 对实时 root 免疫；账本哈希链 + 偿付能力校验 + WebAuthn 提现 → 出口不可被利用。
- **已知残余（第一性下的单机天花板，非缺陷）**：持续 root 下「内部账本数字」与「提现目的地」不可绝对保证，仅做到**可检测 + 有界 + 不丢真钱**（ADR §8.2）。
- **合规**：proto-only 跨机交换、decimal 计价、无 REST endpoint（出站调用除外）、文件行数限制、复用优先（`bash scripts/cap.sh`）。

## 1. 依赖图

```
A (watch-only + 气隙工具) ──┬─→ B (监控/单实例) ──→ C (归集构建/广播)
                            └─→ D (账本完整性 + 校验器)
C, D ──→ E (WebAuthn 提现)
```

- A 必须最先完成（消灭在线私钥是一切安全前提）。
- B、D 可在 A 后并行。
- C 依赖 A（coldsign）+ B（地址状态）。
- E 依赖 C（广播）+ D（冻结/账本）。

---

## Phase A · 在线转 watch-only + 气隙工具（R1 R2 R4 R5）

**前置**：`bash scripts/cap.sh xpub 派生 hdwallet` 确认无现成能力。

- [ ] **A1 xpub 派生库** — 新增 `internal/hdwallet/xpub.go`：`DeriveAddress(xpub,index)`（非硬化 CKDpub → TRON 地址，keccak256 + 未压缩公钥）、`XpubFingerprint`。依赖 `btcutil/hdkeychain`。ADR §10.1。
  - 验收：BIP44 测试向量 xpub+index→已知地址；单测覆盖。
- [ ] **A1b 按需派生 + index 分配（Q1）** — `user_deposit_addresses` 无地址池、无 `AVAILABLE`（status 仅 `ASSIGNED/RETIRED`）；新 migration 建 `deposit_addr_index_seq` SEQUENCE；`GetDepositAddress` 幂等：查已有→无则 `nextval(seq)` 取 index + xpub 派生 + `INSERT ... status='ASSIGNED'`。**禁 `MAX(index)+1`**。ADR §2.3/§5.3/§11 Q1。
  - 验收：并发请求不撞 index/地址；重复请求返回同一地址。
- [ ] **A2 删除在线私钥路径（R1）** — 删 `user_deposit_addresses.encrypted_privkey`（新 migration）、`model.DepositAddress.EncryptedPrivkey`、`secrets.PurposeDepositPrivKey`；重写 `deposit_service.go:107-167` 的 `ImportDepositAddresses` 为仅校验地址格式+index（无解密）。ADR §9.6/§10.1。
- [ ] **A3 AddressBatch proto 去私钥** — `proto/ant/v1` 里 `AddressBatch.entries` 移除 `encrypted_privkey` 字段，改为 `(index, address)`；`buf generate`。ADR §10.2。
  - 验收：`gen/proto` 无 `EncryptedPrivkey`。
- [ ] **A4 启动指纹校验（R5）** — `cmd/server` 装配处校验 `XpubFingerprint(cfg.DepositXpub)==cfg.DepositXpubFingerprint`，不符 `log.Fatal`。ADR §10.1。
- [ ] **A5 交易包 proto（R3, Q6/Q7/Q8）** — `proto/ant/v1`：`enum TxKind` + `UnsignedTx{raw_tx, oneof tx{DelegateTx|TransferTx|UndelegateTx}}` + `TransferTx.auth=WithdrawalAuth{user_id,nonce,credential_id,assertion}` + `UnsignedSweepBundle{txs, built_at_ms} / SignedTx / SignedSweepBundle`（ADR §8.3）；`buf generate`。
  - 验收：编译期类型检查生效；`DelegateTx/UndelegateTx` 用 `energy_account` 而非 `derivation_index`。
- [ ] **A6 `cmd/hdgen`（气隙）** — 生成助记词→seed→account xpub→导出 `xpub + fingerprint`（proto，无私钥）。地址以在线 xpub 按需派生为**权威**；地址表仅**可选**导出供一次性交叉核对（Q1）。ADR §10.2/§11 Q1。
- [ ] **A7 `cmd/coldsign`（气隙，R2 R4）** — 读 `UnsignedSweepBundle`；seed 启动手输不落盘；**按 oneof 派生私钥（Q7/G）**：`TransferTx`→`m/44'/195'/0'/0/derivation_index`，`DelegateTx/UndelegateTx`→能量账户固定路径 `m/44'/195'/0'/1/0`；**白名单（R4）**：`TransferTx` 且无 `auth`（归集）→ `to` 必须==cold_wallet_address 否则 abort；`TransferTx` 有 `auth`（提现）→ 验 WebAuthn + dest∈白名单 + 限额；打印核对；签名→`SignedSweepBundle`。ADR §10.2/§11 D/G。
  - 验收：非冷地址归集目的地 → 拒签且退出码≠0；`expected_txid` 与链上回执一致。
- [ ] **A8 CI 零私钥断言（R1, F）** — `scripts/` 加检查：**在线侧** `internal/{hdwallet,service,repository,chain,connect,sweep}` + `cmd/server/` grep 不到 `PrivateKey|Mnemonic|Seed|Decrypt`；**离线工具** `cmd/hdgen/`、`cmd/coldsign/` 豁免。ADR §10.1/§11 F。
- [ ] **A9 配置项** — `system_config` 增 `deposit_xpub / deposit_xpub_fingerprint / cold_wallet_address / energy_account_address / sweep_alert_threshold / dem_factor / energy_buffer_percent`（ADR §5.6）。
- **Gate A**：`go build ./...` + `check-file-lines --strict` + `gen_capability_map.sh`。

---

## Phase B · 监控全地址 + 单实例（R6）

- [ ] **B1 监控全地址** — `chain/monitor.loadAddresses`：`ListAssignedAddresses`→新增 `ListAllDerivedAddresses`（含 `ASSIGNED/RETIRED`；Q1 按需派生无 `AVAILABLE`）；命中已退役地址→正常入账；命中未知地址→`MANUAL_REVIEW`。ADR §2.5/§10.3/§11 Q1。
  - 验收：向已派生地址打款可被捕获入账。
- [ ] **B2 单实例锁** — monitor/broadcaster/reconcile 启动前 `pg_try_advisory_lock(<const>)`，未获锁不启动 worker。ADR §10.3。
  - 验收：起两个进程仅一个跑 worker。
- [ ] **B3 checkpoint 初始化** — 部署脚本把 `last_scanned_block` 设为当时链高（禁默认 0）。ADR §2.5。
- **Gate B**：三件套。

---

## Phase C · 归集：构建 / 广播 / 状态机 / 防双花（§2.7，R3）

- [ ] **C0 sweep 表迁移（Q3/B/I）** — 新 migration：`sweep_logs` 拆三腿（`batch_id + leg_type(delegate/transfer/undelegate) + leg_seq + 每腿 tx_hash/status`）；新增 `sweep_bundles(batch_id UNIQUE, signed_bundle BYTEA, built_at_ms, status)` 持久化已签 proto。ADR §2.3/§11 Q3。
- [ ] **C1 未签名构建** — `internal/sweep/builder.go`：`BuildUnsignedBundle(addrs)`；gotron-sdk 造 delegate/transfer(→冷)/undelegate raw_tx，**raw_tx 设近 24h 过期**（供崩溃恢复）；Energy 按 `has_received_usdt` 动态算（DEM）。**不签名**。ADR §2.7/§10.4/§11 Q3。
- [ ] **C2 广播器 + 已签持久化（Q3/I）** — `internal/sweep/broadcaster.go`：导入 `SignedSweepBundle` 先**落 `sweep_bundles`**，再按序 delegate→等确认→transfer→等确认→undelegate；每腿落 `sweep_logs`。**重启后读回 `sweep_bundles` 从首个未确认腿续播（按 txid 幂等，重广播不需私钥）**。ADR §10.4/§11 Q3。
  - 验收：广播中途重启→读回续播不双花；三腿状态各自可追踪。
- [ ] **C3 状态机 + 防双花** — `internal/sweep/state.go`：`GetTransactionInfoByID` 判 SUCCESS/FAILED/未知(保持SWEEPING)；**重试前查该分地址链上出账历史确认无成功/待确认再重签**。ADR §2.7/§10.4。
  - 验收：模拟「DB 写 tx_hash 前崩溃 + 链上已成功」→ 重试不产生第二笔。
- [ ] **C4 管理端归集看板** — `connect/admin/sweep_handler.go`：列出分地址链上余额按降序 + 总额 + 阈值高亮；构建 bundle / 导入已签 / 广播 RPC。ADR §2.7。
- [ ] **C5 卡死兜底（Q3）** — 超 `raw_tx` 过期仍未完成：管理端触发 **undelegate-only 回收** 冷签批次，或接受 14 天自然解押；资金始终安全在分地址。ADR §11 Q3。
- **Gate C**：三件套。

---

## Phase D · 账本完整性 + 偿付能力校验（R7 R8 R9 R10）

- [ ] **D1 migration** — `user_wallets` 加 `CHECK(balance>=0)`；`wallet_transactions` 加 `seq/prev_hash/entry_hash/idem_key(UNIQUE)` + append-only 触发器（禁 UPDATE/DELETE）；新增 `ledger_outbox`。ADR §10.5。
- [ ] **D2 AdjustBalanceTx 改造（R7/R8/R9）** — 幂等（`idem_key` ON CONFLICT）；链尾 `FOR UPDATE` 串行；算 `entry_hash`；写流水 + `ledger_outbox`。所有调用方（deposit/fee/subscription/withdrawal）传 `idem_key`。ADR §10.5。
  - 验收：并发记账哈希链不断裂；UPDATE/DELETE 被 DB 拒；重复 idem_key 不双记；超额 debit 被 CHECK 拒。
- [ ] **D3 冻结模式（R9）** — repo 加 `FreezeForWithdrawal / CompleteWithdrawal / CancelWithdrawal`（balance↔frozen_balance 同事务）。ADR §9.3。
- [ ] **D4 实时外发** — `cmd/ledger-shipper`（或后台 goroutine）读 `ledger_outbox`→推 Telegram/邮件→标记已发。ADR §10.5。
- [ ] **D5 GetLedgerSummary RPC** — admin ConnectRPC：返回 `total_liabilities / latest_seq / latest_entry_hash`。ADR §10.6。
- [ ] **D6 偿付能力校验器（R10）** — `cmd/solvency-check`（**在管理员设备/冷机运行**）：链下算 custody vs liability + 链尾对比 + 死人开关。ADR §9.2/§10.6。
  - 验收：手工调高某余额→下轮报资不抵债；停外发→死人开关报警。
- **Gate D**：三件套。

---

## Phase E · 提现授权 WebAuthn（R11 R12）

- [ ] **E1 passkey 注册** — 用户创建 passkey→存 credential 公钥；同步一份到 coldsign 自持库。依赖 `go-webauthn`。ADR §10.7。
- [ ] **E1b 凭证同步（Q2/R12）** — 在线导出 `UserCredentialList`(proto)→USB→coldsign 本地库；每条凭证变更须带外确认 + 冷却期 + 写入 §9.1 哈希链凭证变更流水；coldsign 导入时比对离机镜像，拒绝未经带外确认的凭证。**禁信在线 `UnsignedTx` 现供公钥**。ADR §9.4/§11 Q2。
- [ ] **E2 提现发起** — challenge=`sha256(amount|dest|nonce|user_id)`；用户签→assertion；在线机存 `withdrawal(PENDING)` + 冻结（D3）+ 构建 `TransferTx.auth=WithdrawalAuth{user_id,nonce,credential_id,assertion}`（A5 proto）。ADR §9.4/§10.7/§11 Q8。
- [ ] **E3 coldsign 验签（R11）** — 按 `WithdrawalAuth.credential_id` 查 coldsign **自持公钥库**取公钥→重建 challenge→验 assertion + `dest∈白名单` + 限额→签。伪造断言/未知 credential_id 拒签。ADR §9.4/§11 Q8。
- [ ] **E4 完成/失败** — 广播成功→`DONE`+解冻扣减；失败→解冻回滚。
- [ ] **E5 白名单/公钥变更（R12）** — 2FA + 邮件确认 + 冷却期≥24h + 写入哈希链流水。
  - 验收：伪造 assertion 拒签；改 dest 后验签失败；提现全程 balance+frozen 守恒。
- **Gate E**：三件套。

---

## 收尾 · 迁移与清理

- [x] **F1 清理遗留** — 迁移 211 DROP `deposit_requests` 表 + 删 v1 配置项；旧 RPC 已不存在（proto 中无 CreateDeposit/ApproveDeposit/RejectDeposit）。ADR §7/§9.6。
- [x] **F2 USDT/USD 口径** — 迁移 211 将 `user_wallets.currency` 从 'USD' 改为 'USDT'；前端 WalletManagement/WalletDropdown 回退值改为 'USDT'。ADR §9.6。
- [ ] **F3 部署加固**（运维任务，非代码） — 在线机 LUKS 全盘加密、DB 不对外、备份加密异地存密钥、worker 低权限用户。ADR §8.5。
- [ ] **F4 上线前迁移**（运维任务，非代码） — 冷签名机就绪、种子 Shamir 冷存、填公开配置、能量账户质押 TRX。ADR §7。

## Definition of Done（整体验收）

- [x] 代码库 `internal` 层零私钥（A8 CI 断言通过）。
- [ ] 已知 xpub+index 派生地址与冷签名机一致；篡改 xpub 拒启动。
- [ ] 归集全离线签名；非冷地址拒签；崩溃重试不双花。
- [ ] 账本哈希链连续、DB 层不可改、幂等、余额不可为负。
- [ ] 偿付能力校验器（链下）+ 死人开关可告警。
- [ ] 提现须用户 WebAuthn 断言 + coldsign 验签；余额守恒。
- [x] 全程 `go build ./...` / `check-file-lines --strict` / `gen_capability_map.sh` 通过。
