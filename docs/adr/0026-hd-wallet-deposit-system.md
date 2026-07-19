# ADR-0026 · HD 钱包充值系统 — 每用户独立地址 + 自动到账确认

- **状态**：Accepted
- **日期**：2026-07-17 (Proposed) / 2026-07-19 (Accepted, post-audit)
- **决策者**：Team
- **关联 spec**：无

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

采用 **HD 钱包（BIP32/BIP44）** 为每个用户派生独立的 TRC20 收款地址，配合链上监控实现自动到账确认。

### 2.1 核心架构

```
┌──────────────────────────────────────────────────────────────┐
│                     HD Wallet Layer                          │
│                                                              │
│  离线机器 (冷存储):                                           │
│    Master Seed → 批量派生地址 + 加密私钥                       │
│    主种子永不联网，存金属板/保险箱                              │
│                                                              │
│  服务器 (热存储):                                              │
│    DB 存: address + derivation_index + encrypted_privkey      │
│    DB 不存: 主种子                                             │
│    归集时: 解密私钥(内存) → 签名 → 清零                         │
│                                                              │
│    m/44'/195'/0'/0/0 → 用户A 地址                             │
│    m/44'/195'/0'/0/1 → 用户B 地址                             │
│    m/44'/195'/0'/0/N → 用户N 地址                             │
└──────────────────────────────────────────────────────────────┘
         │
         ▼
┌──────────────────────────────────────────────────────────────┐
│              Chain Monitor (链上监控)                          │
│                                                              │
│  扫描新区块 (每 3s 一个区块):                                  │
│    GET /v1/contracts/{usdt_contract}/events                   │
│      ?event_name=Transfer&block_number=N&only_confirmed=true  │
│    → 获取该区块所有 USDT Transfer 事件                         │
│    → 内存 map[address]bool 匹配用户充值地址                     │
│    → O(1) per block, 与用户数无关                              │
│                                                              │
│  记录 last_scanned_block → 重启后从断点继续                     │
│  → 等待 ≥20 区块确认 → 验证交易字段                            │
│  → 自动入账 (DB 事务) → 触发归集                               │
└──────────────────────────────────────────────────────────────┘
         │
         ▼
┌──────────────────────────────────────────────────────────────┐
│              Sweeper (归集)                                   │
│                                                              │
│  分地址收到 USDT → 解密私钥(内存) → 签名转账                   │
│  → USDT 从分地址 → 平台热钱包                                 │
│                                                              │
│  Gas 优化: 热钱包 Stake TRX 换 Energy                         │
│    → DelegateResource 委托 Energy 到分地址                    │
│    → 签名转账后 UndelegateResource 撤回                       │
│    → 归集成本 ~0 TRX (摊销), 无需给每个分地址预充 TRX           │
│    → 批量归集时也可租 Energy (按需, ~1-3 TRX/笔)               │
└──────────────────────────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────────────┐
│              Wallet (平台内部账本)                        │
│                                                          │
│  user_wallets.balance += amount_usd                      │
│  wallet_transactions 记录入账                             │
│  deposits 记录链上凭证                                    │
└─────────────────────────────────────────────────────────┘
```

### 2.2 派生路径

```
m / 44' / 195' / 0' / 0 / index
  │      │      │   │   └─ 用户序号（从 0 递增，每个用户唯一）
  │      │      │   └─ 外部链（收款地址）
  │      │      └─ 账户号（固定 0）
  │      └─ Tron coin type (BIP44)
  └─ Purpose (BIP44)
```

### 2.3 数据模型变更

#### 新增表：`user_deposit_addresses`（地址池 + 用户分配）

