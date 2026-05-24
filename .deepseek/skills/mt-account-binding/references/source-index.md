# MT 账户绑定 — 源码参考索引

> 基于 `/opt/antclaw/emulator/anttrader` 的完整文件路径映射。

## 前端 (React/TypeScript)

| 文件 | 行数 | 关键内容 |
|---|---|---|
| `frontend/src/types/account.ts` | ~65 | `Account` (28 字段)、`BindAccountRequest` (7 字段) |
| `frontend/src/client/account.ts` | ~100 | `accountApi.create()` / `searchBroker()` / `connect()` / `verifyTradePermission()` / `updateTradingPassword()` |
| `frontend/src/hooks/useAccount.ts` | ~160 | `createAccount()` / `bindAccount()` / `connectAccount()` / `enableAccount()` / `disableAccount()` |
| `frontend/src/stores/accountStore.ts` | ~70 | Zustand store: `addAccount()` / `updateAccount()` / `removeAccount()` / `updateAccountStatus()` / `setEnablingAccount()` |
| `frontend/src/pages/accounts/BindAccount.tsx` | ~500 | 三步向导 UI：Step1(选择经纪商) → Step2(输入凭据) → Step3(确认绑定) |
| `frontend/src/pages/accounts/components/AddAccountCard.tsx` | ~40 | 金色 "+" 卡片入口按钮，hover 效果 |
| `frontend/src/pages/accounts/components/EditAccountModal.tsx` | — | 编辑账户弹窗 |
| `frontend/src/pages/accounts/components/DisabledAccountsSection.tsx` | — | 已禁用账户折叠区域 |

## 后端 (Go)

| 文件 | 关键内容 |
|---|---|
| `proto/account.proto` | `AccountService` 定义（11 个 RPC） |
| `proto/account_crud.proto` | `CreateAccountRequest` / `UpdateAccountRequest` / `DeleteAccountRequest` |
| `proto/account_entity.proto` | `Account` message（28 字段） |
| `proto/account_connection.proto` | `ConnectAccountRequest` / `ConnectAccountResponse` |
| `proto/account_permission.proto` | `VerifyTradePermission` / `UpdateTradingPassword` |
| `backend/internal/connect/account_service.go` | Connect-RPC handler: `CreateAccount()` L96-143, `UpdateAccount()` L145+, `ConnectAccount()`, `VerifyTradePermission()` |
| `backend/internal/service/account_service.go` | 业务层: `BindAccount()` L88-200（含 MT4/MT5 分支连接） |
| `backend/internal/model/models.go` | `MTAccount` 结构体定义 |
| `backend/internal/repository/account_repository.go` | `Create()` / `GetByID()` / `GetByUserID()` / `GetByLoginAndHost()` / `Update()` / `UpdateDisabled()` / `Delete()` / `CountByUserID()` |
| `backend/migrations/001_init.up.sql` | `mt_accounts` 表结构：`password VARCHAR(255) NOT NULL`（明文列） |

## MT 网关 (gRPC Client)

| 文件 | 关键内容 |
|---|---|
| `backend/mt4/mt4_grpc.pb.go` | MT4 gRPC 客户端：`Connect()` / `AccountSummary()` / `Disconnect()` |
| `backend/mt5/mt5_grpc.pb.go` | MT5 gRPC 客户端：同上 |
| `backend/internal/mt4client/` | MT4 客户端封装 |
| `backend/internal/mt5client/` | MT5 客户端封装 |
| `backend/internal/connection/` | `ConnectionManager`：连接池、自动重连、健康检查 |

## 关键数据流

```
BindAccount.tsx:handleBind()
  → request = { mtType, brokerCompany, brokerServer, brokerHost, login, password }
  → useAccount().bindAccount(request)
    → useAccount().createAccount(request)
      → accountApi.create(request)
        → accountClient.createAccount(CreateAccountRequest)  [gRPC]
          → backend: CreateAccount()
            → JWT 鉴权 (interceptor.GetUserID)
            → model.MTAccount{...AccountStatus:"connecting"}
            → accountRepo.Create()           // INSERT INTO mt_accounts
            → connManager.Connect()          // MT gRPC: Connect(login, password, host, port)
              → MT4/MT5: AccountSummary()    // 获取 balance/equity/...
              → accountRepo.Update()         // 更新账户摘要字段
            → connManager.Disconnect()       // 断开测试连接
            → convertMTAccount(account)      // 转为 Proto 响应
      ← addAccount(account)                  // 写入 Zustand store
      ← showSuccess('创建成功')
  ← navigate('/')                            // 跳转首页
```

## 密码安全说明（重要）

**纠正**：MT 交易密码以**明文**存储，不做加密。

- `mt_accounts.password` 列类型 `VARCHAR(255)`，存原始密码
- 连接 MT 服务器时必须使用原始密码，哈希/加密无意义
- 用户登录密码 (`users.password_hash`) 使用 bcrypt 哈希，两者不同
- 安全依赖：HTTPS 传输加密 + JWT 鉴权 + DB 访问控制 + 日志脱敏
