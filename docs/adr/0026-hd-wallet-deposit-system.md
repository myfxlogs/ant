# ADR-0026 · HD 钱包充值系统 — 每用户独立地址 + 自动到账确认

- **状态**：Accepted
- **日期**：2026-07-17 (Proposed) / 2026-07-19 (Accepted, post-audit) / 2026-07-19 (v2, 冷签名安全模型定稿) / 2026-07-22 (v3, 归集策略简化)
- **决策者**：Team
- **关联 spec**：无

> **v3 归集策略（权威，覆盖本文任何冲突表述）**：单台物理主机、无 KMS/HSM/额外资源。
> 采用**在线 watch-only（仅 xpub，零私钥）+ 气隙冷签名机（唯一持钥、签名、USB 传 proto 交易包）**。
> **归集和提现均为管理员手动触发**——系统自动做所有不需要私钥的事（监控、入账、对账、Dashboard 余额展示、Bundle 构建/广播），
> 管理员在 Dashboard 上做决策和触发，冷签机只做签名这一件事。无自动定时归集。
> 目标：**用户链上 USDT 对实时 root 免疫**（在线机无可签之物）。落地必读 §2.4 密钥模型、§2.7 归集、§8 单机安全约束。

## 1. 背景

### 1.1 现状

当前充值系统采用**单一收款地址 + 手动审核**模式：

- 所有用户向同一个 USDT TRC20 地址转账
- 用户提交充值请求（金额 + 可选 txHash）
- 管理员手动审核确认到账后入账

### 1.2 问题

| 问题 | 严重度 | 说明 |
|---|---|---|
| **无法区分用户** | 🔴 高 | 两个用户同时充值相同金额且不填 txHash，管理员无法判断哪笔属于谁 |
| **txHash 可选** | 🔴 高 | 用户可能不填 txHash，完全依赖管理员人工核对 |
| **纯手动审核** | 🟡 中 | 7×24 充值需求无法满足，用户等待时间长 |
| **盗用风险** | 🟡 中 | 用户可提交他人链上交易的 txHash 冒充自己的充值 |

### 1.3 需求

- 每个用户有独立的 USDT 收款地址，从源头区分充值归属
- 链上到账自动确认，无需人工审核
- 主种子安全可控
- 支持后续提现功能

## 2. 决策

采用 **HD 钱包（BIP32/BIP44）为每用户派生独立 TRC20 地址 + 在线 watch-only + 离线气隙冷签名**：在线主机零私钥，所有签名在气隙机完成，实时 root 也无法转移资金。

### 2.1 核心架构

```
┌─ 离线冷签名机 (气隙, 永不联网) ────────────────────────────┐
│  主种子 (BIP39 24词) + Shamir 3-of-2 冷存储                 │
│  按需派生: 分地址私钥 / 能量账户私钥 (唯一持有私钥之处)      │
│  职责: 派生地址、签名 归集/能量委托/提现 交易              │
│  I/O: 仅 USB 交换 proto「未签名包 / 已签名包」(禁 JSON)     │
└────────────────────────────────────────────────────────────┘
   ▲ USB 未签名包 (UnsignedSweepBundle)   │ USB 已签名包 (SignedSweepBundle)
   │                                       ▼
┌─ 在线主机 (唯一物理机, 视为可被 root 攻陷) ────────────────┐
│  Watch-only: 仅存 account-level xpub (无私钥/KEK/种子)      │
│                                                            │
│  地址服务: xpub 公开派生 m/44'/195'/0'/0/index → 用户地址   │
│  链上监控: 区块事件扫描, 匹配「所有已派生地址」→ 自动入账   │
│  归集构建: 看板按余额降序 → operator 选 → 构建未签名包      │
│  广播:     导入已签名包 → 按序广播 → 状态机确认            │
│  对账:     链上余额 vs 内部账本 (检测用, 非权威)           │
│  内部账本: user_wallets / deposits / wallet_transactions   │
└────────────────────────────────────────────────────────────┘
   │ USDT Transfer (分地址→冷钱包, 冷签名后广播)
   ▼
┌─ 冷钱包 (归集目标, 仅地址公开) ────────────────────────────┐
│  归集 USDT 汇入; 提现同样冷签名后广播                       │
└────────────────────────────────────────────────────────────┘
```

### 2.2 派生路径

```
用户收款地址:  m / 44' / 195' / 0' / 0 / index
  │      │      │   │   └─ 用户序号（从 0 递增，每个用户唯一）
  │      │      │   └─ 外部链 0（收款地址）
  │      │      └─ 账户号（固定 0）
  │      └─ Tron coin type (BIP44)
  └─ Purpose (BIP44)

能量账户地址（G, 固定单个）:  m / 44' / 195' / 0' / 1 / 0
  └─ 内部链 1 的 index 0，专用于能量账户（质押 TRX / delegate / undelegate）
     coldsign 对 DelegateTx/UndelegateTx 签名时按此固定路径从种子派生私钥（非按 derivation_index）。
```

### 2.3 数据模型变更

#### 新增表：`user_deposit_addresses`（按需派生 + 用户分配）

```sql
CREATE TABLE user_deposit_addresses (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id),  -- 按需派生: 地址在分配时创建, 无未分配态
    address         VARCHAR(64) NOT NULL UNIQUE,  -- TRC20 地址 (Base58)
    derivation_index INT NOT NULL UNIQUE,          -- BIP44 派生 index
    -- 注: 无 encrypted_privkey 列。服务器 watch-only, 不持有任何私钥。
    -- 地址由在线机用 xpub 按需派生; 私钥仅在气隙冷签名机按需从种子派生。
    network         VARCHAR(16) NOT NULL DEFAULT 'TRC20',
    status          VARCHAR(16) NOT NULL DEFAULT 'ASSIGNED',  -- ASSIGNED / RETIRED (无地址池, 无 AVAILABLE)
    has_received_usdt BOOLEAN NOT NULL DEFAULT false,          -- 是否曾收到 USDT (影响首次归集 Energy 计算)
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    assigned_at     TIMESTAMPTZ                     -- 分配时间
);

CREATE INDEX idx_deposit_addresses_user_id ON user_deposit_addresses(user_id);
CREATE INDEX idx_deposit_addresses_address ON user_deposit_addresses(address);
CREATE INDEX idx_deposit_addresses_status ON user_deposit_addresses(status);
```

**按需派生模式（watch-only，§11 Q1 定案 A）**：**不预导入地址池、无 AVAILABLE 状态**。
用户首次索取地址时，在线机用 `deposit_xpub` 按下一个 `index` 实时派生（`m/44'/195'/0'/0/index`）并落库，status 直接为 `ASSIGNED`。DB 中不含任何私钥。
**index 分配（防并发竞态）**：用专用 PG `SEQUENCE` 或单调计数器行（`SELECT ... FOR UPDATE`）分配，杆绝 `MAX(index)+1` 竞态：
```sql
-- 幂等: 已有则直接返回; 否则分配下一 index 并插入
SELECT address, derivation_index FROM user_deposit_addresses
  WHERE user_id = $1 AND status = 'ASSIGNED' LIMIT 1;
-- 无 → 分配新 index (SEQUENCE 保证单调唯一), 在线用 xpub 派生 address, 再:
INSERT INTO user_deposit_addresses (user_id, address, derivation_index, status, assigned_at)
  VALUES ($1, $addr, nextval('deposit_addr_index_seq'), 'ASSIGNED', NOW())
  RETURNING address, derivation_index;
```
`SEQUENCE` 单调递增保证并发安全，不会两个用户拿到同一 index/地址。

#### 新增表：`deposits`（链上充值记录 — 新系统）

```sql
CREATE TABLE deposits (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             UUID NOT NULL REFERENCES users(id),
    deposit_address_id  UUID NOT NULL REFERENCES user_deposit_addresses(id),
    tx_hash             VARCHAR(64) NOT NULL UNIQUE,       -- 链上交易哈希
    amount              NUMERIC(20,8) NOT NULL,             -- USDT 金额 (1:1 USD)
    block_number        BIGINT NOT NULL,                    -- 交易所在区块
    confirmations       INT NOT NULL DEFAULT 0,             -- 区块确认数
    status              VARCHAR(16) NOT NULL DEFAULT 'CONFIRMED',  -- CONFIRMED / MANUAL_REVIEW
    confirmed_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_deposits_user_id ON deposits(user_id);
CREATE INDEX idx_deposits_address_id ON deposits(deposit_address_id);
CREATE INDEX idx_deposits_status ON deposits(status);
```

`deposits` 是新系统的核心表，只有两种状态：
- `CONFIRMED` — 链上验证通过，已自动入账
- `MANUAL_REVIEW` — 多源验证不一致，需人工核实

没有 `PENDING`/`APPROVED`/`REJECTED` — 这些是旧手动审核的概念，不属于新系统。

**为什么新建表而非沿用旧表？**
- `deposit_requests` 的领域语义是"用户发起的请求"（request → approval）
- `deposits` 的领域语义是"链上发生的事实"（fact → confirmation）
- 两个不同的领域概念不应共用一张表
- 新表 schema 更干净：没有 `reviewer_id`、`review_note`、`wallet_tx_id` 等旧审核流程字段，也没有多余的 `amount_usd` 字段（USDT 即 USD，1:1）

**旧表处理：上线前清完 PENDING，不保留旧代码路径。**
- 上线前管理员将 `deposit_requests` 中所有 PENDING 请求审批完毕
- 审完后旧表就是死数据，不需要"冻结为只读"等特殊处理
- 旧 Approve/Reject RPC 不保留 — 不为有限的、会清零的数据维护废弃代码

#### 新增表：`sweep_logs`（归集记录）

一次归集 = 3 笔链上交易（delegate → transfer → undelegate），故 `sweep_logs` **拆为 3 腿**（§11 Q3），每腿自己的 `tx_hash` + `status`，用 `batch_id` 关联同批次：
```sql
CREATE TABLE sweep_logs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    batch_id        UUID NOT NULL,                     -- 同一归集批次的 3 腿共享
    deposit_address_id UUID NOT NULL REFERENCES user_deposit_addresses(id),
    leg_type        VARCHAR(12) NOT NULL,              -- delegate / transfer / undelegate
    leg_seq         SMALLINT NOT NULL,                 -- 广播顺序 0/1/2
    tx_hash         VARCHAR(64) UNIQUE,                -- 本腿交易 hash (NULL 直到广播成功)
    amount          NUMERIC(20,8) NOT NULL DEFAULT 0,  -- 仅 transfer 腿有值
    energy_used     BIGINT NOT NULL DEFAULT 0,         -- 仅 delegate 腿有值
    status          VARCHAR(16) NOT NULL DEFAULT 'PENDING',  -- PENDING / SWEEPING / DONE / FAILED / MANUAL_REVIEW
    error_message   TEXT,                              -- 失败原因
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at    TIMESTAMPTZ
);

CREATE INDEX idx_sweep_logs_batch_id ON sweep_logs(batch_id);
CREATE INDEX idx_sweep_logs_address_id ON sweep_logs(deposit_address_id);
CREATE INDEX idx_sweep_logs_status ON sweep_logs(status);
```

#### 新增表：`sweep_bundles`（已签 bundle 持久化，§11 Q3）

