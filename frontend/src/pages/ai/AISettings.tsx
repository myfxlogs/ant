import { useEffect, useMemo, useState } from 'react';
import {
  Alert,
  Button,
  Spin,
  Typography,
  message,
} from 'antd';
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
import { AgentEditor, encodeAgentModel, decodeAgentModel } from './components/AgentEditor';

interface ModelOption {
  value: string;
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
      const list = models.length > 0 ? models : (c.default_model ? [c.default_model] : []);
      return list.map((m) => ({
        value: `${c.provider_id}|${m}`,
        label: `${labelOf(c.provider_id, c.name)} · ${m}`,
      }));
    });
}

export default function AISettings() {
  const { t } = useTranslation();
  const navigate = useNavigate();

  const [agents, setAgentsState] = useState<AIAgentDefinitionView[]>([]);
  const [systemConfigs, setSystemConfigs] = useState<SystemAIConfig[]>([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [collapsedMap, setCollapsedMap] = useState<Record<string, boolean | undefined>>({});

  const labelOf = (id: string, dbName?: string) => {
    const key = `ai.settings.providers.${id}` as const;
    const tr = t(key);
    return tr && tr !== key ? tr : (dbName || id);
  };

  const modelOptions = useMemo(
    () => buildModelOptions(systemConfigs, labelOf),
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

  const handleCollapsedChange = (key: string) => {
    setCollapsedMap((prev) => ({
      ...prev,
      [key]: !(prev[key] ?? true),
    }));
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

      <AgentEditor
        agents={agents}
        modelOptions={modelOptions}
        saving={saving}
        collapsedMap={collapsedMap}
        onCollapsedChange={handleCollapsedChange}
        onChange={handleChange}
        onRemove={handleRemove}
        onAdd={handleAdd}
        onLoadDefaults={handleLoadDefaults}
        onResetAllToDefaults={handleResetAllToDefaults}
        onSave={handleSave}
      />
    </div>
  );
}
