---
name: ant-ui-patterns-02-crud-pages
description: >
  前端 CRUD 页面骨架模式。涵盖 Card+Form 创建、Table 列定义、
  Tag 状态渲染、Popconfirm 删除、ConnectRPC 客户端模式、
  i18n 翻译键约定。当需要新建/改造页面时使用。
---

# 02 — CRUD 页面

> **最后验证**：2026-05-24，代码模式提取自 `StrategyTemplatePage.tsx`。

## 标准页面骨架

```tsx
import React, { useState, useEffect, useCallback } from 'react';
import { Table, Tag, Typography, Button, Card, Form, message, Popconfirm } from 'antd';
import { ReloadOutlined, PlusOutlined, DeleteOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { strategyTemplateApi } from '@/client/strategy';

const { Text } = Typography;

const MyPage: React.FC = () => {
  const { t } = useTranslation();

  // ── Data ──
  const [data, setData] = useState<any[]>([]);
  const [loading, setLoading] = useState(false);

  // ── Fetch ──
  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const r = await strategyTemplateApi.list({});
      setData(r.templates || []);
    } catch { setData([]); }
    setLoading(false);
  }, []);
  useEffect(() => { fetchData(); }, [fetchData]);

  // ── Delete ──
  const handleDelete = async (id: string) => {
    try {
      await strategyTemplateApi.delete({ strategyId: id });
      message.success(t('strategy.deleteSuccess', '删除成功'));
      fetchData();
    } catch (e: any) {
      message.error(e?.message || t('strategy.deleteFailed', '删除失败'));
    }
  };

  return (
    <div>
      <Card size="small" style={{ marginBottom: 12 }}>
        {/* Create form (see below) */}
      </Card>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 12 }}>
        <Text strong>{t('strategy.title')}</Text>
        <Button icon={<ReloadOutlined />} onClick={fetchData} loading={loading}>
          {t('common.refresh', '刷新')}
        </Button>
      </div>
      <Table size="small" dataSource={data} rowKey="id"
        loading={loading} columns={columns} />
    </div>
  );
};
export default MyPage;
```

## 创建表单 (Card + Form + Space)

```tsx
const [form] = Form.useForm();
const [creating, setCreating] = useState(false);

const handleCreate = async (values: any) => {
  setCreating(true);
  try {
    await strategyTemplateApi.create(values);
    message.success(t('strategy.createSuccess'));
    form.resetFields();
    fetchData();
  } catch (e: any) {
    message.error(e?.message || t('strategy.createFailed'));
  }
  setCreating(false);
};

// JSX:
<Card size="small" style={{ marginBottom: 12 }}>
  <Form form={form} layout="inline" onFinish={handleCreate}>
    <Form.Item name="name" rules={[{ required: true }]}>
      <Input placeholder={t('strategy.name')} />
    </Form.Item>
    <Form.Item>
      <Button type="primary" htmlType="submit"
        icon={<PlusOutlined />} loading={creating}>
        {t('strategy.create')}
      </Button>
    </Form.Item>
  </Form>
</Card>
```

## Table 列定义

```tsx
const columns = [
  {
    title: 'ID', dataIndex: 'id', width: 100,
    render: (v: string) => <Text code copyable>{v?.slice(0, 8)}</Text>,
  },
  {
    title: t('strategy.name'), dataIndex: 'name',
  },
  {
    title: t('strategy.status'), dataIndex: 'status', width: 100,
    render: (v: string) => (
      <Tag color={v === 'active' ? 'green' : 'default'}>
        {v}
      </Tag>
    ),
  },
  {
    title: '', width: 50,
    render: (_: any, record: any) => (
      <Popconfirm
        title={t('strategy.confirmDelete')}
        onConfirm={() => handleDelete(record.id)}
      >
        <Button size="small" danger icon={<DeleteOutlined />} />
      </Popconfirm>
    ),
  },
];
```

## ConnectRPC 客户端模式

```ts
// client/ 下的每个 API 模块导出一个或多个 service 对象：
import { strategyTemplateApi } from '@/client/strategy';
import { accountApi } from '@/client/account';
import { marketApi } from '@/client/market';

// Proto snake_case → JS camelCase (automatic via ConnectRPC):
strategyTemplateApi.list({})              // → { templates: [...] }
strategyTemplateApi.create({ name: 'x' })  // → { strategy: {...} }
accountApi.listAccounts({})               // → { accounts: [...] }
marketApi.getSymbols(accountId)           // → SymbolInfo[]

// 翻译文件位置：frontend/src/i18n/resources/zh-cn/strategy.ts
```

现有 client 模块：`strategy`, `account`, `market`, `analytics`, `trading`, `ai`, `admin`, `auth`, `log`, `backtestRuns`, `strategyAsset`, `strategyExperiment`, 等。

## i18n 约定

```ts
import { useTranslation } from 'react-i18next';
const { t } = useTranslation();

// 格式：t('<feature>.<key>', 'fallback中文')
t('strategy.create', '创建策略')
t('strategy.deleteSuccess', '删除成功')
t('accounts.bindTitle', '绑定MT账户')

// 翻译文件位置：frontend/src/i18n/resources/zh-cn/<feature>.ts
// 支持 locale：zh-cn, zh-tw, en, ja, vi
```

## 设计决策

- **表格行键**：始终用 `rowKey="id"`（匹配 proto 字段名）
- **表单布局**：垂直表单用 `layout="vertical"` + `<Space size="large" wrap>`；内联创建表单用 `layout="inline"`
- **按钮对齐**：给提交按钮的 `Form.Item` 设 `label=" "` 保持垂直对齐
- **错误状态**：fetch 失败 set 空数组，不要留旧数据
- **数据获取**：用 `useCallback` 包裹 fetch 函数，`useEffect` 直接调用（不使用 deferEffect 包装器 — 此模式已移除）

## 参考代码

- 完整页面：`frontend/src/pages/strategy/StrategyTemplatePage.tsx`
- 三步向导：`frontend/src/pages/accounts/BindAccount.tsx` (498 lines)