```sql
CREATE TABLE user_deposit_addresses (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID REFERENCES users(id),    -- NULL = 未分配 (地址池中待领取)
    address         VARCHAR(64) NOT NULL UNIQUE,  -- TRC20 地址 (Base58)
    derivation_index INT NOT NULL UNIQUE,          -- BIP44 派生 index
    encrypted_privkey BYTEA NOT NULL,              -- AES-256-GCM 加密的私钥
    network         VARCHAR(16) NOT NULL DEFAULT 'TRC20',
    status          VARCHAR(16) NOT NULL DEFAULT 'AVAILABLE',  -- AVAILABLE / ASSIGNED / RETIRED
    has_received_usdt BOOLEAN NOT NULL DEFAULT false,          -- 是否曾收到 USDT (影响首次归集 Energy 计算)
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    assigned_at     TIMESTAMPTZ                     -- 分配时间
);

CREATE INDEX idx_deposit_addresses_user_id ON user_deposit_addresses(user_id);
CREATE INDEX idx_deposit_addresses_address ON user_deposit_addresses(address);
CREATE INDEX idx_deposit_addresses_status ON user_deposit_addresses(status);
```

**地址池模式**：离线预生成 N 个地址导入 DB，status='AVAILABLE'，user_id=NULL。
用户需要地址时，原子 claim：
```sql
UPDATE user_deposit_addresses
SET user_id = $1, status = 'ASSIGNED', assigned_at = NOW()
WHERE id = (
  SELECT id FROM user_deposit_addresses
  WHERE status = 'AVAILABLE'
  ORDER BY derivation_index
  FOR UPDATE SKIP LOCKED
  LIMIT 1
)
RETURNING address, derivation_index;
```
`FOR UPDATE SKIP LOCKED` 保证并发安全，不会两个用户拿到同一个地址。

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

```sql
CREATE TABLE sweep_logs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    deposit_address_id UUID NOT NULL REFERENCES user_deposit_addresses(id),
    tx_hash         VARCHAR(64) UNIQUE,                -- 归集交易 hash (NULL 直到广播成功)
    amount          NUMERIC(20,8) NOT NULL,            -- 归集 USDT 金额
    energy_used     BIGINT NOT NULL,                   -- 实际消耗 Energy
    status          VARCHAR(16) NOT NULL DEFAULT 'PENDING',  -- PENDING / SWEEPING / DONE / FAILED / MANUAL_REVIEW
    error_message   TEXT,                              -- 失败原因
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at    TIMESTAMPTZ
);

CREATE INDEX idx_sweep_logs_address_id ON sweep_logs(deposit_address_id);
CREATE INDEX idx_sweep_logs_status ON sweep_logs(status);
```

#### 新增表：`wallet_secrets`（热钱包私钥加密存储）

```sql
CREATE TABLE wallet_secrets (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    purpose         VARCHAR(32) NOT NULL,          -- 'hot_wallet'
    encrypted_data  BYTEA NOT NULL,                -- AES-256-GCM 加密后的私钥
    key_version     INT NOT NULL,                  -- KEK 版本号
    is_active       BOOLEAN NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
-- 注: 主种子不在服务器上, 此表不存主种子
-- 注: 分地址私钥存在 user_deposit_addresses.encrypted_privkey, 不在此表
```

### 2.4 主种子安全策略

**核心原则：主种子永不触网。**

采用**离线预生成地址批次**模式，服务器永远不接触主种子：

```
离线机器 (空气隔离):
  1. 生成 24 词助记词 (BIP39)
  2. 用主种子派生 N 个地址 + 私钥 (BIP44, m/44'/195'/0'/0/0..N)
  3. 用 AES-256-GCM 加密每个私钥 (KEK 由 KMS 管理)
  4. 导出: address + encrypted_privkey + index → 导入服务器 DB
  5. 主种子留在冷存储 (金属板/保险箱, Shamir 分片 3-of-2)
  6. 用完 N 个地址后再生成下一批

服务器:
  DB 存: address + derivation_index + encrypted_privkey
  DB 不存: 主种子
  归集时: KMS 解密私钥 → 内存签名 → crypto.ZeroBytes 清零
```

**导入时地址-私钥匹配验证：**

导入 .bin 文件时，服务器对**每条**记录执行确定性验证：
1. 用 KEK 解密 encrypted_privkey → 得到 32 字节私钥
2. 从私钥派生 TRON 地址 (`AddressFromPrivateKey`)
3. 比对派生地址与声明地址是否一致
4. 不一致 → 拒绝整个批次（防止篡改 .bin 文件注入地址-私钥不匹配的条目）
5. 验证完毕立即清零私钥字节

