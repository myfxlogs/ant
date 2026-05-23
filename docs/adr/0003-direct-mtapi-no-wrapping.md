# ADR-0003 · mtapi 直连，不再二次包装

- **状态**：Accepted
- **日期**：2026-05-23
- **关联 spec**：`docs/spec/10-mt-adapter.md`、`docs/spec/16-mtapi-quirks-register.md`

## 1. 背景

ant v1 在 mtapi.io gRPC 之上自建了 `internal/mt4client` (1559 行) 与 `internal/mt5client` (1490 行)，提供 connection / pool / search / stream / subscription / trading 等子模块。

但仔细分析这两个包：

| v1 子模块 | 真实价值 | 评估 |
|---|---|---|
| `connection.go` | dial + login | ⭐⭐ 必要但简单 |
| `pool.go` (各 224 行) | 连接池 | ⭐ 与 mdgateway.Manager 重叠 |
| `connection_methods.go` | 包装 mtapi Connect | ⭐⭐ 必要 |
| `account_methods.go` | 余额/持仓查询 | ⭐⭐ 必要 |
| `search_methods.go` | broker/symbol 搜索 | ⭐ 应该用 mdgateway/normalizer |
| `stream_methods.go` | quote stream | ⭐⭐⭐ 核心 |
| `subscription_methods.go` | symbols 订阅 | ⭐ 与 stream_methods 高度重叠 |
| `trading_methods.go` | 下单 | ⭐⭐⭐ 核心 |
| `types.go` | DTO | ⭐⭐ 必要但应在 mdtick |

**总评估**：3049 行里大约 **600 行核心** + **2400 行重复抽象**（pool 与 manager 重复 / search 与 normalizer 重复 / subscription 与 stream 重复）。

alfq 的做法（参考）：
- adapter/mt4 + adapter/mt5 各 ~80 行
- 直接调 mtapi proto，无中间层
- pool / search / subscription 全部上移到 mdgateway 统一处理

## 2. 决策

**v2 删除 `internal/mt4client` 与 `internal/mt5client`，adapter 直接调用 mtapi gRPC**。

具体：
- `internal/mdgateway/adapter/mt4/gateway.go` 直接 `import pb "anttrader/gen/proto/mt4"` (mtapi proto)
- 同 `internal/mdgateway/adapter/mt5/gateway.go`
- mtapi gRPC 连接生命周期由 adapter 内部管理（`grpc.ClientConn`）
- 连接池逻辑由 `mdgateway/manager.go` 统一处理（每账户一个 connection）

**禁止**：
- 重新引入 `mt4client/mt5client` 包
- 在 adapter 之外的任何包 import mtapi proto

## 3. 备选方案

| 方案 | 优点 | 缺点 | 否决理由 |
|---|---|---|---|
| **A · adapter 直连**（采纳）| LOC 最小、心智单一 | 重写时要重踩 mtapi 暗坑 | 用 quirks register 防回归 |
| B · 保留 mt[45]client + adapter 包装 | 现有代码不动 | 永久双层包装 | 长期债 |
| C · 改写 mt[45]client 为薄壳 | 兼容现有引用 | 仍是中间层 | 抽象层次不对 |

## 4. 后果

### 正面
- MT 接入 LOC 从 ~3049 → ≤ 250（adapter）+ ≤ 800（mdgateway）= ≤ 1050
- 心智模型单一（业务 → mthub.OrderExecutor → adapter → mtapi）
- 暗坑修复有单一权威位置（adapter + quirks register）

### 负面
- 需要在 quirks register 中详尽记录 mtapi 暗坑（参考 alfq 已修的 4-5 条 + 后续发现）
- 若 mtapi 出 v2 协议，adapter 需要对应升级（但单点改）

### 中性
- mt4client/mt5client 老包在 M7-rewrite 期间保留（业务代码 import 切换需逐文件）
- M9 milestone 删除两个老包；CI 检测引用清零

## 5. 实施约束

### 5.1 adapter 唯一接触 mtapi

```bash
# CI 校验：mtapi proto 只在 adapter 中 import
ALLOWED='backend/internal/mdgateway/adapter/(mt4|mt5)'
git grep -l 'anttrader/gen/proto/mt[45]' backend/ \
  | grep -vE "$ALLOWED" \
  && { echo "FAIL: mtapi proto leaked outside adapter"; exit 1; } || true
```

### 5.2 业务代码不许 import mt4client/mt5client

```bash
# CI 校验
! grep -rE 'anttrader/internal/(mt4|mt5)client' \
    backend/internal/{ai,marketplace,oms,risk,connect,service,quantengine,factorsvc,mthub}/ \
  || { echo "FAIL: legacy mt[45]client imported"; exit 1; }
```

### 5.3 quirks register 强关联

每条已知暗坑必须：
1. 在 `docs/spec/16-mtapi-quirks-register.md` 有 `## Q-NNN` 条目
2. 在 adapter 代码中用 `// QUIRK Q-NNN: ...` 注释
3. 有对应的回归测试

## 6. 验证方式

### M7 完成时

```bash
# (1) adapter 行数
LOC=$(find backend/internal/mdgateway/adapter -name "*.go" -not -name "*_test.go" \
       | xargs wc -l | tail -1 | awk '{print $1}')
test "$LOC" -le 250

# (2) 无重复 mtapi import
ALLOWED='backend/internal/mdgateway/adapter/(mt4|mt5)'
test -z "$(git grep -l 'anttrader/gen/proto/mt[45]' backend/ | grep -vE "$ALLOWED")"

# (3) quirks 至少 5 条
grep -cE '^## Q-[0-9]+' docs/spec/16-mtapi-quirks-register.md | awk '$1<5{exit 1}'
```

### M9 完成时

```bash
# 老包删除
test ! -d backend/internal/mt4client
test ! -d backend/internal/mt5client

# 0 处 import
! grep -rE 'mt4client|mt5client' backend/internal/
```
