import { useState, useEffect, useCallback } from 'react';
import { Table, Tag, Typography, Button, Select, InputNumber, Card, Form, message, Space, Alert, Popconfirm, DatePicker } from 'antd';
import { ReloadOutlined, BuildOutlined, DatabaseOutlined, DeleteOutlined } from '@ant-design/icons';
import { timestampFromDate } from '@bufbuild/protobuf/wkt';
import type { Dayjs } from 'dayjs';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';
import { researchClient, deleteResearchDataset } from '@/modules/research/client/research';
import { marketApi } from '@/client/market';
import { deferEffect } from '@/pages/strategy/lib/deferEffect';
import BuildProgressPanel, { type BuildProgressState } from '@/pages/strategy/BuildProgressPanel';

const { Text } = Typography;
const { RangePicker } = DatePicker;
const TIMEFRAMES = ['M1','M5','M15','M30','H1','H4','D1','W1','MN'];

const DatasetsPage: React.FC = () => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [form] = Form.useForm();
  const [building, setBuilding] = useState(false);
  const selectedAccountId = Form.useWatch('accountId', form);
  const selectedKind: number = Form.useWatch('kind', form);

  // ── Accounts — fetched via research service ──
  const [accounts, setAccounts] = useState<any[]>([]);
  const [accLoading, setAccLoading] = useState(false);
  const fetchAccounts = useCallback(async () => {
    setAccLoading(true);
    try {
      const r = await researchClient.dataset.listAccounts({});
      const list = (r.accounts || []).filter((a: any) => !a.isDisabled).map((a: any) => ({
        id: a.accountId, login: a.accountNumber, brokerCompany: a.broker || '',
        alias: a.accountNumber,
      }));
      setAccounts(list);
    } catch { setAccounts([]); }
    setAccLoading(false);
  }, []);
  useEffect(() => { deferEffect(fetchAccounts); }, [fetchAccounts]);

  // ── Symbols — fetched via market API (same pattern as experiments page) ──
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
      const opts: { value: string; label: string }[] = [];
      for (const s of list || []) {
        const v = String(s?.symbol || '').trim();
        if (!v || seen.has(v)) continue;
        seen.add(v);
        opts.push({ value: v, label: v });
      }
      setSymbols(opts);
    } catch (e: any) {
      setSymbols([]);
      setSymError(e?.message || String(e) || t('research.dataset.symbolFetchFailed', '无法拉取交易商品'));
    }
    setSymLoading(false);
  }, [selectedAccountId, t]);
  useEffect(() => { deferEffect(fetchSymbols); }, [fetchSymbols]);

  // ── Universes — fetched via research service ──
  const [universes, setUniverses] = useState<{ value: string; label: string }[]>([]);
  const [uniLoading, setUniLoading] = useState(false);
  const fetchUniverses = useCallback(async () => {
    setUniLoading(true);
    try {
      const r = await researchClient.universe.listUniverses({ limit: 100 });
      setUniverses((r.universes || []).map((u: any) => ({ value: u.id, label: u.name })));
    } catch { setUniverses([]); }
    setUniLoading(false);
  }, []);
  useEffect(() => { deferEffect(fetchUniverses); }, [fetchUniverses]);

  useEffect(() => { deferEffect(() => form.setFieldsValue({ symbol: undefined })); }, [selectedAccountId, form]);

  // ── Existing datasets ──
  const [datasets, setDatasets] = useState<any[]>([]);
  const [dsLoading, setDsLoading] = useState(false);
  const fetchDatasets = useCallback(async () => {
    setDsLoading(true);
    try { const r = await researchClient.dataset.listDatasets({ limit: 50 }); setDatasets(r.datasets); } catch {}
    setDsLoading(false);
  }, []);
  useEffect(() => { deferEffect(fetchDatasets); }, [fetchDatasets]);

  // ── Build ──
  const [buildProgress, setBuildProgress] = useState<BuildProgressState | null>(null);
  const buildDataset = async (values: any) => {
    setBuilding(true);
    setBuildProgress(null);
    try {
      const k = values.kind || 1;
      const range = values.range as [Dayjs, Dayjs] | undefined;
      const start = range?.[0]?.toDate();
      const end = range?.[1]?.toDate();
      if (!start || !end) throw new Error(t('research.dataset.selectRange', '请选择K线时间范围'));
      const [from, to] = start <= end ? [start, end] : [end, start];
      const base: any = {
        accountId: values.accountId, kind: k,
        primaryTf: values.timeframe,
        from: timestampFromDate(from),
        to: timestampFromDate(to),
        minBars: values.minBars, canonicalClock: 'utc',
      };
      if (k === 2 /* PANEL */) {
        const syms = values.symbol;
        base.symbols = Array.isArray(syms) ? syms : (syms ? [syms] : []);
      } else if (k === 3 /* UNIVERSE */) {
        base.universeId = values.universeId;
      } else {
        base.symbol = Array.isArray(values.symbol) ? values.symbol[0] : values.symbol;
      }
      const stream = researchClient.dataset.buildDataset(base);
      let finalError = '';
      for await (const p of stream) {
        setBuildProgress({
          step: p.step, totalSteps: p.totalSteps,
          label: p.label, current: p.current, subTotal: p.subTotal,
          detail: p.detail, done: p.done, error: p.error || '',
        });
        if (p.done) {
          if (p.error) {
            finalError = p.error;
          }
        }
      }
      if (finalError) {
        message.error(finalError);
      } else {
        message.success(t('research.dataset.buildSuccess'));
        fetchDatasets();
      }
    } catch (e: any) {
      message.error(e?.message || t('research.dataset.buildFailed'));
    }
    setBuilding(false);
  };

  // ── Delete ──
  const deleteDataset = async (id: string) => {
    try {
      await deleteResearchDataset(id);
      message.success(t('research.dataset.deleteSuccess', '删除成功'));
      fetchDatasets();
    } catch (e: any) {
      message.error(e?.message || t('research.dataset.deleteFailed', '删除失败'));
    }
  };

  const kindLabels = ['','SINGLE','PANEL','UNIVERSE'];
  const accountOptions = accounts.map((a: any) => ({
    value: a.id,
    label: a.alias ? `${a.alias} (${a.login})` : `${a.login} — ${a.brokerCompany}`,
  }));

  const columns = [
    { title: 'ID', dataIndex: 'id', width: 100, render: (v: string) =>
      <Text code copyable>{v?.slice(0, 8)}</Text> },
    { title: t('research.dataset.kind'), dataIndex: 'kind', width: 80, render: (v: number) => <Tag>{kindLabels[v] || v}</Tag> },
    { title: t('research.dataset.symbol'), dataIndex: 'symbols', render: (v: string[]) => v?.join(', ') },
    { title: 'TF', dataIndex: 'timeframe', width: 55 },
    { title: t('research.dataset.bars'), dataIndex: 'rowCount', width: 70 },
    { title: 'SHA256', dataIndex: 'sha256', width: 120, render: (v: string) => <Text code>{v?.slice(0, 12)}</Text> },
    { title: t('research.dataset.status'), dataIndex: 'invalidatedAt', render: (v: any) =>
      v ? <Tag color="red">{t('research.dataset.invalidated')}</Tag> : <Tag color="green">{t('research.dataset.valid')}</Tag> },
    { title: '', width: 50, render: (_: any, record: any) => (
      <Popconfirm title={t('research.dataset.confirmDelete', '确定删除此数据集？')}
        onConfirm={() => deleteDataset(record.id)}>
        <Button size="small" danger icon={<DeleteOutlined />} />
      </Popconfirm>
    )},
  ];

  return (
    <div>
      <Card size="small" style={{ marginBottom: 12 }} title={<><DatabaseOutlined /> {t('research.dataset.buildTitle', '构建数据集')}</>}>
        <Form form={form} layout="vertical" onFinish={buildDataset}>
          <Space size="large" wrap align="start">
            <Form.Item name="accountId" label={t('research.dataset.account')}
              rules={[{ required: true, message: t('research.dataset.selectAccountSymbol') }]}
              help={accounts.length === 0 ? t('research.dataset.noAccounts', '暂无交易账号') : undefined}>
              <Select style={{ minWidth: 200, maxWidth: 260 }} placeholder={t('research.dataset.account')}
                options={accountOptions} loading={accLoading} showSearch
                filterOption={(input, option) => (option?.label ?? '').toLowerCase().includes(input.toLowerCase())}
                allowClear />
            </Form.Item>

            {/* Symbol selector: visible for SINGLE/PANEL */}
            {selectedKind !== 3 && (
              <Form.Item name="symbol" label={t('research.dataset.symbol')}
                rules={[{ required: selectedKind !== 3, message: t('research.dataset.selectAccountSymbol') }]}
                help={!selectedAccountId ? t('research.dataset.pickAccountFirst', '请先选择交易账号') : undefined}>
                {symError ? (
                  <div style={{ display: 'flex', alignItems: 'center', gap: 8, minWidth: 200 }}>
                    <Alert type="error" message={symError} style={{ flex: 1, padding: '4px 12px' }} />
                    <Button size="small" onClick={fetchSymbols} loading={symLoading}>
                      {t('common.retry', '重试')}
                    </Button>
                  </div>
                ) : (
                  <Select mode={selectedKind === 2 ? 'multiple' : undefined}
                    style={{ minWidth: 140, maxWidth: selectedKind === 2 ? 320 : 200 }}
                    placeholder={t('research.dataset.symbol')}
                    options={symbols}
                    disabled={!selectedAccountId} loading={symLoading}
                    showSearch allowClear />
                )}
              </Form.Item>
            )}

            {/* Universe selector: visible only for UNIVERSE */}
            {selectedKind === 3 && (
              <Form.Item name="universeId" label={t('research.dataset.universe')}
                rules={[{ required: true, message: t('research.dataset.selectUniverse') }]}
                help={!uniLoading && universes.length === 0 ? (
                  <span>{t('research.dataset.noUniverses', '暂无标的池，请前往')}{' '}
                    <Button type="link" size="small" style={{ padding: 0 }}
                      onClick={() => navigate('/research/universes')}>
                      {t('research.tabs.universes')}
                    </Button>
                    {' '}{t('research.dataset.createUniverse', '页面创建')}
                  </span>
                ) : undefined}>
                <Select style={{ minWidth: 180, maxWidth: 280 }}
                  placeholder={t('research.dataset.selectUniverse')}
                  options={universes} loading={uniLoading}
                  showSearch allowClear
                  notFoundContent={uniLoading ? undefined : t('research.dataset.noUniverses', '暂无标的池')} />
              </Form.Item>
            )}

            <Form.Item name="timeframe" label={t('research.dataset.timeframe')}
              rules={[{ required: true, message: t('research.dataset.selectTimeframe', '请选择周期') }]}>
              <Select style={{ width: 85 }} options={TIMEFRAMES.map(v => ({ value: v, label: v }))} />
            </Form.Item>
            <Form.Item name="kind" label={t('research.dataset.kind')} initialValue={1}>
              <Select style={{ width: 120 }}
                options={[
                  {value:1,label:t('research.dataset.kindSingle')},
                  {value:2,label:t('research.dataset.kindPanel')},
                  {value:3,label:t('research.dataset.kindUniverse')},
                ]} />
            </Form.Item>
            <Form.Item name="range" label={t('research.dataset.range', 'K线时间范围（UTC）')}
              rules={[{ required: true, message: t('research.dataset.selectRange', '请选择K线时间范围') }]}>
              <RangePicker showTime style={{ width: 360 }} />
            </Form.Item>
            <Form.Item name="minBars" label={t('research.dataset.bars')}
              rules={[{ required: true, message: t('research.dataset.enterBars', '请输入K线数量') }]}>
              <InputNumber min={10} max={50000} style={{ width: 80 }} />
            </Form.Item>
            <Form.Item label=" ">
              <Button type="primary" htmlType="submit" icon={<BuildOutlined />} loading={building}
                disabled={!selectedAccountId}>
                {t('research.dataset.build')}
              </Button>
            </Form.Item>
          </Space>
        </Form>
      </Card>

      {/* ── Build progress panel ── */}
      {building && <BuildProgressPanel progress={buildProgress} />}

      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 12 }}>
        <Text strong>{t('research.tabs.datasets')}</Text>
        <Button icon={<ReloadOutlined />} onClick={fetchDatasets} loading={dsLoading}>{t('research.dataset.refresh')}</Button>
      </div>
      <Table size="small" dataSource={datasets} rowKey="id" loading={dsLoading} columns={columns} />
    </div>
  );
};

export default DatasetsPage;