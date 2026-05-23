# 17 · 密钥管理 + 错误体系规范

> 路径：`backend/internal/secrets/` + `backend/internal/errs/`
> 适用：所有需要加密、解密、错误返回、跨进程传播错误码的代码
> 关联 ADR：0001、0005；关联 spec：10、11、13、14

## 1. 密钥管理（secrets / vault.Client）

### 1.1 设计目标

- 业务存的是 **密文**（PG `BYTEA` 列）
- 业务运行时 **按需解密**，明文绝不落盘、绝不入日志、绝不入 metric label
- 支持 **密钥版本化**（KEK rotation）和 **零停机轮换**
- v2 单机不上 HashiCorp Vault 服务；但接口形态保留可平滑切换

### 1.2 算法选型（行业标杆）

| 维度 | 选择 | 理由 |
|---|---|---|
| 对称算法 | **AES-256-GCM** | NIST 推荐；自带 AEAD（密文 + 完整性）；FIPS 140-2 兼容 |
| 密钥派生 | **HKDF-SHA256** from `ANT_MASTER_KEY` | 主密钥单源；不同用途派生子密钥（password vs token）|
| nonce | 每次加密 `crypto/rand` 12 字节 | GCM 标准 |
| 密文格式 | `version(1B) || nonce(12B) || ciphertext || tag(16B)` | 自包含，便于轮换 |
| 主密钥来源 | env `ANT_MASTER_KEY`（base64 32B）| K8s Secret / docker-compose `.env`；不入仓 |
| 轮换 | 增加 `version`，新密钥派生；后台 worker 重加密旧版本数据 | 双写期 ≤ 7 天 |

**禁止**：
- ❌ 明文密码 / token 入日志（即使 debug 级）
- ❌ 明文落盘（包括 spill jsonl）
- ❌ 明文入 Prometheus label 或 trace span
- ❌ 用 `crypto/des` `crypto/rc4` `md5` `sha1` 任何弱算法

### 1.3 接口契约（`backend/internal/secrets/vault.go`）

```go
// Package secrets provides AES-256-GCM envelope encryption with
// versioned KEK derivation. v2 implementation runs in-process
// (no external Vault service); interface is shaped for future
// migration to HashiCorp Vault transit or AWS KMS.
package secrets

import "context"

// Client 是 ant 唯一的加解密接口。所有需要密钥的模块通过此接口。
type Client interface {
    // Encrypt 用当前 latest 版本的 KEK 加密 plaintext，返回自包含密文
    // （版本号 + nonce + 密文 + tag）。
    // purpose 是用途子密钥（HKDF info），常量见下。
    Encrypt(ctx context.Context, purpose Purpose, plaintext []byte) (ciphertext []byte, err error)

    // Decrypt 自动从 ciphertext 头部提取版本号，用对应 KEK 解密。
    // 版本未知 → ErrUnknownKeyVersion。
    Decrypt(ctx context.Context, purpose Purpose, ciphertext []byte) (plaintext []byte, err error)

    // Reencrypt 解密后用 latest 版本重新加密。轮换 worker 用。
    Reencrypt(ctx context.Context, purpose Purpose, ciphertext []byte) ([]byte, error)

    // CurrentVersion 返回当前 latest 版本号（>= 1）。
    CurrentVersion() uint8
}

// Purpose 区分子密钥用途（HKDF info）。新增 purpose 必须 ADR。
type Purpose string

const (
    PurposeMTPassword   Purpose = "mt-password"      // mt_accounts.password_encrypted
    PurposeMTAPIToken   Purpose = "mtapi-token"      // mt_accounts.mtapi_token_encrypted
    PurposeBrokerCookie Purpose = "broker-cookie"    // 预留：第三方登录态
)
```

### 1.4 实现要点

**`backend/internal/secrets/aes_gcm.go`**：
- `New(masterB64 string, currentVersion uint8) (Client, error)`
- 启动时 base64 解码 `ANT_MASTER_KEY` → 32 字节 KEK0
- 派生 `KEK_v_purpose = HKDF(KEK0, info=fmt.Sprintf("ant/v%d/%s", v, purpose))`
- LRU cache 派生密钥（最多 16 条）
- Encrypt：取 `currentVersion` → 派生 KEK → AES-GCM seal → 拼接版本号
- Decrypt：读首字节版本 → 派生 KEK → AES-GCM open

**`backend/internal/secrets/rotation.go`**（M8 实现，M7 预留接口）：
- `Rotator.Run(ctx)` 后台 worker
- 扫描 PG `mt_accounts` 中 `version < currentVersion` 的行
- 批量 Reencrypt → UPDATE
- metric `secret_reencrypt_total{purpose, status}`

### 1.5 测试要求

```go
func TestVault_RoundTrip(t *testing.T)              // 加密→解密 == 原文
func TestVault_DecryptWithDifferentPurpose_Fails(t) // 防 purpose 串用
func TestVault_DecryptCorruptedTag_Fails(t)         // GCM 完整性
func TestVault_OldVersionDecrypt(t)                 // v1 密文用 v2 KEK 解密
func TestVault_NonceUnique(t)                       // 1000 次加密 nonce 全不同
```

### 1.6 验收命令

