# 登录会话修复 spec（"记住我" + 切用户数据隔离）

> **功能块**：frontend（auth + Zustand stores）
> **来源**：用户报告 2 个 bug（2026-08-08）。Bug1 "记住我"被 DeepSeek 改 4 次未解决；Bug2 切用户账号数据残留、需手动刷新。
> **角色**：审计方出 spec（根因已定位）；施工方实现+回填，**不自行宣告 ✅，等审计方实测**。

---

## 1. 根因（审计方已逐行定位，2026-08-08）

### Bug 1：「记住我」是 100% 死按钮

**事实链（全有 file:line 锚点）：**

1. **传输层是 Bearer-token，不是 cookie**：`frontend/src/client/transport.ts:32` 从 authStore 取 accessToken，`:69/:85` 设 `Authorization: Bearer <token>`。**accessToken 才是凭证。**
2. **accessToken 永远写 localStorage**：`stores/authStore.ts:42` `storage: createJSONStorage(() => localStorage)` 硬编码；`:43-47` `partialize` **无条件**持久化 `{user, accessToken, _rememberMe}`。
3. **`_rememberMe` 是死标志**：存了（`:46`）但**从未被用来控制 storage**——勾不勾"记住我"，token 都进 localStorage → 关浏览器再开仍登录 → checkbox 无效。
4. **rememberMe 根本没传后端**：`hooks/useAuth.ts:16` `authApi.login(data.login, data.password)`——**丢弃了 rememberMe**。

**为什么 DeepSeek 改 4 次没解决**：最后一次 `ee42629c` 的 commit message 自称"rememberMe controls the backend refresh cookie TTL, not frontend storage"——**这是错误前提**。传输层不用 cookie（用 Bearer），后端也收不到 rememberMe（login 没传）。该修复建立在一个**不存在的机制**上，所以无效。git 史：`4f9298f8`→`cf4f00de`(hydration race)→`4b4564f7`(second refresh)→`ee42629c`(放弃条件持久化)——**没有一次让 `_rememberMe` 真正控制 token 持久化**。

### Bug 2：切用户账号数据残留（需手动刷新）

- `hooks/useAuth.ts:51-60` `logout()` 只调 `storeLogout()`。
- `stores/authStore.ts:37` `logout()` **只清 authStore 自己**（user/accessToken/isAuthenticated/_rememberMe）。
- **不清** tradingStore（持仓/账号数据 `tradingStore.ts:20 positions/positionsMap`）、notificationStore、workspaceStore、uiStore、chartIndicatorsStore。
- → A 登录→退出→B 登录：A 的 store 数据残留，UI 显示 A 的交易账号，**手动刷新（整页 reload 重新初始化）才显示 B 的**。
- **附带**：SSE 订阅（user-scoped 流）未在 logout 拆除。
- **测试盲区**：`authStore.test.ts` 的 logout 测只验 authStore **自清**，不验跨 store → 测试绿但 bug 在。

---

## 2. Part A —「记住我」修复（token-based，让 _rememberMe 真正生效）

**设计（Bearer-token 架构下唯一正确解）：** `_rememberMe` 控制 **accessToken 的存储位置/存活**。

| 场景 | 行为 |
|---|---|
| ✅ 勾"记住我" | accessToken 存 **localStorage** → 关浏览器再开仍登录 |
| ❌ 不勾 | accessToken 存 **sessionStorage**（或仅内存）→ 关 tab/浏览器即失效，刷新(tab 内)仍保持 |

**任务（施工方）：**

1. **`setTokens` 按 rememberMe 路由存储**：勾→localStorage；不勾→sessionStorage。`_rememberMe` 选择本身**始终存 localStorage**（这样 rehydrate 时知道该从哪读，避免"第二次刷新丢 token"回归——`4b4564f7` 的老 bug）。
2. **`partialize` 条件化**：恢复 ee42629c **之前**的条件持久化逻辑（`if (_rememberMe) persist accessToken`），但确保 `_rememberMe` 已持久化（当前已持久化，保留）。
3. **自定义 storage adapter 或手动 token 管理**：Zustand persist 单 storage 无法直接按字段分存储——施工方二选一：① 自定义 `StateStorage` 按 `_rememberMe`（从独立 localStorage flag 读）路由 localStorage/sessionStorage；② 不用 persist 管 token，`setTokens`/`logout` 手动写/清 localStorage|sessionStorage，bootstrap 时读回。
4. **hydrate 防 race**：渲染 auth-dependent UI 前等 `_hasHydrated`（已有，确认 App.tsx 守卫到位，`cf4f00de` 教训）。