这确保即使 .bin 文件被篡改（地址被替换但私钥未换），也无法导入——
因为用户存入的资金将永远无法归集（私钥不匹配地址）。

| 层级 | 措施 | 实现方式 |
|---|---|---|
| **生成** | 离线生成 | 空气隔离机器生成 24 词助记词，永不联网 |
| **存储** | 冷存储 | 助记词刻金属板存保险箱，Shamir 分片 3-of-2 |
| **派生** | 离线批量派生 | 离线机器派生地址+私钥，加密后导入服务器 |
| **服务器** | 不存主种子 | 服务器只存 address + encrypted_privkey，主种子不在服务器上 |
| **私钥加密** | KMS 管理解密密钥 | AES-256-GCM 加密私钥，KEK 不在代码/配置/DB 中 |
| **运行时** | 内存使用 | 归集时解密私钥到内存，用完 `crypto.ZeroBytes()` 清零 |
| **轮换** | 支持轮换 | 新种子生成新地址批次，旧地址继续收款，旧私钥保留用于旧地址归集 |

**安全优势**：服务器被入侵 → 攻击者只能拿到加密的私钥（无 KMS 无法解密），主种子完全安全。

### 2.5 链上监控流程

**采用区块事件扫描，非逐地址轮询。** 复杂度 O(1) per block，与用户数无关。

> **Push-First Architecture 合规例外**：项目规则禁止 polling，但 TronGrid API 无 push/webhook 能力，自建 Tron 全节点 + ZeroMQ/Kafka 事件订阅成本过高（月 $200-1000 + 运维）。当前阶段采用区块事件扫描（每 ~3s 一次 API 调用）作为**唯一可行的近似 push 方案**。未来升级路径：自建 Tron 节点 → ZeroMQ 事件订阅 → 真正的 push 模式。

```
1. 启动时加载用户充值地址集合到内存 map[string]uuid.UUID
   (address → user_id, ASSIGNED 状态的地址)

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

8. 触发归集:
   - NATS 发布归集任务
   - 归集 worker 消费: 解密私钥 → 委托 Energy → 签名转账 → 清零私钥
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

### 2.7 归集策略

**采用 Energy 委托模式，非 TRX 预充。** 成本对比：

| 方式 | 每笔归集成本 | 说明 |
|---|---|---|
| 直接 burn TRX | ~6.5 TRX (~$1.85) | 无 Energy，网络直接烧 TRX，最贵 |
| TRX 预充到分地址 | ~6.5 TRX per address | 同上，且每个地址都要充，资金占用大 |
| **Stake TRX 换 Energy + 委托** ✅ | ~0 TRX (摊销) | 热钱包 Stake TRX 生成 Energy，委托给分地址用 |
| 租 Energy (备选) | ~1-3 TRX (~$0.85) | 按需租用，批量归集时灵活 |

**Energy 需求动态计算：**

USDT 合约受 Dynamic Energy Model (DEM) 影响，Energy 消耗不固定：

| 场景 | 基础 Energy | DEM 因子 (~1.3x) | 实际 Energy |
|---|---|---|---|
| 常规归集 (地址已持有 USDT) | ~65,000 | ×1.3 | ~84,500 |
| 首次归集 (地址从未持有 USDT) | ~130,000 | ×1.3 | ~169,000 |

委托 Energy 量必须根据 `has_received_usdt` 字段动态计算：
```go
energyNeeded := 65000 * demFactor // ~84,500
if !addr.HasReceivedUSDT {
    energyNeeded = 130000 * demFactor // ~169,000
}
// 委托 energyNeeded + 10% buffer
```

**Energy 委托归集流程：**

```
触发条件: 分地址 USDT 余额 > 0 且有 ≥20 确认
归集目标: 平台热钱包地址

