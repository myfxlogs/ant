# frontend — 前端界面

> React + Ant Design + ConnectRPC + SSE。策略工作区、回测面板、Agent 聊天、管理后台、交易界面。

## 代码位置

```
frontend/src/
  pages/              ← 17 页面目录
  components/         ← 共享组件（Seo、layout、选择器…）
  hooks/              ← 自定义 hooks
  gen/                ← proto 生成的 TS 类型
```

## 页面地图

| 页面 | 路由 |
|------|------|
| LandingPage | `/` |
| BrokersPage | `/brokers` |
| MarketplacePage | `/marketplace` |
| StrategyWorkspacePage | `/strategy` |
| LiveStrategyPage | `/strategy/live` |
| Dashboard | `/dashboard` |
| AccountDetail/Report | `/accounts/:id` |
| Admin（20+ 子页） | `/admin/*` |
| Trading | `/trading` |
| Wallet | `/wallet` |
| Profile | `/profile` |
| Subscription | `/subscription` |
| SharePerformancePage | `/share/:token` |
| Login/Register/ForgotPassword | `/login` `/register` |

## 关键设计

- i18n：5 语言（en/zh-cn/zh-tw/ja/vi）
- SSE：ConnectRPC server-stream + PG NOTIFY 实时推送
- 暗色主题
- `Seo.tsx`：react-helmet-async 动态 meta 标签
## 关联文档
- [spec/workspace-redesign.md](spec/workspace-redesign.md)
- [spec/strategy-workspace-first-principles-audit.md](spec/strategy-workspace-first-principles-audit.md) — 第一性原理审计 + 3 批改进方案
