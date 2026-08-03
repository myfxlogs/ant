---
name: marketplace-search
description: >
  AlphaForge 市场模糊搜索实现模式。涵盖 PostgreSQL pg_trgm 相似度搜索、
  后端参数化查询构建（计数器模式）、前端 Ant Design Input 双边框修复、
  i18n 搜索提示词。当需要实现或修改市场搜索、模糊匹配、搜索框 UI 时使用此 skill。
---

# Marketplace 模糊搜索实现模式

> **最后更新**：2026-06-22，基于 marketplace 搜索功能完整实现。

## 架构概览

```
用户输入 → React Input (allowClear) → ConnectRPC → Go handler → Service
  → pg_trgm similarity() 跨 6 字段 → GIN trigram 索引 → 按相似度排序返回
```

## 参考文件

| 层级 | 文件 | 关键内容 |
|------|------|----------|
| 前端 | `frontend/src/pages/marketplace/components/MarketTab.tsx` | 搜索框 UI，allowClear + CSS 修复 |
| 前端 | `frontend/src/index.css` | `.marketplace-search-input .ant-input` 双边框修复 |
| 后端 | `backend/internal/marketplace/publish.go` | `buildPublishedQuery` 查询构建 |
| 后端 | `backend/internal/connect/marketplace/marketplace_handler.go` | Handler 透传 keyword |
| DB | `backend/migrations/172_pg_trgm_search.up.sql` | pg_trgm 扩展 + GIN 索引 |

---

## 1. 数据库层 — pg_trgm 模糊搜索

### 启用扩展

```sql
CREATE EXTENSION IF NOT EXISTS pg_trgm;
```

### GIN 索引（6 个搜索字段，3 张表）

```sql
-- marketplace_strategies
CREATE INDEX idx_marketplace_title_trgm ON marketplace_strategies USING gin (title gin_trgm_ops);
CREATE INDEX idx_marketplace_desc_trgm  ON marketplace_strategies USING gin (description gin_trgm_ops);

-- strategy_templates（策略原名）
CREATE INDEX idx_templates_name_trgm   ON strategy_templates USING gin (name gin_trgm_ops);

-- users（发布者）
CREATE INDEX idx_users_nickname_trgm   ON users USING gin (nickname gin_trgm_ops);
CREATE INDEX idx_users_email_trgm      ON users USING gin (email gin_trgm_ops);
```

`tags` 是 `TEXT[]` 列，不需要独立 GIN trigram 索引——`similarity(tags::text, kw)` 走顺序扫描但数组通常较短。

---

## 2. 后端 — 参数化查询构建

### 计数器模式（替代 `WHERE 1=1` + `len(args)`）

```go
args := []interface{}{}
p := 0
next := func() int { p++; return p }

if keyword != "" {
    n := next()  // ← 所有字段共用同一个 $N
    query += fmt.Sprintf(
        " AND (similarity(ms.title, $%[1]d) > 0.2"+
            " OR similarity(ms.description, $%[1]d) > 0.2"+
            " OR similarity(ms.tags::text, $%[1]d) > 0.2"+
            " OR similarity(st.name, $%[1]d) > 0.2"+
            " OR similarity(u.nickname, $%[1]d) > 0.2"+
            " OR similarity(u.email, $%[1]d) > 0.2)", n)
    args = append(args, keyword)
}
```

**核心要点**：
- 阈值 `0.2`：过滤噪声，容忍拼写错误和部分匹配
- 所有字段复用同一个参数 `$N`，只 `append` 一次 `keyword`
- `ORDER BY GREATEST(similarity(...), ...)` 按最高相似度排序
- 关键词搜索**跳过缓存**（变化太频繁，不适合缓存）

### 带搜索的完整 ORDER BY

```go
if hasFuzzySearch {
    n := next()
    query += fmt.Sprintf(
        " ORDER BY GREATEST(similarity(ms.title, $%[1]d), similarity(ms.description, $%[1]d),"+
            " similarity(st.name, $%[1]d), similarity(u.nickname, $%[1]d)) DESC LIMIT $%d", n, next())
    args = append(args, keyword, limit)
} else {
    // 无搜索时用标准排序
    query += fmt.Sprintf(" ORDER BY usp.published_at DESC LIMIT $%d", next())
    args = append(args, limit)
}
```

---

## 3. 前端 — 搜索框 UI

### 组件代码

```tsx
<Input
  className="marketplace-search-input"
  placeholder={t('marketplace.searchPlaceholder')}
  allowClear
  value={m.searchText}
  onChange={e => m.setSearchText(e.target.value)}
/>
```

### 双边框问题与修复

**根因**：Ant Design `allowClear` 的 DOM 结构：
```html
<span class="ant-input-affix-wrapper marketplace-search-input">
  <input class="ant-input">         <!-- 内层边框 ← 全局 CSS 给这里加了 !important -->
  <span class="ant-input-suffix">   <!-- 清除按钮 -->
</span>
```

全局 CSS（`index.css`）同时给 `.ant-input` 和 `.ant-input-affix-wrapper` 加了 `border: 1px solid ... !important`，导致双层边框。

**修复**（`index.css`）：
```css
/* Marketplace search — single border on wrapper, kill inner input border */
.marketplace-search-input .ant-input {
  border: none !important;
  box-shadow: none !important;
}
```

- `.marketplace-search-input .ant-input` 特异性高于 `.ant-input`
- 内层 input 边框强制为 `none`，外层 wrapper 保留全局样式
- 聚焦态由 wrapper 的 `ant-input-affix-wrapper-focused` 处理

### 数据流

```ts
// useMarketplace.ts — 搜索词实时传给 API
const resp = await marketplaceClient.listPublished({
  limit: pageSize,
  offset: (page - 1) * pageSize,
  keyword: searchText || undefined,  // ← 空字符串时传 undefined
  sortBy: sortBy || undefined,
});
```

- 搜索时不传 `userId`（市场展示全部策略）
- 搜索时跳过缓存（keyword 变化太频繁）
- 无需防抖：pg_trgm GIN 索引查询足够快，且无关键词时走缓存

---

## 4. 已搜索字段清单

| # | 字段 | 来源表 | 索引 | 排序权重 |
|---|------|--------|:--:|:--:|
| 1 | 标题 | `marketplace_strategies.title` | ✅ GIN | ✅ |
| 2 | 描述 | `marketplace_strategies.description` | ✅ GIN | ✅ |
| 3 | 标签 | `marketplace_strategies.tags` | — | ❌ |
| 4 | 策略原名 | `strategy_templates.name` | ✅ GIN | ✅ |
| 5 | 发布者昵称 | `users.nickname` | ✅ GIN | ✅ |
| 6 | 发布者邮箱 | `users.email` | ✅ GIN | ❌ |

---

## 5. 扩展搜索字段检查清单

新增搜索字段时：

1. ✅ 在 `buildPublishedQuery` 的 WHERE 子句中加 `OR similarity(new_col, $N) > 0.2`
2. ✅ 如需排序权重，在 `ORDER BY GREATEST(...)` 中加入
3. ✅ 在对应表上加 GIN trigram 索引
4. ✅ 如果字段在 JOIN 的表上（非 `marketplace_strategies`），确认 `buildPublishedQuery` 已 JOIN 该表
5. ✅ 测试查询计划：`EXPLAIN ANALYZE SELECT ...` 确认走 Index Scan
