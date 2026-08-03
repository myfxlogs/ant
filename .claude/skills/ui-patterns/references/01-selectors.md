---
name: ant-ui-patterns-01-selectors
description: >
  前端选择器组件模式。涵盖账号选择器（dropdown）、品种选择器（从 broker 实时拉取）、
  错误重试、选项去重。当需要给页面加账号/品种下拉框时使用。
---

# 01 — 选择器组件

## 账号选择器

数据来源：`accountApi.listAccounts`，过滤掉 disabled 账号。

```tsx
const [accounts, setAccounts] = useState<any[]>([]);
const [accLoading, setAccLoading] = useState(false);

const fetchAccounts = useCallback(async () => {
  setAccLoading(true);
  try {
    const r = await accountApi.listAccounts({});
    const list = (r.accounts || [])
      .filter((a: any) => !a.isDisabled)
      .map((a: any) => ({
        id: a.accountId,
        login: a.accountNumber,
        brokerCompany: a.broker || '',
        alias: a.accountNumber,
      }));
    setAccounts(list);
  } catch { setAccounts([]); }
  setAccLoading(false);
}, []);
useEffect(() => { fetchAccounts(); }, [fetchAccounts]);

const accountOptions = accounts.map((a: any) => ({
  value: a.id,
  label: a.alias ? `${a.alias} (${a.login})` : `${a.login} — ${a.brokerCompany}`,
}));

// JSX:
<Select
  style={{ minWidth: 200, maxWidth: 260 }}
  options={accountOptions}
  loading={accLoading}
  showSearch
  allowClear
  filterOption={(input, option) =>
    (option?.label ?? '').toLowerCase().includes(input.toLowerCase())
  }
/>
```

## 品种选择器

数据来源：`marketApi.getSymbols(accountId)` — 从 broker 实时拉取。**不要用 DB 查询**（`researchClient.dataset.listSymbols` 是静态缓存，不适合用户交互）。

```tsx
const selectedAccountId = Form.useWatch('accountId', form);
const [symbols, setSymbols] = useState<{ value: string; label: string }[]>([]);
const [symLoading, setSymLoading] = useState(false);
const [symError, setSymError] = useState<string | null>(null);

const fetchSymbols = useCallback(async () => {
  if (!selectedAccountId) { setSymbols([]); setSymError(null); return; }
  setSymLoading(true);
  setSymError(null);
  try {
    const list = await marketApi.getSymbols(selectedAccountId);
    const seen = new Set<string>();
    const opts = (list || [])
      .map((s: any) => String(s?.symbol || '').trim())
      .filter((v: string) => v)
      .filter((v: string) => { if (seen.has(v)) return false; seen.add(v); return true; })
      .map((v: string) => ({ value: v, label: v }));
    setSymbols(opts);
  } catch (e: any) {
    setSymbols([]);
    setSymError(e?.message || String(e) || 'Failed to fetch symbols');
  }
  setSymLoading(false);
}, [selectedAccountId]);
useEffect(() => { fetchSymbols(); }, [fetchSymbols]);

// 切换账号时清空品种
useEffect(() => {
  form.setFieldsValue({ symbol: undefined });
}, [selectedAccountId, form]);

// JSX — 正常状态:
<Select
  style={{ minWidth: 140, maxWidth: 200 }}
  options={symbols}
  disabled={!selectedAccountId}
  loading={symLoading}
  showSearch allowClear
/>

// JSX — 错误状态（带重试按钮）:
{symError ? (
  <div style={{ display: 'flex', alignItems: 'center', gap: 8, minWidth: 200 }}>
    <Alert type="error" message={symError} style={{ flex: 1, padding: '4px 12px' }} />
    <Button size="small" onClick={fetchSymbols} loading={symLoading}>Retry</Button>
  </div>
) : (
  <Select ... />
)}
```

## 后端依赖

前端 `marketApi.getSymbols(accountId)` 最终调用 MT4 `Symbols` RPC。详见 `mt-gateway` skill 的 [02-symbols.md](../mt-gateway/references/02-symbols.md)。

## 关键原则

- **品种源**：必须用 `marketApi.getSymbols`（实时 broker 数据），不能用 `researchClient.dataset.listSymbols`（kline_data 静态缓存）
- **错误处理**：拉取失败时用 `<Alert>` + 重试按钮替换 Select 组件，不要留空白
- **去重**：broker 可能返回重复品种名，必须 `Set` 去重
- **类型**：`symbols` 状态类型是 `{ value: string; label: string }[]`，直接匹配 Ant Design Select 的 `options` prop
