---
name: ant-ui-patterns
description: >
  AlphaForge 前端页面开发规范和可复用模式。涵盖选择器组件（账号/品种/数据）、
  CRUD 页面骨架（Card表单+Table+删除）、ConnectRPC 客户端调用、
  i18n 多语言、React 核心约定。
  当需要开发或修改前端页面时使用此 skill。
  子文档：选择器(01)、CRUD页面(02)。
---

# AlphaForge 前端开发规范

> **最后验证**：2026-05-24，所有路径已对标当前前端代码库。

## 参考文档

| 编号 | 功能 | 文件 | 说明 |
|------|------|------|------|
| 01 | 选择器组件 | [01-selectors.md](references/01-selectors.md) | 账号选择器、品种选择器、品种列表获取、错误处理 |
| 02 | CRUD 页面 | [02-crud-pages.md](references/02-crud-pages.md) | 页面骨架、Card表单、Table列、ConnectRPC、i18n、useCallback |

## 目录约定

```
frontend/src/
  pages/strategy/StrategyTemplatePage.tsx   ← 完整页面示例（参考实现）
  pages/strategy/components/                ← 页面子组件（Card/Modal/Table 拆分）
  pages/accounts/BindAccount.tsx            ← 三步向导示例
  client/                                   ← ConnectRPC 聚合客户端（25 个模块）
  i18n/resources/zh-cn/                     ← 中文翻译（按功能分文件：strategy.ts, accounts.ts...）
  stores/                                   ← Zustand 状态管理
  hooks/                                    ← 自定义 hooks
  components/                               ← 共享 UI 组件
```

## 共用客户端

```ts
import { strategyTemplateApi } from '@/client/strategy';
import { accountApi } from '@/client/account';
import { marketApi } from '@/client/market';

// client/ 下每个模块提供 ConnectRPC 客户端
//   strategy.ts     → strategyTemplateApi, strategyScheduleApi
//   account.ts      → accountApi
//   market.ts       → marketApi.getSymbols(accountId) → SymbolInfo[]
//   analytics.ts    → analyticsApi
//   trading.ts      → tradingApi
```

## 核心 React 约定

1. **数据获取**：使用 `useCallback` 包裹 fetch 函数，`useEffect` 调用：
   ```ts
   const fetchTemplates = useCallback(async () => {
     setLoading(true);
     try {
       const r = await strategyTemplateApi.list({});
       setTemplates(r.templates || []);
     } catch { setTemplates([]); }
     setLoading(false);
   }, []);
   useEffect(() => { fetchTemplates(); }, [fetchTemplates]);
   ```

2. **错误处理**：try-catch 中 set 空数组/空对象，message.error 显示错误

3. **表单布局**：`layout="vertical"` + `<Space size="large" wrap align="start">`

4. **提交按钮对齐**：给按钮的 `Form.Item` 设 `label=" "`

5. **i18n**：使用 `useTranslation()` hook，翻译键按 `feature.key` 命名

## 参考代码

- 完整页面示例：`frontend/src/pages/strategy/StrategyTemplatePage.tsx`
- 三步向导示例：`frontend/src/pages/accounts/BindAccount.tsx` (498 lines)
- 选择器模式：`frontend/src/pages/strategy/StrategyTemplatePage.tsx` (account/symbol selectors)