已签交易**不含私钥**，可安全落库；在线机重启后读回续播（重广播不需私钥，按 txid 幂等），闭合归集原子性缺口：
```sql
CREATE TABLE sweep_bundles (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    batch_id        UUID NOT NULL UNIQUE,              -- 关联 sweep_logs.batch_id
    signed_bundle   BYTEA NOT NULL,                    -- 序列化 SignedSweepBundle proto (无私钥)
    built_at_ms     BIGINT NOT NULL,                   -- 构建时间 (raw_tx 过期判断, 近 24h 上限)
    status          VARCHAR(16) NOT NULL DEFAULT 'BROADCASTING',  -- BROADCASTING / DONE / EXPIRED
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

#### 不新增 `wallet_secrets` 表（服务器零私钥）

冷签名模型下**服务器不存任何钱包私钥**（含热钱包/能量账户/分地址）。
所有签名密钥仅存在于气隙冷签名机，由主种子按需派生。
在线机只需一个**公开配置** `deposit_xpub`（+ `deposit_xpub_fingerprint` 校验）、
`cold_wallet_address`、`energy_account_address`——均为公开数据，非机密。
因此不引入 `wallet_secrets` 表，也不引入任何 deposit/hot-wallet 的 secrets Purpose。

### 2.4 密钥安全模型：在线 watch-only + 离线冷签名（零在线资金密钥）

**第一性原则：能动资金的私钥永不出现在在线主机上。** 在线主机视为「随时可能被 root 完全攻陷」，故不持有任何可签名密钥；实时 root 也签不出任何转账——机器上没有钥匙。这是单机条件下的理论最优（在线资金密钥数 = 0，无法更低）。

**离线冷签名机（气隙，永不联网）：** 冷存 24 词助记词（金属板 + Shamir 3-of-2）；按需从种子派生「分地址私钥」与「能量账户私钥」；唯一职责是签名（归集/能量委托/提现）；唯一 I/O 是 USB 传 **proto 序列化**交易包（禁 JSON，遵守 AGENTS.md）。

**在线主机：** 只存 account-level **xpub**（`m/44'/195'/0'/0` 节点），零私钥/KEK/种子；用 xpub 公开派生 `m/44'/195'/0'/0/index` 得用户地址（secp256k1 非硬化派生，无需私钥）；跑监控、入账、对账、归集构建与广播，全不需私钥。

**xpub 完整性（防 root 换 xpub 劫持新地址）：** 启动校验 xpub 指纹 == 配置 `deposit_xpub_fingerprint`，不符拒启动；冷签名机定期导出「index→address 权威表」（公开数据）与 DB 逐条比对，不一致告警。

**安全结论：** 在线主机被实时 root 完全攻陷 → 只泄漏公开信息（xpub、地址、内部账本），**签不出任何链上转账**，用户 USDT（分地址 + 冷钱包）不可被盗。

#### 2.4.1 冷钱包独立密钥（安全域隔离）

**冷钱包私钥独立于 BIP39 种子。** 冷钱包不在 HD 树内——它是一个独立生成的 TRC20 密钥对，私钥由冷签机单独管理。

| | BIP39 种子（HD 树）| 冷钱包私钥 |
|---|---|---|
| 保护范围 | 用户充值地址私钥 + 能量账户私钥 | 冷钱包（归集目标 + 提现源）|
| 派生方式 | m/44'/195'/0'/0/{index} + m/44'/195'/0'/1/0 | 独立生成的 secp256k1 密钥对 |
| 冷签机获取方式 | 输入 BIP39 助记词后按需派生 | `-cold-wallet-key <hex>` 参数导入 |
| 泄露后果 | 充值地址 + 能量账户全损 | 冷钱包全损 |
| 密钥轮换 | 新 BIP39 种子 + 新 xpub | 新 TRC20 地址 + 更新 config |

**安全优势：** 种子泄露 ≠ 冷钱包泄露。冷钱包累计归集了大量 USDT——这个隔离意味着攻击者需要同时攻破种子和冷钱包私钥才能拿走所有资金。

**冷签机持有清单：**
```
├─ BIP39 助记词 (stdin 输入, 不落盘)
│   → 充值地址私钥 (TransferTx, DerivationIndex=index) — sweep 签名
│   → 能量账户私钥 (DelegateTx/UndelegateTx, m/44'/195'/0'/1/0) — 能量签名
└─ 冷钱包私钥 (-cold-wallet-key 参数)
    → TransferTx FromAddress==coldWalletAddr — 提现签名
```

**signTx() 签名分支（R3b：key_source 为权威，不靠 from_address 猜测）：**
```
key_source = cold_wallet_key  → 用独立冷钱包私钥签名 (提现, -cold-wallet-key 参数)
key_source = bip39_derivation_index X:
  ├─ TransferTx            → m/44'/195'/0'/0/X  (归集分地址)
  └─ DelegateTx/UndelegateTx → m/44'/195'/0'/1/0 (能量账户固定路径)
key_source 未设置 → abort (proto 不合法)
```

| 层级 | 措施 | 实现方式 |
|---|---|---|
| **生成** | 离线生成 | 气隙机生成 24 词助记词，永不联网 |
| **存储** | 冷存储 | 助记词刻金属板存保险箱，Shamir 分片 3-of-2 |
| **在线派生** | 公开派生 | 在线机仅用 xpub 派生地址，**不接触任何私钥** |
| **服务器** | watch-only | 只存 xpub，无私钥/KEK/种子 |
| **签名** | 冷签名 | 100% 在气隙机，USB 传 proto 交易包 |
| **能量** | 冷签名 | 能量账户（冷）质押 TRX，委托/收回也冷签名 |
| **轮换** | 支持轮换 | 新种子派生新 xpub 段，旧地址继续收款，旧地址归集仍由冷机签名 |

### 2.5 链上监控流程

**采用区块事件扫描，非逐地址轮询。** 复杂度 O(1) per block，与用户数无关。

> **Push-First Architecture 合规例外**：项目规则禁止 polling，但 TronGrid API 无 push/webhook 能力，自建 Tron 全节点 + ZeroMQ/Kafka 事件订阅成本过高（月 $200-1000 + 运维）。当前阶段采用区块事件扫描（每 ~3s 一次 API 调用）作为**唯一可行的近似 push 方案**。未来升级路径：自建 Tron 节点 → ZeroMQ 事件订阅 → 真正的 push 模式。

```
1. 启动时加载**所有已派生地址**集合到内存 map[string]uuid.UUID
   (address → user_id; 含 ASSIGNED 与 RETIRED；§11 Q1 按需派生无未分配地址)
   **必须监控全部已派生地址**: 用户误用旧 QR、或打款到已退役地址时,
   仍能捕获入账, 避免资金静默丢失。命中未知（非已派生）地址 → 记 MANUAL_REVIEW 告警。
   **last_scanned_block 初始化**: 部署时必须设为当时链高, 不可用默认 0
   (否则会从创世块全量回扫)。

2. 扫描新区块 (每 ~3s 一个区块):
   GET /v1/contracts/{usdt_contract}/events
     ?event_name=Transfer&block_number=N&only_confirmed=true
   → 获取该区块所有 USDT Transfer 事件
   → 如返回分页 (meta.fingerprint 存在), 继续翻页直到取完
   → 遍历事件, 检查 result.to 是否在用户地址 map 中
   → 命中: 记录到待确认队列

3. 记录 last_scanned_block 到 DB → 重启后从断点继续, 不遗漏
   **checkpoint 安全**: 如果 saveCheckpoint 失败, 不推进 *lastBlock,
   下一轮重试同一区块。避免因 checkpoint 写入失败而永久跳过区块。

4. 等待区块确认 (≥20 确认, ~1 分钟):
   - only_confirmed=true 已经过 TronGrid 确认 (~19 solidified blocks)
   - 额外等待是双保险, 确保不会因 reorg 回滚

5. 多源交叉验证 (§2.6):
   - TronGrid 确认交易存在且字段匹配
   - TronScan 备源交叉验证
   - 两个源都确认 → 进入步骤 6
   - 只有一个源确认 → 标记 MANUAL_REVIEW, 人工核实

6. 验证交易字段:
   - contract == USDT TRC20 (TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t)
   - to == 用户分地址
   - amount == 交易金额
   - txHash 未被使用 (DB 唯一约束)

7. 自动入账 (DB 事务):
   BEGIN
     INSERT deposits (tx_hash, amount, status='CONFIRMED', ...)
     UPDATE user_wallets SET balance = balance + amount_usd
     INSERT wallet_transactions (tx_type='deposit', ...)
   COMMIT
   **失败安全**: 如果 ConfirmDeposit 因任何原因失败 (DB 连接闪断、wallet not found 等),
   插入一条 status='MANUAL_REVIEW' 的 deposit 记录作为降级, 确保资金不丢失,
   admin 可后续人工补账。绝不静默丢弃。

8. 标记待归集 (不自动归集):
   - 仅更新地址 USDT 余额供管理端看板展示
   - 归集由 operator 手动发起, 走离线冷签名流程 (见 §2.7)
   - 在线机不持有私钥, 无法也不会自动签名归集
```

**为什么不用逐地址轮询？**
- 逐地址轮询: O(N) API calls per poll cycle, 1000 用户 = 1000 calls/10s
- TronGrid 免费限制: 20 QPS, 100k req/day
- 区块事件扫描: O(1) per block, 1 call/3s, 与用户数无关
- 1000 用户和 10 用户成本完全一样

### 2.6 多源验证（安全加固）

```
主源: TronGrid API
备源: TronScan API (https://apilist.tronscanapi.com)

验证逻辑:
  - 两个源都确认交易存在且字段匹配 → 自动入账
  - 只有一个源确认 → 标记 MANUAL_REVIEW，人工核实
  - 两个源都查不到 → 忽略（可能是假交易）
```

### 2.7 归集策略：人工决策 + 系统自动执行 + 离线冷签名

**不做自动定时归集。** 分地址的 USDT 私钥只在冷签名机，资金本身已安全；归集只为便于记账/提现汇入冷钱包，无时间压力。

**管理端归集看板（只读，便利用途，非权威）：**
- 列出所有分地址链上 USDT 余额，**按金额降序**。
- 显示待归集总额；单地址 ≥ `sweep_alert_threshold` 高亮提醒。
- 数据来自在线机查链，可能被 root 篡改——最终以冷签名机屏幕与链上事实为准。

**归集 = 管理端一键操作，系统自动 3 腿执行。** 每地址 3 笔链上交易，全部冷签名：

| 步 | 交易 | 签名方 | 谁触发 | 谁执行 |
|---|---|---|---|---|
| 1 | `DelegateResource`（能量账户 → 分地址）| 能量账户（冷）| Admin 点"构建" | 系统自动 Broadcast + 确认 |
| 2 | `TRC20 Transfer`（分地址 → 冷钱包）| 分地址私钥（冷）| 同上 | 同上 |
| 3 | `UndelegateResource`（收回 Energy）| 能量账户（冷）| 同上 | 同上 |

分地址仅**接收**委托，不需其私钥参与委托交易；三笔签名密钥都只在冷签名机。

**归集操作流程（admin 手动触发）：**

```
1. Admin 打开 Dashboard → Sweep 视图 → 查看各地址余额（按金额降序）
2. 选择要归集的地址 → 点击 [构建归集 Bundle]
   → 系统自动: 构建 UnsignedSweepBundle (3N 笔交易, proto binary)
3. 导出 Bundle → USB → 气隙冷签机
4. 冷签机校验 + 签名 (SignedSweepBundle):
   - 硬白名单: transfer.to == cold_wallet_address，否则拒签
   - 屏幕显示每笔 金额/源/目的地 供 operator 核对
   - 冷钱包用独立私钥（非 BIP39 种子派生，§2.4.1）
5. 签名包 USB → 在线机
6. Admin 点击 [导入签名包] → 系统自动按序广播:
   delegate → 等确认 → transfer → 等确认 → undelegate → 等确认
   - 每步落 sweep_logs 状态机，Dashboard 实时显示进度
7. 更新 has_received_usdt; 触发对账
```

**批量归集：** 一次冷签名批次内委托整批 Energy、归集多个地址、统一收回，减少 USB 往返与签名次数。

**提现操作流程（admin 手动触发）：**

```
1. 用户通过 WalletPage 发起提现:
   - BeginWithdrawal → Passkey 签名 → FinishWithdrawal (资金冻结)
   - 状态: SIGNED_WAITING_BUNDLE
2. Admin 打开 Dashboard → 提现审核列表
   → 查看: 用户、金额、目标地址、白名单状态
3. Admin 点击 [构建提现 Bundle] → 导出 → USB → 冷签机
4. 冷签机:
   - 用自持公钥库验证 WebAuthn 断言 (R11)
   - 验证 dest ∈ 用户提现白名单
   - 验证单笔/单日限额
   - 用冷钱包独立私钥签名 TransferTx (§2.4.1)
5. 签名包 USB → 在线机 → Admin 点击 [导入并广播]
6. 广播成功 → CompleteWithdrawal (扣除冻结余额)

**能量账户：** 独立冷账户，只质押 TRX 换 Energy、只做委托/收回，**绝不持有 USDT**；私钥在冷签名机。质押（FreezeBalanceV2）为一次性冷签名设置。批量归集可一次委托整批、统一收回（Stake 2.0 委托/收回无锁定）。

**Energy 动态计算（DEM）：** 按 `has_received_usdt`：首次 130k×`dem_factor`，常规 65k×`dem_factor`，另加 `energy_buffer_percent`。

**为什么用 Energy 委托而非直接烧 TRX（第一性推导）：** TRON TRC20 转账需消耗资源，两种路径：

| | A. 每地址预存 TRX 直接烧 | B. 能量账户质押 + Energy 委托（选定） |
|---|---|---|
| 交易数/地址/次 | 1（直接 transfer） | 3（delegate → transfer → undelegate） |
| 状态机复杂度 | 低（单笔幂等，有 tx_hash 即成立） | 高（三腿状态、超时恢复、孤儿回收） |
| 持续成本 | 每笔 ~10–20 TRX 被销毁 | 首次质押 5000 TRX 锁定（~$500 机会成本），后续零消耗 |
| 1000 地址 5 年 | ~$10,000–$20,000 | ~$500（仅质押机会成本） |
| 新用户 onboarding | 分地址需预存 TRX（增加一步操作或冷签额外签名） | 无需额外操作 |

**选 B 的理由**：规模上去后成本差异是数量级的。三腿复杂度是 TRON Stake 2.0 资源模型的必然产物（非偶然设计），且每腿由 `sweep_logs.leg_type` 独立追踪、已签 bundle 持久化（§11 Q3）使崩溃可恢复，不依赖手工协调。如果用户量确定很小（<100），A 的简单性有优势；但本系统面向增长，选 B 是第一性下的最优解。

**管理员 Dashboard 操作清单：**

| 操作 | 触发方式 | 说明 |
|------|---------|------|
| 查看待归集余额 | 自动刷新 | GetSweepDashboard — 地址列表 + 余额降序 |
| 构建归集 Bundle | Admin 点按钮 | BuildUnsignedBundle — 系统自动构建 3N 笔交易 |
| 导出 Bundle | Admin 点导出 | 下载 UnsignedSweepBundle proto binary |
| 导入签名包 | Admin 点导入 | 上传 SignedSweepBundle → 自动广播 |
| 查看广播进度 | 自动刷新 | 每腿状态: PENDING→SWEEPING→DONE/FAILED |
| 构建 undelegate-only | Admin 点按钮 | 回收卡住地址的委托能量 |
| 提现审核 | Admin 点按钮 | 查看 SIGNED_WAITING_BUNDLE 列表 → 构建/导出/冷签/导入/广播 |

**归集幂等与广播安全（确定性验证 — 第一性原理）：**
- sweep_logs 记录状态: PENDING → SWEEPING → DONE / FAILED / MANUAL_REVIEW
- Tron 没有 Ethereum 式 nonce 机制, 幂等性由状态机 + 链上交易状态检查保证
- **核心原则: 永不猜测, 基于链上事实确定性验证**

广播后的三种链上状态:
  - SUCCESS → 标记 DONE (确定性)
  - FAILED → 标记 FAILED, 可安全重新签名归集 (确定性)
  - 未知 (超时) → **保持 SWEEPING**, 不标 DONE 也不标 FAILED

超时处理 (reconfirmation checker):
  - 每轮 scanAndSweep 开始时, 查询所有 SWEEPING + tx_hash 的记录
  - 调用 GetTransactionInfoByID 查询链上最终状态
  - SUCCESS → DONE / FAILED → FAILED / 仍未找到 → 继续保持 SWEEPING
  - SWEEPING 状态阻止 ListUnsweptAddresses 重复归集同一地址
  - 绝不盲目重新广播, 避免双花

卡死恢复 (MarkStuckSweepingAsFailed, 5 分钟超时):
  - SWEEPING + tx_hash → MANUAL_REVIEW (广播已成功, 资金可能已转移, 人工核实)
  - SWEEPING 无 tx_hash → 见下方「重试防双花」, 确认链上无出账后才转 FAILED
  - PENDING → FAILED (未开始, 可安全重试)

**重试防双花（关键 — 必须实现）：**
- 对「FAILED 或无 tx_hash」的重试, **必须先查该分地址链上最近出账历史**,
  确认无成功/待确认的归集交易, 再重新构建冷签名; 不得仅凭本地无 tx_hash 就重播
  (DB 写入可能晚于广播成功 → 否则会 double-sweep, Tron 无 nonce, 重签即新 txID)。

- DONE 状态不再重复归集

**批量归集优化：** 一次冷签名批次内委托整批 Energy、归集多个地址、统一收回, 减少 USB 往返与签名次数。

**注意事项：**
- 归集后地址保留，支持用户多次充值到同一地址
- 最小充值金额 1 USDT；低于归集阈值的小额可暂不归集（资金仍安全在分地址）

### 2.8 对账机制

**两阶段对账，避免 API 调用量超限：**

```
阶段 1 — 内部对账 (每 6h, 无 API 调用):
  预期链上余额 = Σ(deposits WHERE status='CONFIRMED' AND amount)
               - Σ(sweep_logs WHERE status='DONE' AND leg_type='transfer' AND amount)  -- 仅 transfer 腿有金额(§2.3 三腿)
  DB 余额 = Σ(user_wallets.balance)

  预期 == DB → ✅ 内部账本一致, 无需查链上
  预期 != DB → ⚠️ 内部不一致, 查 DB 事务完整性

阶段 2 — 链上对账 (每 24h, 00:00 UTC):
  链上托管 (冷签名模型无热钱包):
    Σ(所有已派生分地址 USDT 余额)     -- TronGrid API 查询 (全地址, 非仅 ASSIGNED)
    + 冷钱包 USDT 余额
  =?
  预期链上余额 (阶段 1 计算)

  差异 = 0 → ✅ 链上与内部一致
  差异 (= 链上托管 − 内部负债) > 0 → 链上比账本多: 未入账充值/多打, 相对良性 → 排查链上监控
  差异 < 0 → 链上比账本少: 潜在资不抵债/被盗, 危险方向 → 紧急排查

  差异 > +$10 或 < −$1 → 立即告警
  (阈值刻意不对称: 少钱=偿付能力风险, 更紧急故 −$1; 多钱多为良性, 放宽到 +$10 减少噪声)
```

**API 调用量优化**：
- 内部对账 (6h): 0 API calls, 纯 DB 计算
- 链上对账 (24h): N calls (N = ASSIGNED 地址数), 但只在内部对账通过时才执行
- 1000 用户: 1000 calls/day (链上对账) + ~28,800 calls/day (链上监控) = ~30,000 calls/day
- TronGrid 免费限制: 100k/day, 余量 70k, 充足

## 3. 备选方案

| 方案 | 优点 | 缺点 | 否决理由 |
|---|---|---|---|
| **A. txHash 必填 + TronGrid 验证** | 实现简单（1-2 天），无需管理私钥 | 不能 100% 防盗用（竞态），用户需手动填 txHash | 用户体验差且安全性不足，一步到位做 HD 钱包 |
| **B. 金额加随机尾数** | 实现极简 | 用户转账不便（需转 100.37 而非 100），大额时尾数不够区分 | 用户体验差，不可靠 |
| **C. 第三方支付网关** | 无需自建，合规性好 | 手续费高（1-3%），引入第三方依赖，提现受限 | 成本高，丧失自主性 |
| **D. 每用户独立地址 (HD)** ✅ | 从根源区分用户，自动到账，用户体验最佳 | 实现复杂，需管理主种子安全 | **选定** — 安全性和用户体验最优 |

直接实施方案 D，不做过渡方案。

## 4. 后果

### 正面
- 每个用户独立地址，充值归属 100% 确定
- 链上自动确认（入账），7×24 无需人工审核
- 用户无需填 txHash，体验更好
- **在线机零私钥**：实时 root 攻陷也无法转移用户资金（单机可达的最强姿态）
- 为后续提现功能奠定基础（提现同走冷签名）

### 负面
- 归集/提现需 Admin 手动触发 + 离线冷签名（USB 往返），非全自动
- 需 Admin 定期查看 Dashboard 并操作（每日 ~15 分钟），无人值守时充值地址余额积压
- 需要离线管理主种子、冷钱包私钥与气隙冷签名机
- 需要 Stake TRX 获取 Energy（资金锁定 14 天解押期）
- 冷钱包私钥独立于 BIP39 种子——需额外备份

### 中性
- 需要新增链上监控服务
- 需要定期对账机制
- DB schema 变更

## 5. 实施约束

### 5.1 技术栈

| 组件 | 选型 | 说明 |
|---|---|---|
| 在线地址派生 (watch-only) | `github.com/btcsuite/btcd/btcutil/hdkeychain` | 通用 BIP32, 仅用 xpub 公开派生地址, 不接触私钥（与 §10.1 一致） |
| 离线派生 + 签名 | `gotron-sdk` (气隙机) | 从种子派生私钥, 签名归集/能量/提现交易 (仅气隙机) |
| 链上查询 | TronGrid API (主) + TronScan API (备) | 多源交叉验证 |
| 密钥 | **不引入新 secrets Purpose** | 服务器零私钥, 无需加密存储 |
| 监控 | in-process goroutine | 区块事件扫描; 单实例 + PG advisory lock 防重入 |
| 跨机交换 | proto 序列化 (USB) | UnsignedSweepBundle / SignedSweepBundle, 禁 JSON |

### 5.2 secrets Purpose（不新增）

冷签名模型下服务器**不持有任何钱包私钥**，故**不新增** `PurposeDepositPrivKey` / `PurposeHotWalletKey`。
`internal/secrets/vault.go` 保持现状（仅 MT 相关 Purpose）。在线机只需公开配置（xpub / 冷地址 / 能量账户地址），都不是机密。

### 5.3 API 变更

#### 新增 RPC

```proto
// 获取用户的充值地址（幂等：已有则返回，无则按需派生）
// 逻辑: 先查 user_deposit_addresses WHERE user_id=$1 AND status='ASSIGNED'
//       有 → 返回; 无 → 从 SEQUENCE 取下一 index, xpub 派生地址, INSERT (§2.3, §11 Q1)
rpc GetDepositAddress(GetDepositAddressRequest) returns (GetDepositAddressResponse);

message GetDepositAddressRequest {}
message GetDepositAddressResponse {
  string address = 1;       // TRC20 地址 (Base58)
  string network = 2;       // "TRC20"
}
// 注: QR 码由前端生成 (qrcode.react), 后端不处理展示层关注点

// 查询用户充值历史（替代旧 CreateDeposit + ListMyDeposits）
// 新模式下用户不需要手动提交充值请求, 系统自动检测入账
// ListMyDeposits 保留, 但返回的 deposits 由系统自动创建
```

#### 删除旧 RPC

```proto
// 删除: CreateDeposit, ApproveDeposit, RejectDeposit
// 删除: message CreateDepositRequest, ApproveDepositRequest, RejectDepositRequest
// 删除: message DepositRequest (旧)
```

#### 新增 proto message

```proto
// 新: Deposit (对应 deposits 表)
message Deposit {
  string id = 1;
  string user_id = 2;
  string deposit_address_id = 3;
  string tx_hash = 4;
  string amount = 5;               // USDT 金额 (NUMERIC → string)
  int64 block_number = 6;
  int32 confirmations = 7;
  string status = 8;               // CONFIRMED / MANUAL_REVIEW
  Timestamp confirmed_at = 9;
  Timestamp created_at = 10;
}

// 修改: ListMyDeposits 返回 Deposit (新) 而非 DepositRequest (旧)
message ListMyDepositsResponse {
  repeated Deposit deposits = 1;
  int32 total = 2;
}
```

### 5.4 文件结构

```
backend/internal/
├── hdwallet/
│   ├── xpub.go              # HD 公开派生 (watch-only, 仅 xpub, 无私钥)
│   ├── derive_priv.go       # 私钥派生 (仅 cmd/coldsign 使用, 不引入在线代码)
│   ├── sign_tron.go         # TRON 交易签名 (仅 cmd/coldsign 使用)
│   └── wallet.go            # BIP39 助记词生成 (cmd/hdgen)
├── chain/
│   ├── monitor.go           # 链上监控 worker (区块事件扫描 + 双源验证)
│   ├── tron_grid.go         # TronGrid API client (主源)
│   └── tron_scan.go         # TronScan API client (备源)
├── sweep/                   # 在线侧: 只构建与广播, 不签名
│   ├── builder.go           # 构建 UnsignedSweepBundle (delegate/transfer/undelegate)
│   ├── batch_builder.go     # 批量构建 (多地址一个 USB 往返)
│   ├── broadcaster.go       # 导入 SignedSweepBundle → 按序广播 + 确认
│   ├── state.go             # 状态机: ReconfirmSweeping + CheckDoubleSpend + 防双花
│   ├── worker.go            # Worker 周期: 仅保留 ReconfirmSweeping + resumeBroadcasting
│   ├── tron_client.go       # TRON gRPC client (构建交易 + 广播 + 确认)
│   ├── admin.go             # Admin RPC: Dashboard + Export/Import Bundle
│   ├── interfaces.go        # 接口定义 (TronClient/SweepRepo/TronGrid)
│   ├── repo.go              # BundleRepository (未签名/已签名 bundle 持久化)
│   └── sweep_test.go        # Broadcaster + StateMachine 测试 (884 行)
├── reconcile/
│   └── reconcile.go         # 两阶段对账 (内部 6h + 链上 24h)
├── service/
│   ├── deposit_service.go   # 充值: 地址派生 + 按需分配 + 到账确认入账
│   ├── withdrawal_builder.go # 提现 Bundle 构建 (Admin 手动触发)
│   └── webauthn_withdrawal.go # WebAuthn 提现: Begin → Passkey 签名 → Finish → 冻结
├── repository/
│   ├── deposit_repo_v2.go   # deposits / sweep_logs / sweep_bundles CRUD
│   └── sweep_log_repo.go    # sweep_logs 3 腿状态管理
└── connect/user/
    ├── deposit_handler.go   # DepositService RPC (充值 + 归集 Admin 操作)
    └── sweep_handler.go     # Sweep RPC: Dashboard + Export/Import Bundle + Undelegate

backend/cmd/
├── hdgen/
│   └── main.go             # 气隙机: 生成助记词 → 导出 account-level xpub + fingerprint
│                             # (§11 Q1: 地址以在线 xpub 按需派生为权威; 可选导出一份地址表供一次性交叉核对)
└── coldsign/
    └── main.go             # 气隙机: 读 UnsignedSweepBundle → 校验白名单 →
                              # 从种子派生私钥签名 → 屏幕展示供核对 → 输出 SignedSweepBundle
```

### 5.5 实施阶段

| 阶段 | 内容 | 预估工期 | 依赖 |
|---|---|---|---|
| **Phase 1** | `hdgen`(离线: 种子/xpub/fingerprint + 独立冷钱包密钥对) + xpub watch-only 按需派生 + index SEQUENCE 分配 + GetDepositAddress RPC + DB migration | 3 天 | 无 |
| **Phase 2** | 区块事件扫描(全地址) + 自动确认入账 + txHash 唯一约束 + 断点恢复 + advisory lock | 3 天 | Phase 1 |
| **Phase 3** | `cmd/coldsign`(离线签名, 含 -cold-wallet-key) + UnsignedSweepBundle 构建/广播(Admin 手动触发) + 归集状态机 + 防双花 | 4 天 | Phase 2 |
| **Phase 4** | 多源验证 + 对账机制 (每 6h) + 地址审计 | 2 天 | Phase 2 |
| **Phase 5** | 前端改造（用户: 充值地址+QR+提现 Passkey；管理端: Sweep Dashboard+提现审核+Bundle 管理） | 3 天 | Phase 1/3 |
| **总计** | | **~15 天** | |

### 5.6 配置项

```sql
-- system_config 新增
INSERT INTO system_config (key, value) VALUES
('tron_grid_api_key', ''),                           -- TronGrid API key
('tron_scan_api_key', ''),                           -- TronScan API key
('usdt_contract_address', 'TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t'),  -- USDT TRC20 合约
('min_confirmations', '20'),                         -- 最小区块确认数
('min_deposit_amount', '1'),                         -- 最小充值金额 USDT
('sweep_threshold', '0.01'),                         -- 归集阈值 USDT
('sweep_alert_threshold', '1000'),                   -- 单地址余额高亮提醒阈值 USDT
('deposit_xpub', ''),                                -- account-level 扩展公钥 (公开)
('deposit_xpub_fingerprint', ''),                    -- xpub 指纹, 启动校验防篡改
('cold_wallet_address', ''),                         -- 归集目标冷钱包地址 (公开)
('energy_account_address', ''),                      -- 能量账户地址 (冷, 只质押 TRX)
('stake_trx_amount', '5000'),                        -- Stake TRX 换 Energy 的数量
('reconcile_alert_threshold', '10'),                 -- 对账告警阈值 USD
('reconcile_interval_hours', '6'),                   -- 对账频率 (小时)
('last_scanned_block', '0'),                         -- 链上监控最后扫描的区块号
('sweep_batch_size', '10'),                          -- 批量归集每批数量
('sweep_min_confirmations', '20'),                   -- 归集所需最小确认数
-- (§11 Q1: 按需派生无地址池, 删除 address_pool_min_threshold)
('dem_factor', '1.3'),                               -- USDT 合约 DEM 因子
('energy_buffer_percent', '10');                     -- Energy 委托额外 buffer 百分比
```

## 6. 验证方式

### 6.1 单元测试
- HD 派生：已知种子 → 已知地址（BIP44 测试向量）
- 交易验证：mock TronGrid 响应 → 验证字段检查逻辑
- 归集：mock 签名 → 验证转账参数
- 对账：mock 链上余额 + DB 余额 → 验证差异检测

### 6.2 集成测试
- TronGrid API 连通性（testnet）
- 端到端：生成地址 → 转入 USDT（testnet）→ 自动确认 → 余额入账 → 归集

### 6.3 安全验证
- **在线机零私钥**: 扫描代码库确认无任何私钥/种子/KEK 路径
- xpub 公开派生地址与冷签名机派生结果一致（BIP44 测试向量）
- xpub 指纹校验: 篏改 xpub 应拒绝启动
- 冷签名白名单: 目的地 != cold_wallet_address 的交易应拒签
- txHash 唯一约束测试（重复入账应失败）
- 重放入账测试（同一交易不应入账两次）
- **归集防双花**: 无 tx_hash 重试前必查链上出账历史, 已出账则不重广播
- 并发地址分配测试（基于 SEQUENCE/计数器: 两个并发请求不会拿到同一 index/地址）
- 已签 bundle 持久化恢复测试: 广播中途重启→读回 sweep_bundles 续播, 不双花
- 已退役(RETIRED)地址监控测试: 打款到已退役地址也能捕获入账

### 6.4 对账验证
- 模拟差异场景 → 验证告警触发
- 日常对账 cron 执行 → 验证报告输出

## 7. 上线前迁移

HD 钱包上线前需要完成的准备工作：

1. **清理旧 PENDING**：管理员将 `deposit_requests` 中所有 PENDING 请求审批完毕
2. **移除旧 RPC**：删除 CreateDeposit / ApproveDeposit / RejectDeposit RPC 和对应 handler
3. **现有用户**：首次访问充值页面时按需派生分配独立地址（§11 Q1）
4. **旧收款地址**：从 `system_config` 中移除，不再使用
5. **地址派生**（§11 Q1）：无需预导入地址池；在线机用 `deposit_xpub` 按需派生。可选：`hdgen` 导出一份地址表供与在线派生结果一次性交叉核对
6. **冷签名机就绪**：气隙机部署 `hdgen`/`coldsign`，生成种子、Shamir 分片冷存、导出 xpub + 指纹
7. **配置公开项**：填 `deposit_xpub` / `deposit_xpub_fingerprint` / `cold_wallet_address` / `energy_account_address`
8. **能量账户质押**：能量账户（冷）一次性 `FreezeBalanceV2` 质押 TRX（离线签名后广播）
9. **初始化 `last_scanned_block`**：设为部署当时链高，严禁默认 0

## 8. 单机气隙冷签名安全模型（落地权威约束）

> 本节为 GLM 落地的**强制约束**。归集和提现均为 Admin 手动触发（§2.7），系统自动执行已签名的 Bundle 广播和链上确认。

### 8.1 不可违反的红线

- **R1 在线机零私钥**：在线代码库中**不得出现**任何私钥、助记词、种子、KEK 的读取/存储/派生路径。仅允许 xpub（公钥）与公开地址。CI 应有 grep 断言（无 `PrivateKey`/`Mnemonic`/`Seed`/`Decrypt` 于钱包路径）。
- **R2 签名只在气隙机**：归集、能量委托/收回、提现的**所有签名**只能由 `cmd/coldsign` 在离线机完成。在线机只构建未签名交易与广播已签名交易。
- **R3 跨机交换用 proto**：`UnsignedSweepBundle` / `SignedSweepBundle` 用 protobuf 序列化经 USB 传递，**禁止 JSON**（遵守 AGENTS.md）。
- **R3b 密钥源显式声明**：`UnsignedTx.key_source` 必须显式声明 `bip39_derivation_index` 或 `cold_wallet_key`。冷签机**不得依赖 `from_address` 字符串匹配或 context 推断来决定用哪个密钥**——proto 本身是唯一权威。防配置篡改导致的静默密钥切换。
- **R4 冷地址硬白名单**：`coldsign` 只签 `transfer.to == cold_wallet_address` 的转账，其余一律拒签（防在线机被 root 篡改改道）。
- **R5 xpub 完整性**：服务启动校验 `deposit_xpub` 指纹 == `deposit_xpub_fingerprint`，不符则拒绝启动。
- **R6 单实例**：监控/广播/对账用 PG advisory lock 保证全局仅一个执行者（单机无需 leader 选举）。

### 8.2 威胁模型与保证

| 资产 / 属性 | 对实时 root 的保证 | 依据 |
|---|---|---|
| 分地址 USDT / 冷钱包 USDT | ✅ 完全免疫 | 私钥永不在线机 (R1/R2) |
| 归集不被改道 | ✅ | 冷地址硬白名单 (R4) |
| 密钥不被静默切换 | ✅ | key_source 显式声明 (R3b)，冷签机不猜测 |
| 新地址不被劫持 | ✅ | xpub 指纹校验 + 地址审计 (R5/§2.4) |
| 能量账户 TRX | ✅ | 能量账户私钥也在冷机 |
| 内部账本完整性 | ⚠️ 不可绝对保证 | 持续 root 可篡改 DB；靠链上对账检测 + 从链重建 |
| 提现目的地正确性 | ⚠️ 靠人工核对 | `coldsign` 屏幕核对 + 单笔/单日限额 |

**已记录的残余风险（单机第一性天花板，非缺陷）：**
- **BIP32 非硬化级联（Q4）**：`m/44'/195'/0'/0/index` 末级非硬化，`xpub + 任一子私钥 → 可推全部兄弟私钥`。但本设计下**子私钥永不离开气隙机**（无泄漏路径），且气隙机一旦失陷即等于主种子失陷（无论硬化与否都全损）→ 非硬化未增加实际风险。缓解：`coldsign` 签名后立即清零派生私钥、任何路径禁止 log/导出私钥、`xpub` 视为半机密（不公开发布）。
- **死人开关告警可被阻断（Q5）**：持续 root 可切断在线机出站网络使 Telegram/邮件发不出。缓解：**告警判定逻辑运行在离机校验器上**（期望按时收到上报，静默即告警）——切网反而触发而非抑制告警；校验器同时独立查链，冷钱包出现未授权资金移动即报警。

**结论**：用户链上资金对实时 root 免疫；不可在单机绝对保证的仅「内部账本」与「提现目的地」，二者均不导致存款被盗空。

### 8.3 交易包 proto（落地契约 — v3 修订，显式密钥源）

**v3 关键变更**：`derivation_index`（隐式——冷签机靠 `from_address == cold_wallet_address` 字符串比较猜测用哪个密钥）替换为 `oneof key_source`（显式——proto 本身声明密钥来源）。意图显式化消除冷签机猜测逻辑，也消除"配置被改后静默切换到错误密钥"的隐患。

```proto
enum TxKind {                 // Q6: enum 提供编译期类型检查
  TX_KIND_UNSPECIFIED = 0;
  TX_KIND_DELEGATE    = 1;
  TX_KIND_TRANSFER    = 2;
  TX_KIND_UNDELEGATE  = 3;
}

message UnsignedTx {
  // ── 跨切面公共字段（冷签机 flat-read，oneof tx 仅用于类型特定逻辑）──
  TxKind kind         = 1;
  string from_address = 2;    // 源地址（充值地址 / 能量账户 / 冷钱包）
  string to_address   = 4;    // 目标地址（冷钱包 / 用户地址 / 能量账户）
  string amount       = 5;    // 金额（USDT/TRX，decimal string，禁 float）
  bytes  raw_tx       = 6;    // gotron-sdk 构造的待签原始交易（24h 过期，崩溃恢复窗口）
  int64  expiry_ms    = 7;    // raw_tx 过期时间戳 (ms)
  string expected_txid = 8;   // 预期 txid（idempotent broadcast）

  // ── 密钥源（显式声明，v3 新增 — 冷签机不需要猜测用哪个密钥）──
  oneof key_source {
    uint32 bip39_derivation_index = 3;  // BIP39 种子派生 m/44'/195'/0'/0/{index}
                                        // 用于: sweep transfer (分地址 0+),
                                        //       delegate/undelegate (能量账户 index=0, change chain)
    bool   cold_wallet_key       = 15;  // 冷钱包独立私钥 (-cold-wallet-key 参数, §2.4.1)
                                        // 用于: withdrawal transfer
  }

  // ── 交易类型（Q7: oneof 按 kind 拆分字段语义）──
  oneof tx {
    DelegateTx   delegate   = 10;
    TransferTx   transfer   = 11;
    UndelegateTx undelegate = 12;
  }
}

message DelegateTx {
  string energy_account = 1;  // 能量账户 TRC20 地址
  string resource       = 2;  // "ENERGY"
}
message UndelegateTx {
  string energy_account = 1;
  string resource       = 2;
}
message TransferTx {
  // v3: derivation_index 移除，改用 UnsignedTx.key_source
  string token_contract = 1;  // TRC20 合约地址（USDT），空 = TRX
  WithdrawalAuth auth   = 5;  // Q8: 仅提现时填, 归集为空
}

message WithdrawalAuth {      // Q8: coldsign 据此重建 challenge=sha256(amount|dest|nonce|user_id)
  string user_id       = 1;
  uint64 nonce         = 2;
  string credential_id = 3;   // WebAuthn 凭证 ID
  bytes  assertion     = 4;   // WebAuthn 断言
}

// ── sweep 场景的 key_source 规则 ──
//   DelegateTx   → bip39_derivation_index = 0 (能量账户, change chain: m/44'/195'/0'/1/0)
//                   from_address = energy_account, to_address = deposit_address
//   TransferTx   → bip39_derivation_index = {分地址 index}
//                   from_address = deposit_address, to_address = cold_wallet
//   UndelegateTx → bip39_derivation_index = 0 (能量账户, change chain)
//                   from_address = energy_account, to_address = deposit_address
//
// ── withdrawal 场景的 key_source 规则 ──
//   TransferTx   → cold_wallet_key = true
//                   from_address = cold_wallet, to_address = 用户白名单地址

message LegacyUnsignedTx_DEPRECATED {
  // 旧平铺结构，仅作迁移参考。v3 落地用上方 oneof key_source。
  string kind = 1; string from_address = 2; string to_address = 3;
  string amount = 4; int64 energy = 5; bytes raw_tx = 6;
  int32 derivation_index = 7;
}
message UnsignedSweepBundle {
  repeated UnsignedTx txs = 1;
  string bundle_id        = 2;    // 唯一 batch_id（跨在线机/冷签机关联）
  int64  built_at_ms      = 3;    // 构建时间（expiry 判断）
  string xpub_fingerprint = 4;    // xpub 指纹（冷签机验证身份）
}
message SignedTx {
  TxKind kind            = 1;
  string from_address    = 2;
  string to_address      = 3;
  string amount          = 4;
  bytes  signed_tx_data  = 5;    // 已签名的 TRON 交易数据
  string tx_hash         = 6;    // 匹配 UnsignedTx.expected_txid
}
message SignedSweepBundle {
  repeated SignedTx txs  = 1;
  string bundle_id       = 2;    // 匹配 UnsignedSweepBundle.bundle_id
  int64  signed_at       = 3;    // 签名时间戳 (ms)
  string xpub_fingerprint = 4;   // 已验证的 xpub 指纹
}
```

### 8.4 提现（后续功能，同一模型）

- 提现请求在线机构建 `UnsignedTx.transfer{from=冷钱包, to=用户地址, auth=WithdrawalAuth{...}}`（§8.3 oneof; 无热钱包）。
- `coldsign` 展示 金额/目的地 供人工核对，强制单笔/单日限额，签名后回传广播。
- 提现目的地非固定，无法用白名单锁定 → 依赖 R 限额 + 人工核对（8.2 已声明的残余风险）。

### 8.5 部署与运维

- 在线机全盘加密（LUKS）、DB 不对外暴露、备份加密且密钥异地存放（防非实时威胁：拖库/备份泄漏/磁盘失窃）。
- `coldsign`/`hdgen` 二进制与种子只在气隙机；USB 介质单向、专用。
- 监控与广播进程以独立低权限用户运行；`system_config` 中均为公开值，无机密。

## 9. 内部账本完整性 + 提现授权（权威约束）

> §8 保证「链上资金」对实时 root 免疫；本节保证「内部账本」与「资金出口（提现）」也不被利用。
> 核心洞察：**账本被篡改只有在提现那一刻才变成真实损失**，故控制点是「偿付能力不变量」+「提现授权」。

### 9.1 哈希链 append-only 账本流水（tamper-evident）

在 `wallet_transactions` 之上引入不可篡改流水，每条哈希链接前一条：

```sql
-- 新增列 (或独立 ledger_journal 表)
ALTER TABLE wallet_transactions
  ADD COLUMN seq         BIGINT GENERATED ALWAYS AS IDENTITY,  -- 全局单调序号
  ADD COLUMN prev_hash   BYTEA NOT NULL,                       -- 前一条的 entry_hash
  ADD COLUMN entry_hash  BYTEA NOT NULL,                       -- = SHA256(prev_hash || 规范化本条字段)
  ADD COLUMN idem_key    TEXT UNIQUE;                          -- 幂等键 (tx_hash / withdrawal_id / 业务ID)
```

- `entry_hash = SHA256(prev_hash || seq || wallet_id || tx_type || amount || balance_after || idem_key)`。
- **append-only 触发器**：`BEFORE UPDATE OR DELETE ON wallet_transactions` → `RAISE EXCEPTION`（DB 层禁改，挡住非 root 篡改；root 仍可绕过但会断链留痕）。
- **实时外发**：每写入一条，立即把 `(seq, entry_hash)` 推送到主机外的 append-only 汇聚点（管理员 Telegram/邮件/外部只追加存储）。已离机的历史条目 root 改不了 → 可事后重建、定位篡改点。

### 9.2 偿付能力不变量 + 链下校验器 + 死人开关（核心）

**不变量**：`Σ(user_wallets.balance + frozen_balance) ≤ 链上托管`
其中 `链上托管 = Σ(分地址 USDT) + 冷钱包 USDT − 待处理提现`。

- **链下独立校验器**（跑在管理员设备/冷机，**不在被攻陷的在线机上**）：
  1. 独立查公开链算出链上托管（分地址集由 xpub 派生，管理员持 xpub）——**不可伪造**。
  2. 经认证 admin RPC 拉平台自报总负债 + 最新 `(seq, entry_hash)`。
  3. 负债 > 托管 + 容差 → **资不抵债告警**；哈希链断裂/回退 → **篡改告警**。
- **死人开关**：校验器期望每 N 小时收到心跳/报告，**漏收即告警**（攻击者关掉上报反而暴露）。
- 告警走**出站**通道（Telegram bot / 邮件；不违反"禁 REST endpoint"，属出站调用）。

### 9.3 DB 纵深防御（落地必做）

- `user_wallets.balance` 加 **`CHECK (balance >= 0)`**（当前缺失 → 见 §9.6）。
- **冻结模式**：提现发起 → `balance -= X, frozen_balance += X`（同事务）；提现完成 → `frozen_balance -= X`；提现取消 → 反向。杜绝"发起后余额仍可花"。
- 所有记账走 `idem_key` 幂等：重复请求 `ON CONFLICT (idem_key) DO NOTHING`，防重放/重试双记。

### 9.4 提现授权：用户签名 + 冷签器验证（WebAuthn）

**信任锚移到用户端，在线机降级为不可信中继。**

```
1. 用户端用 WebAuthn/passkey 对 (amount + dest_address + nonce + user_id) 签名
2. 在线机仅转发: 构建 UnsignedTx.transfer{to=dest, auth=WithdrawalAuth{user_id,nonce,credential_id,assertion}} (§8.3 oneof)
3. coldsign 在气隙机验证:
   - 按 auth.credential_id 查「冷签器自持」公钥库取公钥 → 重建 challenge=sha256(amount|dest|nonce|user_id) → 验 WebAuthn 断言 (不信在线 DB 的公钥; 未知 credential_id 拒签)
   - 校验 dest ∈ 用户提现白名单; 校验单笔/单日限额 (硬编码在气隙机)
   - 屏幕展示 金额/目的地/用户 供 operator 人工核对
   - 全部通过才签名; 否则拒签
4. 在线机广播已签名交易
```

- **root 伪造不出用户 WebAuthn 断言** → 无法把提现改道到自己地址。
- **提现地址/用户公钥的新增或轮换**：需 2FA + 邮件/短信带外确认 + 冷却期（24–48h），并写入 §9.1 哈希链流水（防 root 悄悄换白名单/公钥）。
- 残余：提现目的地非固定无法白名单锁死到单一地址 → 由「用户签名 + 冷签器限额 + 人工核对」共同兜底（8.2 已声明）。

### 9.5 红线（R7–R12，GLM 落地强制）

- **R7** 每笔记账必须有 `idem_key` 且唯一；重复请求不得双记。
- **R8** `wallet_transactions` 哈希链 + DB 层 append-only 触发器；`(seq, entry_hash)` 实时外发。
- **R9** `user_wallets.balance` 加 `CHECK (balance >= 0)`；提现走冻结模式。
- **R10** 偿付能力校验器**必须运行在在线机之外**；含死人开关（漏报即告警）。
- **R11** 提现必须携带用户 WebAuthn 断言，由 `coldsign` 用自持公钥验证后才签。
- **R12** 提现白名单/用户公钥的变更必须带外确认 + 时间锁 + 入哈希链流水。

### 9.6 现状差距（审计所得 — 闭合状态）

代码审计（`wallet_repo.go` / `deposit_service.go` / migrations 147–148）发现与本 ADR 的偏差。

| 差距 | 现状 | 必改 | 闭合状态 |
|---|---|---|---|
| **在线机仍持私钥** | ~~`deposit_service.ImportDepositAddresses` 仍用 `PurposeDepositPrivKey`~~ | ~~改为 watch-only~~ | ✅ 已闭合：watch-only xpub 派生，DB 不存私钥，`derive_priv.go` 仅 `cmd/coldsign`/`cmd/hdgen` 使用 |
| **余额可为负** | ~~migration 147 无 `CHECK (balance >= 0)`~~ | ~~加 CHECK + 冻结模式~~ | ✅ 已闭合：`CHECK (balance >= 0)` 已加，`AdjustBalanceTx` 捕获 `isCheckViolation`，冻结模式 `FreezeForWithdrawal`/`CompleteWithdrawal`/`CancelWithdrawal` 已实现 |
| **流水非真不可改** | ~~无 DB 触发器、无哈希链~~ | ~~加 append-only 触发器 + 哈希链~~ | ✅ 已闭合：`wallet_transactions` 有 `seq`/`prev_hash`/`entry_hash`/`idem_key`，`ledgerChainInsert` 实现哈希链 + `ledger_outbox` 外发 |
| **无幂等键** | ~~`AdjustBalance` 无 `idem_key`~~ | ~~全路径加 `idem_key`~~ | ✅ 已闭合：`AdjustBalanceTx` 强制 `idemKey`，`ConfirmDeposit` 用 `"deposit-"+txHash`，`FreezeForWithdrawal` 用 `"withdrawal-{id}"` |
| **无提现实现/授权** | 无提现路径与用户签名验证 | 按 §9.4 实现 | ✅ 已闭合：`webauthn_withdrawal.go` + `coldsign` R11 WebAuthn 验证 + `FreezeForWithdrawal`/`CompleteWithdrawal`/`CancelWithdrawal` |
| **无偿付能力校验** | 无链下校验器/死人开关 | 按 §9.2 实现 | ✅ 已实现、MVP 不部署：`cmd/solvency-check` 已实现（偿付能力 + 死人开关 + 篡改检测），MVP 阶段不部署运行（见 §13.2） |
| **USDT=USD 1:1** | 钱包 `currency='USD'`，充值按 USDT 数额 1:1 记 USD | 明确口径为 USDT，或书面承担脱锚风险 | ✅ 已闭合：口径明确为 USDT 1:1 USD，书面承担脱锚风险 |
| **遗留手动充值** | `deposit_requests`（migration 198）与新 HD 系统并存 | 迁移后清理（§7） | ✅ 已闭合：旧 RPC 已删除，`deposit_requests` 为死数据 |

## 10. 落地实现指南（GLM 权威执行手册 — 最优方法）

> 本节给出**具体实现方法**，非仅约束。GLM 按此逐 Phase 执行；每 Phase 的「验收」是硬门槛。
> 库选型即最优选：BIP32 用 `github.com/btcsuite/btcd/btcutil/hdkeychain`，TRON 用 `github.com/fbsobreira/gotron-sdk`，WebAuthn 用 `github.com/go-webauthn/webauthn`。

### 10.1 Phase A — 在线转 watch-only（消灭在线私钥，R1）

**改法（最优）：地址派生只用 xpub，删除私钥路径。**

```go
// internal/hdwallet/xpub.go  (新增)
// account-level xpub = m/44'/195'/0'/0 的扩展公钥
func DeriveAddress(accountXpub string, index uint32) (string, error) {
    ext, err := hdkeychain.NewKeyFromString(accountXpub) // 仅公钥
    if err != nil { return "", err }
    child, err := ext.Derive(index)                       // 非硬化 CKDpub
    if err != nil { return "", err }
    pub, err := child.ECPubKey()
    if err != nil { return "", err }
    return tronAddressFromPubKey(pub), nil                // keccak256(pub[1:])[12:] → 0x41 → base58check
}
func XpubFingerprint(xpub string) string { return hex(sha256(xpub)) }
```

- **删除**：`user_deposit_addresses.encrypted_privkey` 列、`model.DepositAddress.EncryptedPrivkey`、`secrets.PurposeDepositPrivKey`、`DepositService.ImportDepositAddresses` 里所有解密/私钥校验分支（`deposit_service.go:107-167` 重写为仅校验地址格式 + index）。
- **启动校验（R5）**：`main.go` 装配时 `if XpubFingerprint(cfg.DepositXpub) != cfg.DepositXpubFingerprint { log.Fatal }`。
- **CI 断言（R1, §11 F）**：**在线侧**`internal/{hdwallet,service,repository,chain,connect,sweep}` + `cmd/server/` 下 grep 不到 `PrivateKey|Mnemonic|Seed|Decrypt`；**离线工具**`cmd/hdgen/`、`cmd/coldsign/` 豁免（应持私钥, 但仅存于气隙机二进制）。
- **验收**：全代码库无在线私钥；已知 xpub+index → 已知地址（BIP44 测试向量）；篡改 xpub 拒启动。

### 10.2 Phase A — `cmd/hdgen` 与 `cmd/coldsign`（气隙机，R2）

```
cmd/hdgen (离线):  生成助记词 → seed → 派生 account xpub
  → 导出: xpub, fingerprint (proto XpubExport, 无私钥)
  另: 独立生成冷钱包密钥对 → 冷钱包私钥单独保管
cmd/coldsign (离线): 读 UnsignedSweepBundle(proto)
  1) seed := mnemonic (启动时手输, 不落盘)
     cold_sk := 冷钱包私钥 (-cold-wallet-key hex 参数, §2.4.1)
  2) 每条 tx 按 UnsignedTx.key_source (R3b) 选择私钥:
     - key_source = bip39_derivation_index X:
         Transfer → m/44'/195'/0'/0/X (归集分地址)
         Delegate/Undelegate → m/44'/195'/0'/1/0 (能量账户, 固定路径; key_source index 仅用于验证)
     - key_source = cold_wallet_key:
         → 用 cold_sk 签名 (提现, §2.4.1)
     - key_source 未设置 → abort (proto 不合法)
  3) 白名单(R4): TransferTx + 无 auth → to 必须==cold_wallet_address, 否则 abort;
                 TransferTx + 有 auth → 验 WebAuthn 断言 + dest∈白名单 + 限额
  4) 打印 tx 类型/from/to/amount 供 operator 核对
  5) 用对应私钥签 raw_tx → SignedTx{signed_tx_data, tx_hash}
  6) 输出 SignedSweepBundle(proto)
```

- **验收**：白名单命中非冷地址即拒签并退出非零；签名产物 `expected_txid` 与在线广播回执一致。

### 10.3 Phase B — 监控全地址 + 单实例（R6，修 §9.6）

- `monitor.loadAddresses`：`ListAssignedAddresses` → **`ListAllDerivedAddresses`**（含 ASSIGNED/RETIRED；§11 Q1 按需派生无 AVAILABLE）。命中已退役地址 → 正常入账；命中未知地址 → 记 `MANUAL_REVIEW`。
- worker 启动：`SELECT pg_try_advisory_lock(<const>)`，拿不到锁则不启动 monitor/broadcaster/reconcile。
- `last_scanned_block` 部署脚本设为当时链高。
- **验收**：向已退役/已派生地址打款可捕获入账；起两个进程仅一个跑 worker。

### 10.4 Phase C — 归集：构建/广播/状态机/防双花（§2.7，Admin 手动触发）

**归集为 admin 手动触发，不是自动定时周期。** 自动部分仅限于：构建 UnsignedBundle、按序广播+确认、状态机追踪。触发权在管理员。

> **v3 定案（2026-07-22）**：`runCycle` 中的自动 `buildPendingBundles` 调用已删除。Worker 周期仅保留：ReconfirmSweeping、MarkStuckSweeping、expireStalePendingSign、resumeBroadcasting（崩溃恢复）。构建未签名 Bundle 仅通过 Admin RPC `ExportUnsignedBundle` 手动触发。未配置 sweep TRON gRPC 客户端时，`sweepWorker` 优雅禁用（`handlers.go` 中 `sweepTronClient == nil` → 不创建 Worker → `main.go` 不启动 goroutine）。

```go
// internal/sweep/builder.go
BuildUnsignedBundle(addrs []Addr) (*antv1.UnsignedSweepBundle, error) // gotron-sdk 造 raw_tx, 不签名
// internal/sweep/broadcaster.go
BroadcastBundle(signed *antv1.SignedSweepBundle) error  // 按序: delegate→等确认→transfer→等确认→undelegate
// internal/sweep/state.go
//  - 广播后 GetTransactionInfoByID 判定 SUCCESS/FAILED/未知(保持SWEEPING)
//  - 重试前检查链上出账(防双花): TronGrid 查 from→cold_wallet 转账记录
```

**保留：** Builder、Broadcaster、StateMachine、CheckDoubleSpend、ReconfirmSweeping（追踪已广播 bundle 状态）。

**已删除：** 自动 `buildPendingBundles` 定时器调用（v3 定案，2026-07-22）。
**保留：** `expireStalePendingSign` — 清理 24h 未冷签的 stale bundle，释放地址。
**保留：** `resumeBroadcasting` — 崩溃恢复。Admin 导入签名包后广播过程中服务重启，需要自动恢复。
**保留：** `ReconfirmSweeping` — 追踪已广播但未确认的腿状态。

**验收：** Admin Dashboard 选地址 → 构建 Bundle → 导出 → 冷签 → 导入 → 系统自动广播 3 腿 → Dashboard 显示进度。模拟 DB 写 tx_hash 前崩溃 + 链上已成功 → 重试不产生第二笔。

### 10.5 Phase D — 账本完整性（R7/R8/R9）

**migration（新）**：
```sql
ALTER TABLE user_wallets ADD CONSTRAINT chk_balance_nonneg CHECK (balance >= 0);
ALTER TABLE wallet_transactions
  ADD COLUMN seq BIGINT GENERATED ALWAYS AS IDENTITY,
  ADD COLUMN prev_hash BYTEA, ADD COLUMN entry_hash BYTEA,
  ADD COLUMN idem_key TEXT UNIQUE;
CREATE OR REPLACE FUNCTION wt_no_mutate() RETURNS trigger AS $$
  BEGIN RAISE EXCEPTION 'wallet_transactions is append-only'; END; $$ LANGUAGE plpgsql;
CREATE TRIGGER wt_append_only BEFORE UPDATE OR DELETE ON wallet_transactions
  FOR EACH ROW EXECUTE FUNCTION wt_no_mutate();
```

**改 `AdjustBalanceTx`（最优：链尾在同事务内串行）**：
```
BEGIN
  -- 幂等: 若 idem_key 已存在直接返回 (ON CONFLICT DO NOTHING + 命中即 no-op)
  SELECT entry_hash INTO prev FROM wallet_transactions ORDER BY seq DESC LIMIT 1 FOR UPDATE; -- 链尾锁
  UPDATE user_wallets SET balance = balance + $amt ...;   -- CHECK 保证 >=0
  entry_hash := sha256(prev || seq || wallet_id || tx_type || amount || balance_after || idem_key)
  INSERT wallet_transactions(..., prev_hash=prev, entry_hash, idem_key)
  INSERT ledger_outbox(seq, entry_hash)   -- 供实时外发
COMMIT
```

- **实时外发**：`cmd/ledger-shipper` 或后台 goroutine 读 `ledger_outbox` → 推 Telegram/邮件 → 标记已发。
- **验收**：并发记账哈希链连续无断裂；UPDATE/DELETE 流水被 DB 拒绝；重复 idem_key 不双记；debit 超额被 CHECK 拒绝。

### 10.6 Phase D — 偿付能力校验器（R10，跑在在线机之外）

```
cmd/solvency-check (在管理员设备/冷机运行, 独立):
  custody   = Σ TronGrid.GetTRC20Balance(DeriveAddress(xpub,i)) for all i  +  冷钱包余额
  liability = adminRPC.GetLedgerSummary().total_liabilities   // 平台自报
  tip       = adminRPC.GetLedgerSummary().latest_seq/entry_hash
  IF liability > custody + tol      → ALERT 资不抵债
  IF tip.seq 未在 N 小时内推进        → ALERT 死人开关
  IF 本地记录的 entry_hash 链与 tip 不一致 → ALERT 篡改
```

- 新增 admin RPC `GetLedgerSummary`（ConnectRPC）：返回 `total_liabilities`、`latest_seq`、`latest_entry_hash`。
- **验收**：手工把某余额调高 → 校验器在下一轮报资不抵债；停掉外发 → 死人开关报警。

### 10.7 Phase E — 提现授权（R11/R12，WebAuthn）

```
注册期: 用户创建 passkey → 服务器存 credential 公钥; 同步一份到 coldsign 自持库
提现:
  1) challenge = sha256(amount|dest|nonce|user_id)
  2) 用户 passkey 签 challenge → assertion
  3) 在线机: 存 withdrawal(PENDING) + 冻结(balance-=X, frozen+=X, R9) + 带 assertion 构建 UnsignedTx
  4) coldsign: go-webauthn 用「自持公钥」验 assertion + dest∈白名单 + 限额 → 签
  5) 广播成功 → withdrawal(DONE) + frozen-=X;  失败 → 解冻
地址/公钥变更: 2FA + 邮件确认 + 冷却期(≥24h) + 写入哈希链流水
```

- **验收**：伪造 assertion（无用户私钥）被 coldsign 拒签；改 dest 后 assertion 验签失败；提现全程余额守恒（balance+frozen 不变直至完成）。

### 10.8 交付顺序与验收矩阵

| Phase | 交付 | 对应红线 | 关键验收 |
|---|---|---|---|
| A | watch-only + hdgen/coldsign | R1 R2 R4 R5 | 代码库零私钥；白名单拒签；xpub 指纹校验 |
| B | 全地址监控 + advisory lock | R6 | 已退役/已派生地址可入账；单实例 |
| C | 归集构建/广播/状态机 | R3 (proto) | 崩溃重试不双花 |
| D | 哈希链+CHECK+幂等+校验器 | R7 R8 R9 R10 | 链不可改；不可透支；资不抵债/死人开关告警 |
| E | WebAuthn 提现 | R11 R12 | 伪造断言拒签；余额守恒 |

**每个 Phase 结束必跑**：`go build ./...` + `cd backend && go run ./tools/check-file-lines --strict` + `bash scripts/gen_capability_map.sh`（遵守 AGENTS.md Before Commit）。

## 11. 设计澄清决议（Q1–Q12，权威结论）

> 对 `0026-clarifications-needed.md` 逐条定案。GLM 以本节为准；与前文冲突处以本节最新决议为准。

### 🔴 阻塞项

**Q1 地址派生 → 定案 A：xpub 按需派生。**
- 在线机从 `deposit_xpub` 按 `index` 实时派生（`internal/hdwallet/xpub.go`），不搞预导入地址池、不需要 `AVAILABLE` 状态机。
- **index 分配**：用 PG 单调计数器（`SELECT ... FOR UPDATE` 计数器行 或专用 `SEQUENCE`），杜绝 `MAX(index)+1` 并发竞态。
- `hdgen` **保留** 种子/xpub/fingerprint 生成；**删除** 地址列表导出为权威来源（在线派生才是权威）——可选导出一份地址供一次性交叉核对。
- 理由：两方案安全性等同（都无在线私钥），A 运维/代码/schema 全面更简，即最优。

**Q2 WebAuthn 公钥分发 → 定案：USB 导出 + R12 治理，拒绝裸 TOFU。**
- 机制走 A：在线导出 `UserCredentialList`（proto）→ USB → 冷签器本地库。
- **关键（否决 B）**：在线机是不可信 root，**绝不能信在线机在 `UnsignedTx` 里现供的公钥**（否则攻击者塞自己公钥+自签断言即可盗签）。因此每个新增/变更凭证条目是 **R12 事件**：须带用户带外确认 + 冷却期，且写入 §9.1 哈希链凭证变更流水；冷签器导入时**比对离机哈希链镜像**，注入的假凭证因用户从未带外确认、且在离机镜像暴露而被发现。
- 冷签器凭证库无实际容量上限（公钥体积小）；丢失可**从哈希链凭证变更流水（已实时外发离机）重建**，非灾难——这正是凭证变更必须入账本的原因。
- 同步频率：每批提现前或按需。

**Q3 归集原子性 → 定案：持久化已签 bundle + 三腿状态 + 长过期，卡死走人工/自然解押。**
- `SignedSweepBundle` **持久化入 DB**。核心洞察：**重广播不需要私钥**（只有签名需要），故在线重启后读回已签 bundle，从首个未确认交易继续广播（按 txid 幂等）→ 原子性缺口闭合，无需再回冷机。
- `sweep_logs` **拆为 3 腿**（delegate/transfer/undelegate）各自追踪广播状态，供崩溃恢复定位。
- 构建 `raw_tx` 时设**接近 24h 上限的过期时间**，给崩溃恢复留足窗口。
- 极端卡死（超过期仍未完成）兜底二选一：管理端触发**「undelegate-only 回收」** 冷签批次，或接受 **14 天自然解押**。资金本身始终安全在分地址，无时间压力。

### 🟡 安全模型

**Q4 BIP32 非硬化级联 → 定案 A：接受并记录（已写入 §8.2）。** 见 §8.2 残余风险第 1 条：子私钥永不离机、气隙失陷=种子失陷（硬化无益）、缓解为签后清零 + 禁 log/导出 + xpub 半机密。

**Q5 死人开关可被阻断 → 定案：接受并记录（已写入 §8.2）。** 见 §8.2 残余风险第 2 条：告警判定逻辑运行在离机校验器，切网触发而非抑制；校验器独立查链兜底。

### 🟢 技术精确度（已落入 §8.3）

**Q6 kind → enum**：采纳，`TxKind` enum。
**Q7 按 kind 拆字段 → oneof**：采纳，`DelegateTx/TransferTx/UndelegateTx`。
**Q8 nonce 入 proto**：采纳，新增 `WithdrawalAuth{user_id,nonce,credential_id,assertion}`，冷签器据此重建 challenge。
**Q9 v1 残余（§2.8 热钱包）**：已修正，链上托管 = Σ(分地址) + 冷钱包，无热钱包。

### 📋 文档补全

**Q10 MANUAL_REVIEW SOP → 定案：ADR 定框架，细节入实现文档。** 框架：
- admin RPC `ListManualReviewDeposits` / `ResolveManualReview{deposit_id, action: CONFIRM|REJECT, note}`。
- 核实标准：人工在 TronScan + TronGrid 双查该 `tx_hash`，核对 `to∈已派生地址`、`contract==USDT`、`amount`、确认数≥19。
- CONFIRM → 走 `ConfirmDeposit`（带 `idem_key=tx_hash` 幂等补账，R7）；REJECT → 记 note 关闭，不入账。全部动作写哈希链流水。

**Q11 灾难恢复 → 定案：ADR 记要点，完整 runbook 独立运维文档。** 要点：
- 冷签机故障 → 用 Shamir 2-of-3 在新气隙机重建种子，恢复签名能力；`deposit_xpub` 不变，无需重派地址。
- 丢 1 枚分片 → 2-of-3 仍可恢复；随后重新分片轮换。
- 丢含已签 bundle 的 USB → **风险低**：已签归集交易目的地被硬白名单锁死（transfer→冷钱包，delegate/undelegate 仅限能量账户↔分地址），泄漏者广播也只能把钱送进冷钱包；仍按敏感处理，让其自然过期（Q3 长过期设定下最长 24h）。

**Q12 对账阈值不对称 → 定案：刻意为之（已在 §2.8 注明理由）。** 少钱=偿付能力风险更紧急（−$1），多钱多为良性（+$10）。

## 12. 运维风险缓解（落地必做）

> 以下 5 项是架构固有风险，不涉及设计变更，通过运维 SOP 和系统内轻量机制兜底。

### 12.1 xpub 运行时篡改（§8.2 残余）

**风险**：root 在运行时修改 `system_config.deposit_xpub` → 新用户地址指向攻击者钱包。R5 仅在启动校验，挡不住运行时。

**缓解**：
- 地址审计 cron：每 24h 自动用当前 `deposit_xpub` 派生最近 10 个 index 的地址，与 DB 中 `user_deposit_addresses.address` 比对；不一致 → 立即告警（Telegram/邮件）+ 拒绝新地址分配（`GetDepositAddress` 返回错误，保护新用户）。
- 配置不可变：`deposit_xpub` / `deposit_xpub_fingerprint` 应存于配置文件（`config.yaml` 或环境变量）而非 `system_config`。启动时写入 `system_config` 作为缓存，但运行时审计以配置文件为权威源。如果配置文件不可行（受当前部署约束），则审计频率提高到每 6h。
- SOP：告警触发 → 停止 `GetDepositAddress` → 核实 `deposit_xpub` 是否被改 → 从配置恢复 → 排查入侵范围。

### 12.2 能量账户 TRX 不足

**风险**：TRON 网络 Energy 价格上涨 → 质押的 TRX 不够委托整批归集 → delegate 腿 FAILED → 归集停。能量账户 TRX 耗尽会导致所有充值地址无法归集。

**缓解**：
- Sweep Dashboard 显示能量账户当前 TRX 余额和可用 Energy 量（Admin 在操作前可自行判断）。
- 构建 Bundle 前检查：若能量账户 TRX 余额不足以覆盖所选地址的委托需求 → 返回错误提示，不构建 Bundle（防止部分失败后能量锁死，§2.7）。
- Admin 手动减少归集地址数量降级处理。
- 补质押流程：Admin 评估 → 冷签名机签一笔 `FreezeBalanceV2` 追加质押 → 广播。14 天解押期意味着从决定到新增质押生效有延时，需要提前量。
- Energy 价格监控：每日记录 TRON 网络 Energy 均价（TronGrid `/wallet/getenergyprices`），在 Dashboard 展示趋势，供 Admin 预判。

### 12.3 链上监控主源不可用

**风险**：TronGrid API 故障 → 区块事件扫描停 → 充值延迟。TronScan 备源仅用于单笔验证（§2.6），不能替代区块事件扫描。

**缓解**：
- TronGrid 3 次连续失败（指数退避重试后仍失败）→ 自动切换至 TronGrid 备用 endpoint（`api.trongrid.io` → `api.shasta.trongrid.io` 或其他可用节点）→ 告警通知 admin。
- 手动降级：如果所有 TronGrid 节点不可用，admin 可切至 TronScan 分页查 Transfer 事件（功能受限：TronScan 不支持 `only_confirmed` 过滤，需自行跳过未确认区块）。`system_config.tron_scan_fallback_enabled` 控制。
- 恢复：主源恢复后自动切回，从 `last_scanned_block` 续扫，不遗漏。

### 12.4 MANUAL_REVIEW 主动告警

**风险**：deposit 进入 MANUAL_REVIEW → admin 不查看 → 用户钱已到链上但没入账 → 客服投诉才发现。

**缓解**：
- 写入触发器：每次 `INSERT deposits WHERE status='MANUAL_REVIEW'` 后，通过 `ledger_outbox` 的同一外发通道推送通知（Telegram/邮件），含 `tx_hash` + `amount` + `user_id` + `reason`。
- 定期提醒：每 6h 对账（§2.8 阶段 1）同时统计 `COUNT(deposits WHERE status='MANUAL_REVIEW' AND created_at > NOW() - INTERVAL '24h')`，> 0 则附带在对账报告中推送。
- SOP：admin 收到通知 → 打开管理端 → `ListManualReviewDeposits` → 按 §11 Q10 框架逐笔核实 → CONFIRM 或 REJECT。

### 12.5 偿付能力校验器存活保障

**风险**：`cmd/solvency-check` 是独立工具，未部署/未运行/配置错误 → 无人校验偿付能力 → 死人开关不会响。

**缓解**：
- 部署：校验器二进制 + 配置随系统一同交付（作为 release artifact），非事后补装。
- 引导：doc/runbook 明确首次部署步骤含"启动 solvency-check，确认首轮校验通过（日志输出 `solvency OK`）"。
- 自检：校验器启动时做一次空跑验证（`custody` 查询成功 + `liability` RPC 可达），失败即退出非零（部署脚本据此中断）。
- cron：管理员设备上设 `crontab`（如每 2h），日志落盘；若校验器未安装或 cron 未配置 → 部署检查清单勾选。
- 告警：校验器连续 3 轮（6h）无输出 → 运营方应收到空窗告警（如果出站通道正常）；但**最终依赖人工确认**（这是校验器自身存活问题在单机下的天花板）。

> 完整运维 SOP 见独立 runbook：`docs/runbook/hd-wallet-operations.md`。

## 13. MVP 上线档位

> MVP = 最小可信上线集。只包含「不动则不赔钱」的核心功能；推迟项不阻塞上线。

### 13.1 启用项（MVP 必须）

| 功能 | 模块 | 验收标准 |
|------|------|---------|
| 充值监控 | `chain/monitor.go` | 区块事件扫描→多源验证→自动入账；单实例 |
| 内部账本 | `wallet_repo.go` | `idem_key` 幂等 + `CHECK(balance>=0)` + 哈希链 |
| WebAuthn 提现 | `webauthn_withdrawal.go` + `withdrawal_builder.go` + `coldsign` R11 | 用户 Passkey 签名→在线机构建 UnsignedTx→冷签机验断言→广播；`runWithdrawalBuilder` 自动构建 PENDING_SIGN bundle |
| 手动归集 | `sweep/` admin RPC | Admin 触发 Export→冷签→Import→广播 |
| xpub 校验 | `main.go` 启动 | DB xpub + env 指纹校验，不匹配拒启动 |
| 地址审计 | `xpub_audit.go` | DB 地址 vs xpub 派生比对，不一致告警 |

### 13.2 推迟项（MVP 不含）

| 功能 | ADR 章节 | 推迟原因 |
|------|---------|---------|
| 自动归集 | §10.4 | 手动归集已满足 MVP |
| 哈希链外发 | §9.1 R8 | `ledger_shipper.go` 已实现（LISTEN/NOTIFY + fallback ticker），MVP 不部署运行 |
| 离机偿付校验器 | §9.2 R10 | `cmd/solvency-check` 已实现（偿付能力 + 死人开关 + 篡改检测），MVP 不部署运行 |
| 自建 Tron 节点 | §2.5 | 成本过高，TronGrid API 足够 |
| MPC 多签 | §8.2 | 单机气隙已为理论最优 |

## 14. 操作 SOP — WebAuthn 提现 + 手动归集

### 14.1 WebAuthn 提现（用户 Passkey 签名 → 冷签机验断言 → 广播）

**密钥**：冷钱包独立私钥（非 BIP39 种子），通过 `-cold-wallet-key` 传入。

1. **[用户端]** 用户在 WalletPage 发起提现：`BeginWithdrawal`（金额+目标地址）→ Passkey 签 challenge → `FinishWithdrawal`（资金冻结，状态 `SIGNED_WAITING_BUNDLE`）
2. **[在线机]** `runWithdrawalBuilder` 每 30s 自动扫描 `SIGNED_WAITING_BUNDLE` 提现 → 构建 `UnsignedTx{TRANSFER, from=cold_wallet, to=user_addr, key_source=cold_wallet_key, auth=WithdrawalAuth{...}}` → 持久化为 `PENDING_SIGN` bundle
3. **[在线机]** Admin 导出 `PENDING_SIGN` bundle → proto binary → USB
4. **[USB]** 在线机 → 气隙机
5. **[气隙机]** `coldsign -i bundle.bin -o signed.bin -cold-wallet <addr> -cold-wallet-key <hex>` → 验证 WebAuthn 断言（R11）+ 白名单 + 限额 + 屏幕核对 → 签名
6. **[USB]** 气隙机 → 在线机
7. **[在线机]** Admin 导入 `ImportSignedBundle` → 自动广播 → `CompleteWithdrawal`（扣冻结）或失败 `CancelWithdrawal`（解冻）
8. **[验证]** TronScan 确认上链 + 余额已扣 + `wallet_transactions` 有 `idem_key=withdrawal-{id}`

### 14.2 手动归集（补流动性）

**密钥**：BIP39 种子派生分地址私钥 + 能量账户私钥，均在气隙机。

1. **[在线机]** Admin 打开 Sweep Dashboard → 查看各地址余额降序 → 评估能量账户 TRX 是否足够
2. **[在线机]** 选地址 → [构建归集 Bundle] → 每地址 3 笔（delegate+transfer+undelegate）→ 导出 proto binary
3. **[USB]** 在线机 → 气隙机
4. **[气隙机]** `coldsign -i bundle.bin -o signed.bin -cold-wallet <addr>` → Mnemonic from stdin → 验证 R4 白名单 + 逐笔屏幕核对 → 签名
5. **[USB]** 气隙机 → 在线机
6. **[在线机]** Admin 导入 `ImportSignedBundle` → 自动按序广播 delegate→transfer→undelegate → Dashboard 显示进度
7. **[异常]** FAILED→可重建 Bundle 重试；超时→Worker 自动 ReconfirmSweeping；卡死→MANUAL_REVIEW；能量不足→减少地址数降级

## 15. 提现功能移除（2026-07-23）

### 15.1 移除原因

WebAuthn 作为提现授权方案存在以下问题：

1. **设备绑定问题**：当前使用 Platform 认证器（Touch ID/Windows Hello），passkey 绑定在单台设备上。用户换设备后无法提现，无恢复机制。放开为 CrossPlatform 认证器（YubiKey）仍需用户额外购买硬件。
2. **JSON 违规**：go-webauthn 库输出 JSON 格式给浏览器 `navigator.credentials.create()`，需要 `encoding/json`，违反项目 proto-only 规则。proto 化改造需将 `CredentialCreation` 结构体手动转换为 proto 消息再在前端重建，工作量大且脆弱。
3. **用户体验差**：不支持漫游认证器的用户需要购买 YubiKey；无生物识别的设备无法使用 passkey。

### 15.2 后续方案：TOTP 交易签名

提现功能后续应采用 **TOTP 交易签名** 方案：

- 用户绑定 Google Authenticator（TOTP），换机可用恢复码重新绑定
- 提现时后端生成包含「金额+目标地址+nonce」的 TOTP challenge
- 用户输入 6 位 TOTP 码确认交易
- proto 原生支持，无 JSON 问题
- 安全性足够（防重放 + 交易绑定），用户体验好
- 冷签机验签逻辑从 WebAuthn 断言验证改为 TOTP 验证，简化气隙机代码

### 15.3 移除范围

- **前端**：删除 `PasskeyManagement.tsx`、`WithdrawalPanel.tsx`、`WhitelistManagement.tsx`、`webauthn.ts`；`WalletPage.tsx` 移除三个组件引用；`WalletDropdown.tsx` 移除提现按钮；`connect.ts` 移除 WebAuthnService
- **后端**：删除 `handlers_webauthn.go`、`webauthn_handler.go`、`webauthn_service.go`、`webauthn_registration.go`、`webauthn_withdrawal.go`、`webauthn_whitelist.go`、`webauthn_test.go`；`handlers.go` 注释掉 `wireWebAuthn` 调用
- **保留**：`gen/` 下 proto 生成代码（`webauthn.pb.go`、`webauthn.connect.go`）和 repository（`webauthn_repo.go`、`withdrawal_repo.go`）不删除，避免影响其他引用，后续恢复可直接复用
- **i18n**：`wallet.passkey.*`/`wallet.withdraw.*`/`wallet.whitelist.*` key 保留在 textproto 中（不影响功能，后续恢复时可直接复用）
- **DB**：migration 210（`webauthn_withdrawal`）不回滚，表数据保留
