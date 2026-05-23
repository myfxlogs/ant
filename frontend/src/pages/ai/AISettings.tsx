import { useEffect, useMemo, useRef, useState } from 'react';
import {
  Alert,
  Button,
  Card,
  Empty,
  Input,
  List,
  Popconfirm,
  Select,
  Space,
  Spin,
  Switch,
  Typography,
  message,
} from 'antd';
import { DownOutlined, RightOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';

import { aiApi, type AIAgentDefinitionView } from '@/client/ai';
import { listSystemAIConfigs } from './systemai/api';
import type { AIConfig as SystemAIConfig } from './systemai/model';
import { useAgentStore } from './agentStore';
import {
  getDefaultAgentTemplates,
  mergeWithDefaultAgentTemplates,
} from './defaultAgentTemplates';

const { Text } = Typography;

// AISettings 自 060 起聚焦在「Agent 编辑」单一职责：
// - 厂商 Key / 默认模型 / 可用模型清单 全部移交 SystemAI 页（/ai/system）。
// - 本页只读 system_ai_configs 来填「provider × model」选择器。
// - 库里没有 Agent 时，由后端 ListAgents 自动 seed 8 个默认 Agent，
//   前端不再自行写入；首次访问看到的就是已 seed 的列表。
//
// 提交路径：aiApi.setAgents() → 整体替换；保存成功后同步进 useAgentStore，
// debate 页无需再次请求。

interface ModelOption {
  value: string; // "providerId|modelId"
  label: string;
}

function buildModelOptions(
  systemConfigs: SystemAIConfig[],
  labelOf: (id: string, dbName?: string) => string,
): ModelOption[] {
  return systemConfigs
    .filter((c) => c && c.provider_id && c.has_secret && c.enabled)
    .flatMap((c) => {
      const models = Array.from(
        new Set((c.models || []).map((m) => (m || '').trim()).filter(Boolean)),
      );
      // 没列出 models 时退回 default_model，至少能让用户挑到一个。
      const list = models.length > 0 ? models : (c.default_model ? [c.default_model] : []);
      return list.map((m) => ({
        value: `${c.provider_id}|${m}`,
        label: `${labelOf(c.provider_id, c.name)} · ${m}`,
      }));
    });
}

function encodeAgentModel(a: AIAgentDefinitionView): string {
  if (!a.providerId) return '';
  return `${a.providerId}|${a.modelOverride || ''}`;
}

function decodeAgentModel(value: string): { providerId: string; modelOverride: string } {
  if (!value) return { providerId: '', modelOverride: '' };
  const idx = value.indexOf('|');
  if (idx < 0) return { providerId: value, modelOverride: '' };
  return {
    providerId: value.slice(0, idx),
    modelOverride: value.slice(idx + 1),
  };
}

export default function AISettings() {
  const { t } = useTranslation();
  const navigate = useNavigate();

  const [agents, setAgentsState] = useState<AIAgentDefinitionView[]>([]);
  const [systemConfigs, setSystemConfigs] = useState<SystemAIConfig[]>([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [collapsedMap, setCollapsedMap] = useState<Record<string, boolean | undefined>>({});

  const lastItemRef = useRef<HTMLDivElement | null>(null);
  const prevCountRef = useRef(0);

  const labelOf = (id: string, dbName?: string) => {
    const key = `ai.settings.providers.${id}` as const;
    const tr = t(key);
    return tr && tr !== key ? tr : (dbName || id);
  };

  const modelOptions = useMemo(
    () => buildModelOptions(systemConfigs, labelOf),
    // labelOf 闭包包含 t / i18n，t 切换时仅影响展示，options.value 不变；
    // 故依赖只列实际数据源即可。
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [systemConfigs],
  );
  const hasUsableModel = modelOptions.length > 0;

  useEffect(() => {
    let mounted = true;
    (async () => {
      setLoading(true);
      try {
        const [list, sysList] = await Promise.all([
          aiApi.listAgents(),
          listSystemAIConfigs().then((r) => r.items).catch(() => [] as SystemAIConfig[]),
        ]);
        if (!mounted) return;
        setAgentsState(list);
        setSystemConfigs(sysList);
        useAgentStore.getState().setAgents(list);
      } catch (e: any) {
        if (!mounted) return;
        message.error(e?.message || t('ai.settings.agent.messages.saveFailed'));
      } finally {
        if (mounted) setLoading(false);
      }
    })();
    return () => {
      mounted = false;
    };
  }, [t]);

  useEffect(() => {
    if (agents.length > prevCountRef.current && lastItemRef.current) {
      lastItemRef.current.scrollIntoView({ behavior: 'smooth', block: 'start' });
    }
    prevCountRef.current = agents.length;
  }, [agents.length]);

  const handleChange = (idx: number, patch: Partial<AIAgentDefinitionView>) => {
    setAgentsState((prev) => prev.map((a, i) => (i === idx ? { ...a, ...patch } : a)));
  };

  const handleAdd = () => {
    setAgentsState((prev) => [
      ...prev,
      {
        id: '',
        agentKey: `custom-${Date.now()}`,
        type: 'custom',
        name: t('ai.settings.agent.defaultName'),
        identity: '',
        inputHint: '',
        enabled: true,
        position: prev.length,
        providerId: '',
        modelOverride: '',
      },
    ]);
  };

  const handleRemove = (idx: number) => {
    setAgentsState((prev) => prev.filter((_, i) => i !== idx));
  };

  const handleLoadDefaults = () => {
    setAgentsState((prev) => mergeWithDefaultAgentTemplates(prev, t));
    message.success(t('ai.settings.agent.messages.defaultsLoaded'));
  };

  const handleResetAllToDefaults = () => {
    setAgentsState(getDefaultAgentTemplates(t));
    message.success(t('ai.settings.agent.messages.defaultsLoaded'));
  };

  const handleSave = async () => {
    setSaving(true);
    try {
      const cleaned = agents.map((a, i) => ({
        ...a,
        position: i,
        name: (a.name || '').trim(),
        identity: (a.identity || '').trim(),
        inputHint: (a.inputHint || '').trim(),
        providerId: (a.providerId || '').trim(),
        modelOverride: (a.modelOverride || '').trim(),
      }));
      const saved = await aiApi.setAgents(cleaned);
      setAgentsState(saved);
      useAgentStore.getState().setAgents(saved);
      message.success(t('ai.settings.agent.messages.saveSuccess'));
    } catch (e: any) {
      message.error(e?.message || t('ai.settings.agent.messages.saveFailed'));
    } finally {
      setSaving(false);
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Spin size="large" />
      </div>
    );
  }

  return (
    <div className="p-6">
      <div className="mb-4">
        <Typography.Title level={3} style={{ margin: 0 }}>
          {t('ai.settings.pageTitle')}
        </Typography.Title>
      </div>

      {!hasUsableModel ? (
        <Alert
          type="warning"
          showIcon
          className="mb-4"
          message={t('ai.settings.agent.fields.modelProfileEmpty')}
          action={
            <Button type="primary" size="small" onClick={() => navigate('/ai/settings')}>
              {t('ai.requireConfig.actions.goSettings')}
            </Button>
          }
        />
      ) : null}

      <Card
        title={t('ai.settings.agent.title')}
        extra={(
          <Space wrap>
            {agents.length === 0 ? (
              <Button size="small" onClick={handleResetAllToDefaults}>
                {t('ai.settings.agent.actions.loadDefaults')}
              </Button>
            ) : (
              <Popconfirm
                title={t('ai.settings.agent.actions.restoreDefaultsConfirmTitle')}
                description={t('ai.settings.agent.actions.restoreDefaultsConfirmContent')}
                onConfirm={handleLoadDefaults}
                okText={t('ai.settings.agent.actions.restoreDefaults')}
              >
                <Button size="small">
                  {t('ai.settings.agent.actions.restoreDefaults')}
                </Button>
              </Popconfirm>
            )}
            <Button size="small" onClick={handleAdd}>
              {t('ai.settings.agent.actions.add')}
            </Button>
            <Button size="small" type="primary" loading={saving} onClick={handleSave}>
              {t('ai.settings.agent.actions.save')}
            </Button>
          </Space>
        )}
      >
        {agents.length === 0 ? (
          <Empty description={t('ai.settings.agent.messages.empty')}>
            <Button type="primary" onClick={handleResetAllToDefaults}>
              {t('ai.settings.agent.actions.loadDefaults')}
            </Button>
          </Empty>
        ) : (
          <List
            dataSource={agents}
            renderItem={(agent, idx) => {
              const isSystemAgent = agent.agentKey?.startsWith('default-');
              const key = agent.agentKey || agent.id || String(idx);
              const collapsed = collapsedMap[key] ?? true;
              const cur = encodeAgentModel(agent);
              const inList = !cur || modelOptions.some((o) => o.value === cur);
              const opts = inList
                ? modelOptions
                : [...modelOptions, { value: cur, label: `${cur}（历史绑定）` }];

              return (
                <List.Item
                  ref={idx === agents.length - 1 ? lastItemRef : undefined}
                  actions={[
                    <Switch
                      key="enabled"
                      checked={agent.enabled}
                      onChange={(v) => handleChange(idx, { enabled: v })}
                    />,
                    <Button
                      key="remove"
                      type="link"
                      danger
                      size="small"
                      onClick={() => handleRemove(idx)}
                    >
                      {t('ai.settings.agent.actions.remove')}
                    </Button>,
                  ]}
                >
                  <List.Item.Meta
                    title={(
                      <Space wrap>
                        {isSystemAgent ? (
                          // 系统内置 8 个 Agent：名字始终用 i18n，不展示 DB 里
                          // seed 时按当时 locale 烤进去的旧字符串，否则切换语言
                          // 后会出现「左中文 + 右英文」并存。
                          <Text strong style={{ minWidth: 180, display: 'inline-block' }}>
                            {t(`ai.settings.agent.types.${agent.type}` as any)}
                          </Text>
                        ) : (
                          <Input
                            style={{ minWidth: 180 }}
                            value={agent.name}
                            placeholder={t('ai.settings.agent.fields.namePlaceholder')}
                            onChange={(e) => handleChange(idx, { name: e.target.value })}
                          />
                        )}
                        <Select
                          size="small"
                          style={{ minWidth: 240 }}
                          allowClear
                          showSearch
                          optionFilterProp="label"
                          value={cur || undefined}
                          placeholder={
                            modelOptions.length === 0
                              ? t('ai.settings.agent.fields.modelProfileEmpty')
                              : t('ai.settings.agent.fields.modelProfilePlaceholder')
                          }
                          options={opts}
                          notFoundContent={t('ai.settings.agent.fields.modelProfileEmpty')}
                          onChange={(v) => {
                            const dec = decodeAgentModel(v || '');
                            handleChange(idx, dec);
                          }}
                        />
                        <Button
                          size="small"
                          type="link"
                          onClick={() =>
                            setCollapsedMap((prev) => ({
                              ...prev,
                              [key]: !(prev[key] ?? true),
                            }))
                          }
                        >
                          {collapsed ? <RightOutlined /> : <DownOutlined />}
                        </Button>
                      </Space>
                    )}
                    description={collapsed ? null : (
                      <div className="space-y-2" style={{ marginTop: 8 }}>
                        <Input.TextArea
                          rows={3}
                          value={agent.identity}
                          placeholder={t('ai.settings.agent.fields.identityPlaceholder')}
                          onChange={(e) => handleChange(idx, { identity: e.target.value })}
                        />
                        <Input.TextArea
                          rows={2}
                          value={agent.inputHint}
                          placeholder={t('ai.settings.agent.fields.inputHintPlaceholder')}
                          onChange={(e) => handleChange(idx, { inputHint: e.target.value })}
                        />
                      </div>
                    )}
                  />
                </List.Item>
              );
            }}
          />
        )}
      </Card>
    </div>
  );
}