**验收（审计方实测）：**
- 勾"记住我"登录 → 关浏览器全关 → 重开 → **仍登录**。
- 不勾登录 → 关 tab/浏览器 → 重开 → **要求重新登录**（token 不在）。
- 两种情况下，**刷新页面（tab 内）都不丢登录**（不回归 `4b4564f7` "second refresh"）。

**对抗证明：**
- 删掉条件化（恢复"always persist"）→ 不勾也跨重启保持 → 测试必红。
- 新增测试：`rememberMe=false` 时 localStorage 不含 accessToken / sessionStorage 含；`rememberMe=true` 反之。

## 3. Part B — 切用户数据隔离（logout 清所有 user-scoped store + 拆 SSE）

**任务（施工方）：**

1. **每个 user-scoped store 加 `reset()`**：至少 tradingStore、notificationStore、workspaceStore（+ uiStore/chartIndicatorsStore 视是否含用户态）。reset 回初始 state（清空 positions/positionsMap/accounts 等）。
2. **中央 `resetAllStores()`**：聚合上述 reset，在 `useAuth.logout()`（`useAuth.ts:51`）`storeLogout()` 之后调用。
3. **SSE 订阅拆除**：logout 时关闭所有 user-scoped stream（profit/order/bar/notification）。定位 `useNotificationListener`/`App.tsx` 中的 stream 订阅，logout 时 abort。
4. **防御层（推荐）**：`App.tsx` 加 effect——当 `user?.id` 变化（含 null）时 reset 所有 user-scoped store（防漏，不只依赖 logout 路径）。
5. **测试补全**：`authStore.test.ts` 加**跨 store** 用例——login A → logout → login B → 断言 tradingStore 为空（B 的数据未拉前不显示 A 的）。

**验收（审计方实测）：**
- A 登录（加载 A 的账号/持仓）→ 退出 → B 登录 → **无需手动刷新，UI 立即不显示 A 的任何数据**（空态或 B 的数据）。
- DevTools 查 store：logout 后 tradingStore 等归零。

**对抗证明：** 移除 `resetAllStores()` 调用 → A→B 切换后 tradingStore 仍含 A 数据 → 测试必红。

---

## 4. Non-goals
- 不改后端 auth（除非 Part A 选"传 rememberMe 给后端控 cookie"——本 spec 选 token 方案，后端不动）。
- 不引入 refresh token 机制（当前 refreshToken 恒为 ''，属另一议题，不在本 spec）。

## 5. REUSE 核对（施工方动工前 `bash scripts/cap.sh`）
- `_hasHydrated` hydrate 守卫（已有）、`useAuthStore.getState()`（transport 已用模式）、各 store 现有 `set`/初始 state（reset 复用）。
- 测试基建：`stores/__tests__/` 现有模式（mockClient.ts）。

## 6. 完工回填纪律（施工方，不做=任务失败）
1. `tech-debt-registry.md` 新增 `AUTH-REMEMBER`（Bug1）+ `AUTH-SWITCH-LEAK`（Bug2）条目 🟦→✅ + 真实根因/修复/对抗证明/测试结果。若真根因与 spec 不同→如实写明。
2. `handover-audit-plan.md` 变更日志加一行。
3. **不自行宣告完成**——等审计方核对状态 + 实测（浏览器实测两个验收场景）。

---

> **审计方备注**：两 bug 同属 auth/session 生命周期但独立可 PR。**Part A 关键警告**：传输是 Bearer-token 非 cookie——别再犯 DeepSeek 的"cookie 控 session"错误前提；让 `_rememberMe` 控制 token 存储。**Part A 易踩 `4b4564f7` "second refresh" 回归**——`_rememberMe` 选择必须跨刷新稳定。
