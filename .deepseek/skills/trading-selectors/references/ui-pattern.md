# UI Pattern: Account + Symbol Picker Form

Complete working example from `frontend/src/pages/strategy/DatasetsPage.tsx`.

## Imports

```tsx
import { useState, useEffect, useCallback } from 'react';
import { Table, Tag, Typography, Button, Select, InputNumber, Card, Form, message, Space, Alert } from 'antd';
import { ReloadOutlined, BuildOutlined, DatabaseOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { researchClient } from '@/modules/research/client/research';
import { marketApi } from '@/client/market';
```

## Account Fetching

```tsx
const [accounts, setAccounts] = useState<any[]>([]);
const [accLoading, setAccLoading] = useState(false);

const fetchAccounts = useCallback(async () => {
  setAccLoading(true);
  try {
    const r = await researchClient.dataset.listAccounts({});
    const list = (r.accounts || [])
      .filter((a: any) => !a.isDisabled)
      .map((a: any) => ({
        id: a.accountId,
        login: a.accountNumber,
        brokerCompany: a.broker || '',
        alias: a.accountNumber,
      }));
    setAccounts(list);
  } catch {
    setAccounts([]);
  }
  setAccLoading(false);
}, []);
useEffect(() => { fetchAccounts(); }, [fetchAccounts]);
```

## Symbol Fetching (Live Broker)

```tsx
const [symbols, setSymbols] = useState<{ value: string; label: string }[]>([]);
const [symLoading, setSymLoading] = useState(false);
const [symError, setSymError] = useState<string | null>(null);
const selectedAccountId = Form.useWatch('accountId', form);

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

// Clear symbol when account changes
useEffect(() => { form.setFieldsValue({ symbol: undefined }); }, [selectedAccountId, form]);
```

## Form Layout

- `Form layout="vertical"` — labels above controls
- `<Space size="large" wrap align="start">` — all items in one row, wrap when needed
- Each `Form.Item` uses `minWidth`/`maxWidth` for responsive sizing
- Submit button `Form.Item` uses `label=" "` to align with other controls

```tsx
<Form form={form} layout="vertical" onFinish={onSubmit}>
  <Space size="large" wrap align="start">

    <Form.Item name="accountId" label="Account"
      rules={[{ required: true }]}
      help={accounts.length === 0 ? 'No accounts available' : undefined}>
      <Select style={{ minWidth: 200, maxWidth: 260 }}
        options={accountOptions} loading={accLoading}
        showSearch allowClear />
    </Form.Item>

    <Form.Item name="symbol" label="Symbol"
      rules={[{ required: true }]}
      help={!selectedAccountId ? 'Select an account first' : undefined}>
      {symError ? (
        <div style={{ display: 'flex', alignItems: 'center', gap: 8, minWidth: 200 }}>
          <Alert type="error" message={symError}
            style={{ flex: 1, padding: '4px 12px' }} />
          <Button size="small" onClick={fetchSymbols} loading={symLoading}>
            Retry
          </Button>
        </div>
      ) : (
        <Select style={{ minWidth: 140, maxWidth: 200 }}
          options={symbols}
          disabled={!selectedAccountId} loading={symLoading}
          showSearch allowClear />
      )}
    </Form.Item>

    <Form.Item label=" ">
      <Button type="primary" htmlType="submit"
        disabled={!selectedAccountId}>
        Submit
      </Button>
    </Form.Item>

  </Space>
</Form>
```

## Account Options Construction

```tsx
const accountOptions = accounts.map((a: any) => ({
  value: a.id,
  label: a.alias ? `${a.alias} (${a.login})` : `${a.login} — ${a.brokerCompany}`,
}));
```

## Anti-Patterns to Avoid

- **Don't use `researchClient.dataset.listSymbols`** — it queries `kline_data` (static cache),
  not live broker data. User-facing pickers need live broker symbols.
- **Don't use `usePageQuery` for symbols** — symbols must reactively re-fetch on
  account change; `usePageQuery` is for static lists.
- **Don't hardcode error messages** — `symError` should be `string | null` with
  the catch extracting `e?.message`.
- **Don't use `layout="inline"`** — labels should be above controls for readability.
- **Don't forget `setFieldsValue({ symbol: undefined })`** — stale symbol from
  previous account will cause build errors.
