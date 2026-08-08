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

---

## 文案 / Messaging（2026-08-08 战略复审后补，依据 `docs/roadmaps/market-strategy-review.md` §11）

> 定位 = 可信度护城河。话术三原则：**问题先行 / 具体不抽象（禁 jargon）/ 绝不承诺赚钱**。**双边市场 = 两条信任线，别混着说。**

### Landing 主入口（买家向，主流量）

**Hero：**
- **H1**：「别再买改得了的战绩。」
- **副标**：「每个策略都在我们服务器上真金白银地跑，战绩公开、加密锁定、卖家改不了。买之前你看到的，就是真实表现。」
- **宣言条（页脚/滚动区）**：「我们不承诺你赚钱——承诺这个的人都在骗你。我们只承诺：你看到的战绩是真的，改不了。」

**Meta description（≤155 字符）：**
- 中文：「别再买改得了的战绩。AlphaForge 策略真金白银实盘跑，战绩公开、卖家改不了、买前可查。MT4/MT5 EA 市场，MQL5 Market 之外的可靠选择。」
- EN：「Stop buying track records sellers can edit. Every AlphaForge strategy runs on our servers — real money, public, tamper-proof. The MQL5 Market alternative.」

**OG（社交分享预览）：**
- `og:title`：「别再买改得了的战绩 | AlphaForge」
- `og:description`：「策略真金白银实盘跑，战绩公开、卖家改不了。买之前就能看到真实表现。」

### 作者入口（/publish 或作者 CTA，独立信任线）

- **H1**：「上架策略，不怕被盗版，也不怕被平台偷。」
- **副标**：「① 代码不出平台——盗版不可能；② 我们不自营、只抽成——偷你=砸饭碗，没动机；③ 随时下架、代码归你。AI 还帮你对抗衰减。」
- **作者承诺一句话**：「我们不偷你的策略——不是看不到，是靠你赚钱，不靠偷你赚钱。」

### 关键词对齐（按新定位调整上方「关键词扩展」）

| 处置 | 关键词 | 理由 |
|---|---|---|
| ⬆️ 强化 | **MQL5 Market alternative** / forex EA marketplace / buy trading strategies | 直接对标 incumbent，转化意图强 |
| ⬆️ 新增（信任簇）| verified trading strategies / tamper-proof track record / real EA performance / 不能造假的 EA 战绩 / 真实实盘战绩 / 策略市场不刷回测 | 呼应"战绩改不了"定位 |
| ⬇️ 删/避 | AI trading signals / no-code trading strategy | 不做信号、不是 no-code——误导，拉高跳出率 |
| ⚠️ 保留搜索意图 | "verified trading strategies" | 人们搜这个词→捕获意图；落地文案用"改不了"承接 |

> **注意**：关键词 targeting（捕获搜索意图）≠ 页面文案（转化话术）。"verified" 作为搜索词要抢，但页面 H1 用"改不了"——两层不冲突。

### 本地化

中文为主稿；en/zh-tw/ja/vi **重译非直译**（按各市场"反诈话术"文化重写，保留"问题先行 + radical honesty"骨架）。

### 落地映射（→ 上方实施模块）

- S1 `Seo.tsx`：用本节 Meta/OG 文案填默认 props。
- S2 策略详情页：title/description 套"策略名 — 真金白银实盘，战绩改不了"模板。
- S3 Landing：Hero 用本节 H1/副标；broker 名块保留（SEO），但主视觉文案以"战绩改不了"为锚。