流程:
  1. 从 DB 读取分地址的 encrypted_privkey + has_received_usdt
  2. 计算所需 Energy (首次 169k, 常规 84.5k, +10% buffer)
  3. KMS 解密 → 内存中得到私钥
  4. 热钱包 DelegateResource 委托计算出的 Energy 量到分地址
  5. 用私钥签名 USDT TRC20 转账 → 热钱包
  6. 确认转账成功 (等待区块确认)
  7. 热钱包 UndelegateResource 撤回 Energy
  8. crypto.ZeroBytes() 清除内存中的私钥
  9. 更新 has_received_usdt = true (如果首次)
  10. 在 sweep_logs 中记录归集结果 (状态机: PENDING→SWEEPING→DONE/FAILED)
```

**归集幂等与重试安全（确定性验证 — 第一性原理）：**
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
  - SWEEPING 无 tx_hash → FAILED (广播前卡死, 可安全重试)
  - PENDING → FAILED (未开始, 可安全重试)

- DONE 状态不再重复归集

**批量归集优化：**
- 多个分地址待归集时，批量委托 Energy
- 或租 Energy 按小时计费，一个租期内归集多个地址

**注意事项：**
- 归集后地址保留，支持用户多次充值到同一地址
- 最小充值金额 1 USDT，低于此金额归集成本不划算

### 2.8 对账机制

**两阶段对账，避免 API 调用量超限：**

```
阶段 1 — 内部对账 (每 6h, 无 API 调用):
  预期链上余额 = Σ(deposits WHERE status='CONFIRMED' AND amount)
               - Σ(sweep_logs WHERE status='DONE' AND amount)
  DB 余额 = Σ(user_wallets.balance)

  预期 == DB → ✅ 内部账本一致, 无需查链上
  预期 != DB → ⚠️ 内部不一致, 查 DB 事务完整性

阶段 2 — 链上对账 (每 24h, 00:00 UTC):
  链上余额:
    Σ(所有 ASSIGNED 分地址 USDT 余额)  -- TronGrid API 查询
    + 热钱包 USDT 余额
    + 冷钱包 USDT 余额
  =?
  预期链上余额 (阶段 1 计算)

  差异 = 0 → ✅ 链上与内部一致
  差异 > 0 → 有未入账的充值 → 排查链上监控
  差异 < 0 → 有未授权的扣款 → 紧急排查

  差异 > $10 或 < -$1 → 立即告警
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
- 链上自动确认，7×24 无需人工审核
- 用户无需填 txHash，体验更好
- 为后续提现功能奠定基础

### 负面
- 需要离线管理主种子和批量派生地址
- 需要 Stake TRX 获取 Energy (资金锁定 14 天解押期)
- 归集涉及链上交易签名 (私钥在内存短暂存在)
- 实现复杂度较高

### 中性
- 需要新增链上监控服务
- 需要定期对账机制
- DB schema 变更

## 5. 实施约束

### 5.1 技术栈

| 组件 | 选型 | 说明 |
|---|---|---|
| HD 派生 + Tron 地址 + 交易签名 | `github.com/fbsobreira/gotron-sdk` | 内置 BIP39/BIP44, Tron 地址派生, TRC20 转账签名, Energy 委托 |
| 链上查询 | TronGrid API (主) + TronScan API (备) | 多源交叉验证 |
| 加密 | 复用 `internal/secrets/vault.go` | 新增 `PurposeDepositPrivKey` |
| 监控 | in-process goroutine + NATS | 链上区块事件扫描 worker |

### 5.2 新增 secrets Purpose

```go
// internal/secrets/vault.go
const (
    PurposeMTPassword      Purpose = "mt-password"       // 已有
    PurposeMTAPIToken      Purpose = "mtapi-token"       // 已有
    PurposeBrokerCookie    Purpose = "broker-cookie"     // 已有
    PurposeDepositPrivKey  Purpose = "deposit-privkey"   // 新增: 用户分地址私钥加密
    PurposeHotWalletKey    Purpose = "hot-wallet-key"    // 新增: 热钱包私钥
)
// 注: 主种子不在服务器上, 无需 PurposeMasterSeed
```

### 5.3 API 变更

#### 新增 RPC