```bash
( cd backend && go test -race -cover ./internal/secrets/... )
# 覆盖率 ≥ 90%
go test -cover ./internal/secrets/... | grep -oE 'coverage:.*%' | awk '{gsub("%",""); if ($2<90) exit 1}'

# CI 静态扫：明文不漏到 log
! grep -rE 'log\.(Info|Debug|Warn|Error)\([^)]*(password|mtapi_token|secret)' backend/internal/
```

---

## 2. 错误体系（errs）

### 2.1 现状（保留）

`backend/internal/errs/` 已有：
- `errs.go`：定义 `Code` 类型 + 数百枚举（auth 1xxx / account 2xxx / order 3xxx / market 4xxx / report 5xxx / admin 6xxx / broker 7xxx / strategy 8xxx / system 9xxx）
- `messages.go`：`Code → 中文 user_message` 映射
- `errs_test.go`：单测

**v2 不重写**，仅 **补充约束 + 接入 ConnectRPC**。

### 2.2 类型契约

```go
// 已有
type Code int

// v2 新增 / 强约束
type Error struct {
    Code    Code   // 必填，业务错误码
    Msg     string // 必填，中文用户可见消息
    Cause   error  // 可选，原始错误（不暴露给前端）
    TraceID string // 自动注入
}

func (e *Error) Error() string                       // 已有
func (e *Error) Unwrap() error                       // 必加，配合 errors.Is/As
func (e *Error) ToConnectError() *connect.Error      // 必加（spec 14 用）

// 新增：分类映射到 connect.Code
func (c Code) ConnectCode() connect.Code {
    switch {
    case c == CodeOK:                          return 0
    case c == CodeInvalidParam:                return connect.CodeInvalidArgument
    case c == CodeUnauthorized, c == CodeTokenExpired, c == CodeTokenInvalid:
                                               return connect.CodeUnauthenticated
    case c == CodeForbidden, c == CodeUserDisabled:
                                               return connect.CodePermissionDenied
    case c == CodeNotFound:                    return connect.CodeNotFound
    case c == CodeRateLimited:                 return connect.CodeResourceExhausted
    case c == CodeTimeout:                     return connect.CodeDeadlineExceeded
    case c == CodeServiceUnavail:              return connect.CodeUnavailable
    case c >= 7000 && c < 8000:                return connect.CodeUnavailable  // broker
    case c == CodeInternal, c == CodeUnknown:  return connect.CodeInternal
    default:                                   return connect.CodeInternal
    }
}
```

### 2.3 强约束（违反 = lint 失败）

| 规则 | CI 检查 |
|---|---|
| 业务代码 0 处裸字符串 error | `! grep -rE 'errors\.New\("[\u4e00-\u9fa5]' backend/internal/{ai,marketplace,oms,risk,connect,quantengine,factorsvc,mthub,mdgateway}/` |
| 所有 connect handler 必须 return `errs.ToConnectError()` | golangci 自定义 lint：handler 函数返回 error 时 type 必须是 `*errs.Error` |
| messages.go 必须覆盖所有 Code | 单测 `TestAllCodesHaveMessage` |
| Code 数值不可改、不可删（向后兼容）| CI：`git diff backend/internal/errs/errs.go` 不许出现 `-\s*Code[A-Z][a-zA-Z]+\s+=` |

### 2.4 ConnectRPC handler 模式（spec/14 接入）

```go
func (s *Server) PlaceOrder(ctx context.Context, req *connect.Request[v1.PlaceOrderRequest]) (*connect.Response[v1.PlaceOrderResponse], error) {
    if err := validate(req.Msg); err != nil {
        return nil, errs.New(errs.CodeInvalidParam, "请求参数无效", err).ToConnectError()
    }
    out, err := s.svc.Place(ctx, req.Msg)
    if err != nil {
        // err 已经是 *errs.Error；包装一下 trace 后转 connect.Error
        return nil, errs.Wrap(err).ToConnectError()
    }
    return connect.NewResponse(out), nil
}
```

### 2.5 验收命令

```bash
# 全部 Code 都有 message
( cd backend && go test -run TestAllCodesHaveMessage ./internal/errs/... )

# 业务代码无裸字符串中文 error
! grep -rE 'errors\.New\("[^"]*[\u4e00-\u9fa5]' \
    backend/internal/{ai,marketplace,oms,risk,connect,quantengine,factorsvc,mthub,mdgateway}/ \
  || { echo "FAIL: bare Chinese error string found"; exit 1; }

# handler 返回 error 必须 *errs.Error（用 staticcheck 自定义规则或 review）
```

---

## 3. 跨模块约束

| 模块 | 用 secrets | 用 errs |
|---|---|---|
| mdgateway/runner | 解密 mt_accounts.password_encrypted/mtapi_token_encrypted | 加载失败 → CodeInternal + 中文 |
| mthub | — | broker 拒单 → 7xxx + 中文 |
| connect/* | — | 全部 handler |
| repository/* | — | sqlc 错误包装 → CodeInternal |
| factorsvc / quantengine | — | DSL 解析失败 → 8xxx |

## 4. 实施卡片（提级到 ROADMAP）

- M7.0-9（新增）：实现 `internal/secrets/`（vault.Client + AES-GCM 实现 + 测试 ≥ 90% cover）
- M7.1-2 前置：等待 M7.0-9 完成，否则 ETL 没工具
- M7.2-5 实施时：所有 9 个 mthub handler 用 `errs.ToConnectError()`
