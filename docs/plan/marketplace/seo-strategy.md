# SEO 策略 — 策略市场

> 关联：`docs/roadmaps/strategy-marketplace.md`
> 现状：`index.html` 的 SEO 较好，但 SPA 页面（`Seo.tsx`）缺失大量标签，关键词仅覆盖 MT4/MT5。

---

## 当前问题

> **纠正**：`Seo.tsx` 不覆盖的标签会从 `index.html` fallback（helmet 的默认行为）。所以 `og:image`、`twitter:image` 实际上**存在**，只是所有页面共用同一张通用 og-image。真正的问题是策略详情页应该有**定制化**的社交预览图（含策略指标）。

| 问题 | 严重度 |
|------|--------|
| 所有页面共用同一张 og-image，策略分享页没有定制化的社交预览图（含策略名+收益率等指标） | 🔴 |
| 关键词仅覆盖 MT4/MT5/MQL——没有 broker 名、没有 AI 长尾词、没有转化词 | 🔴 |
| Sitemap 静态 5 个 URL，`/share/:token` 策略详情页未收录 | 🟡 |
| `public/` 下 prerendered HTML 全部重定向到 `/`，浪费抓取预算 | 🟡 |
| 策略详情页无 JSON-LD Product/SoftwareApplication 结构化数据 | 🟡 |
| 5 语言 i18n 但 `Seo.tsx` 没有 hreflang（`index.html` 有 `og:locale:alternate` 但不是真正的 hreflang link） | 🟢 |

---

## 关键词扩展

当前只覆盖 MT4/MT5。需要加入三类关键词：

### Broker 名（30+）

```
IC Markets, Pepperstone, XM, Exness, FXCM, OANDA, Forex.com,
EightCap, ThinkMarkets, Tickmill, Vantage, FP Markets, BlackBull,
GO Markets, Blueberry Markets, Fusion Markets, TMGM, Axi,
RoboForex, OctaFX, FBS, JustMarkets, HFM, Moneta Markets,
Admiral Markets, AvaTrade, IG, CMC Markets, Saxo, Darwinex, Swissquote
```

**用法**：不是 spam 列表。在以下场景自然出现——
- 策略详情页："此策略支持以下 broker"（MT 兼容的所有 broker）
- Landing page："兼容 30+ MT4/MT5 broker"
- Marketplace 搜索描述："为 {broker} 用户发现策略"

### AI 长尾词（20+）

```
AI trading strategy generator, AI EA builder, AI forex strategy,
MQL AI compiler, automated strategy creation, AI backtesting,
strategy optimization AI, machine learning trading bot,
AI-powered trading platform, no-code trading strategy,
AI trading signals, smart trading bot, algorithmic trading AI,
自动交易策略生成, AI量化交易, 智能交易机器人, AI外汇EA,
MetaTrader AI tool, expert advisor AI generator
```

### 转化词（10+）

```
best MT4 backtesting platform, free forex strategy tester,
buy trading strategies, forex EA marketplace,
MQL5 Market alternative, MetaTrader strategy store,
professional EA for sale, verified trading strategies,
实盘交易策略购买, 外汇EA市场
```

---

## 实施任务

### 模块 S1 · `Seo.tsx` 补全 🔴

**文件**：`frontend/src/components/common/Seo.tsx`

- [ ] **S1a 补充缺失标签**：`og:image`、`og:image:width`、`og:image:height`、`og:image:alt`、`og:type`、`og:site_name`、`og:locale`、`twitter:card`、`twitter:image`、`keywords`
- [ ] **S1b 默认值**：无 page 特定值时 fallback 到 `index.html` 的全局值
- [ ] **S1c 添加 `keywords` prop**：各页面传入自己的关键词数组

### 模块 S2 · 策略详情页 SEO 🔴

**文件**：`frontend/src/pages/marketplace/components/StrategyDetailModal.tsx`
**文件**：`frontend/src/pages/share/SharePerformancePage.tsx`

- [ ] **S2a 动态 title**：`"{策略名} — {品种} {周期} {策略类型} | AlphaForge Marketplace"`
- [ ] **S2b 动态 description**：`"收益{total_return}，夏普{sharpe}，最大回撤{max_dd}。{提供者}发布在AlphaForge的{品种}策略。支持IC Markets、Pepperstone等30+ MT4/MT5 broker。"`
- [ ] **S2c 动态 keywords**：策略名 + 品种 + 周期 + 策略类型 + broker 名列表 + AI 相关
- [ ] **S2d 动态 og:image**：服务端生成策略战绩卡片（复用 `og-image.svg` 模式，叠加策略指标）

### 模块 S3 · 关键词落地页 🟡

**文件**：`frontend/src/pages/landing/` 相关组件

- [ ] **S3a Landing page 关键词扩展**：在现有基础上增加 broker 名（"兼容 30+ broker"）+ AI 词（"AI 策略生成"）
- [ ] **S3b Marketplace 页描述扩展**：`"发现并购买 MT4/MT5 交易策略。支持 IC Markets, Pepperstone, XM 等 30+ broker。AI 辅助策略生成与优化。"`
- [ ] **S3c 新增 Broker 兼容性页面**（可选）：`/brokers` 页列出所有兼容 broker，每 broker 一个卡片（含名称、logo、支持平台）——这是 SEO 金矿

### 模块 S4 · Sitemap + 结构化数据 🟡

- [ ] **S4a 动态 Sitemap**：后端定时生成 `/sitemap.xml`，包含所有公开策略的 `/share/:token` URL
- [ ] **S4b JSON-LD**：策略分享页加入 `Product` 或 `SoftwareApplication` schema（名称、提供者、评分、价格、回测指标）
- [ ] **S4c 清理 prerendered HTML**：删除 `public/landing.html` 等重定向文件（或改为真正的 prerender 快照）

### 模块 S5 · hreflang 🟢

- [ ] **S5a `Seo.tsx` 加上 hreflang**：`<link rel="alternate" hreflang="zh-cn" href="...">`、`ja`、`vi`
- [ ] **S5b `index.html` 同步**：已有 `og:locale:alternate`，扩展为完整的 hreflang set

---

## 优先级

```
Phase 1 同步做:
  S1 (Seo.tsx 补全) + S3 (关键词落地页)  ← 改动小，SEO 效果大

Phase 2 同步做:
  S2 (策略详情页 SEO) + S4 (sitemap + JSON-LD)

Phase 3 同步做:
  S5 (hreflang)
```