```proto
// 获取用户的充值地址（幂等：已有则返回，无则从地址池 claim）
// 逻辑: 先查 user_deposit_addresses WHERE user_id=$1 AND status='ASSIGNED'
//       有 → 返回; 无 → 原子 claim 一个 AVAILABLE 地址
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
│   ├── wallet.go           # HD 派生逻辑 (使用 gotron-sdk/pkg/keys)
│   ├── tron.go             # Tron 地址生成 + TRC20 交易签名
│   └── wallet_test.go
├── chain/
│   ├── monitor.go          # 链上监控 worker
│   ├── verify.go           # 交易验证 + 多源交叉
│   ├── tron_grid.go        # TronGrid API client
│   ├── tron_scan.go        # TronScan API client (备源)
│   └── monitor_test.go
├── sweeper/
│   ├── sweeper.go          # 归集 worker
│   ├── energy.go           # Energy 委托/撤销 + 租赁
│   └── sweeper_test.go
├── reconcile/
│   ├── reconcile.go        # 两阶段对账 (内部 6h + 链上 24h)
│   └── reconcile_test.go
├── service/
│   ├── deposit_service.go  # 修改: 自动确认逻辑
│   └── wallet_service.go   # 修改: 新增地址管理
├── repository/
│   ├── deposit_address_repo.go  # 新增
│   └── wallet_secrets_repo.go   # 新增
└── connect/user/
    └── deposit_handler.go  # 修改: 新增 GetDepositAddress RPC

backend/cmd/
└── hdgen/
    └── main.go             # 离线地址批量生成 CLI 工具
                              # 生成助记词 → 派生 N 个地址 → 加密私钥 → 输出 JSON 导入文件
                              # 在空气隔离机器上运行, 输出文件通过 USB 导入服务器
```

### 5.5 实施阶段

| 阶段 | 内容 | 预估工期 | 依赖 |
|---|---|---|---|
| **Phase 1** | HD 钱包派生(离线工具) + 用户地址分配 + GetDepositAddress RPC + DB migration | 3 天 | 无 |
| **Phase 2** | 区块事件扫描 + 自动确认入账 + txHash 唯一约束 + 断点恢复 | 3 天 | Phase 1 |
| **Phase 3** | 归集 worker + Energy 委托/撤销 + 归集状态机 | 3 天 | Phase 2 |
| **Phase 4** | 多源验证 + 对账机制 (每 6h) | 2 天 | Phase 2 |
| **Phase 5** | 前端改造（显示独立地址 + QR 码） | 2 天 | Phase 1 |
| **总计** | | **~13 天** | |

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
('hot_wallet_address', ''),                          -- 热钱包地址
('stake_trx_amount', '5000'),                        -- Stake TRX 换 Energy 的数量
('reconcile_alert_threshold', '10'),                 -- 对账告警阈值 USD
('reconcile_interval_hours', '6'),                   -- 对账频率 (小时)
('last_scanned_block', '0'),                         -- 链上监控最后扫描的区块号
('sweep_batch_size', '10'),                          -- 批量归集每批数量
('sweep_min_confirmations', '20'),                   -- 归集所需最小确认数
('address_pool_min_threshold', '100'),               -- 可用地址低于此值时告警
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
- 私钥加密/解密往返测试
- 私钥内存清零验证
- txHash 唯一约束测试（重复入账应失败）
- 重放入账测试（同一交易不应入账两次）
- 地址池并发 claim 测试（两个并发请求不会拿到同一地址）
- 地址池耗尽告警测试（可用地址 < 阈值时触发告警）

### 6.4 对账验证
- 模拟差异场景 → 验证告警触发
- 日常对账 cron 执行 → 验证报告输出

## 7. 上线前迁移

HD 钱包上线前需要完成的准备工作：

1. **清理旧 PENDING**：管理员将 `deposit_requests` 中所有 PENDING 请求审批完毕
2. **移除旧 RPC**：删除 CreateDeposit / ApproveDeposit / RejectDeposit RPC 和对应 handler
3. **现有用户**：首次访问充值页面时从地址池 claim 独立地址
4. **旧收款地址**：从 `system_config` 中移除，不再使用
5. **地址池补充**：管理员通过离线工具生成新批次地址，导入 DB，监控自动检测可用地址补充
