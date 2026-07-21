> ⚠️ 已迁移至 docs/blocks/strategy-marketplace/plans/seo-strategy.md。此文件保留为兼容旧引用。

# SEO 策略 — 策略市场

> 关联：`docs/roadmaps/strategy-marketplace.md`
> 现状：`index.html` SEO 较好，但 SPA 页面（`Seo.tsx`）缺失 `keywords`/`og:type` 等标签，所有页面共用同一张 og-image。关键词仅覆盖 MT4/MT5。

## 当前问题

| 问题 | 严重度 |
|------|--------|
| 所有页面共用同一张 og-image，策略分享页没有定制化社交预览图 | 🔴 |
| 关键词仅覆盖 MT4/MT5/MQL——没有 broker 名、没有 AI 长尾词、没有转化词 | 🔴 |
| Sitemap 静态 5 个 URL，`/share/:token` 策略详情页未收录 | 🟡 |
| 策略详情页无 JSON-LD 结构化数据 | 🟡 |

## 关键词扩展

### Broker 名（30+）

IC Markets, Pepperstone, XM, Exness, FXCM, OANDA, Forex.com, EightCap, ThinkMarkets, Tickmill, Vantage, FP Markets, BlackBull, GO Markets, Fusion Markets, TMGM, Axi, RoboForex, OctaFX, FBS, HFM, Admiral Markets, AvaTrade, IG, CMC Markets, Saxo, Darwinex

### AI 长尾词

AI trading strategy generator, AI EA builder, AI forex strategy, MQL AI compiler, automated strategy creation, AI backtesting, strategy optimization AI, AI-powered trading platform, no-code trading strategy, AI trading signals, 自动交易策略生成, AI量化交易, 智能交易机器人

### 转化词

best MT4 backtesting platform, free forex strategy tester, buy trading strategies, forex EA marketplace, MQL5 Market alternative, verified trading strategies, 实盘交易策略购买, 外汇EA市场

## 实施模块

| 模块 | Phase | 内容 |
|------|-------|------|
| S1 | Phase 1 | `Seo.tsx` 补全 `keywords`/`ogImage`/`ogType`/`twitterCard` props |
| S2 | Phase 3 | 策略详情页动态 title/description/keywords + JSON-LD Product schema |
| S3 | Phase 1 | Landing page 嵌入 broker 名 + AI 关键词；新建 `/brokers` 页 |
| S4 | Phase 3 | 动态 Sitemap（含 `/share/:token`）；策略分享页 JSON-LD |
