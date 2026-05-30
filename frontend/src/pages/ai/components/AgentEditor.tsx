import { useRef } from 'react';
import {
  Button,
  Card,
  Empty,
  Input,
  List,
  Popconfirm,
  Select,
  Space,
  Switch,
  Typography,
} from 'antd';
import { DownOutlined, RightOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import type { AIAgentDefinitionView } from '@/client/ai';

const { Text } = Typography;

interface ModelOption {
  value: string;
  label: string;
}

export function encodeAgentModel(a: AIAgentDefinitionView): string {
  if (!a.providerId) return '';
  return `${a.providerId}|${a.modelOverride || ''}`;
}

export function decodeAgentModel(value: string): { providerId: string; modelOverride: string } {
  if (!value) return { providerId: '', modelOverride: '' };
  const idx = value.indexOf('|');
  if (idx < 0) return { providerId: value, modelOverride: '' };
  return {
    providerId: value.slice(0, idx),
    modelOverride: value.slice(idx + 1),
  };
}

export function AgentEditor(props: {
  agents: AIAgentDefinitionView[];
  modelOptions: ModelOption[];
  saving: boolean;
  collapsedMap: Record<string, boolean | undefined>;
  onCollapsedChange: (key: string) => void;
  onChange: (idx: number, patch: Partial<AIAgentDefinitionView>) => void;
  onRemove: (idx: number) => void;
  onAdd: () => void;
  onLoadDefaults: () => void;
  onResetAllToDefaults: () => void;
  onSave: () => Promise<void>;
}) {
  const { t } = useTranslation();
  const {
    agents, modelOptions, saving, collapsedMap, onCollapsedChange,
    onChange, onRemove, onAdd, onLoadDefaults, onResetAllToDefaults, onSave,
  } = props;

  const lastItemRef = useRef<HTMLDivElement | null>(null);

  return (
    <Card
      title={t('ai.settings.agent.title')}
      extra={(
        <Space wrap>
          {agents.length === 0 ? (
            <Button size="small" onClick={onResetAllToDefaults}>
              {t('ai.settings.agent.actions.loadDefaults')}
            </Button>
          ) : (
            <Popconfirm
              title={t('ai.settings.agent.actions.restoreDefaultsConfirmTitle')}
              description={t('ai.settings.agent.actions.restoreDefaultsConfirmContent')}
              onConfirm={onLoadDefaults}
              okText={t('ai.settings.agent.actions.restoreDefaults')}
            >
              <Button size="small">
                {t('ai.settings.agent.actions.restoreDefaults')}
              </Button>
            </Popconfirm>
          )}
          <Button size="small" onClick={onAdd}>
            {t('ai.settings.agent.actions.add')}
          </Button>
          <Button size="small" type="primary" loading={saving} onClick={onSave}>
            {t('ai.settings.agent.actions.save')}
          </Button>
        </Space>
      )}
    >
      {agents.length === 0 ? (
        <Empty description={t('ai.settings.agent.messages.empty')}>
          <Button type="primary" onClick={onResetAllToDefaults}>
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
              : [...modelOptions, { value: cur, label: t('ai.settings.agent.fields.historicalBinding', { value: cur }) }];

            return (
              <List.Item
                ref={idx === agents.length - 1 ? lastItemRef : undefined}
                actions={[
                  <Switch
                    key="enabled"
                    checked={agent.enabled}
                    onChange={(v) => onChange(idx, { enabled: v })}
                  />,
                  <Button
                    key="remove"
                    type="link"
                    danger
                    size="small"
                    onClick={() => onRemove(idx)}
                  >
                    {t('ai.settings.agent.actions.remove')}
                  </Button>,
                ]}
              >
                <List.Item.Meta
                  title={(
                    <Space wrap>
                      {isSystemAgent ? (
                        <Text strong style={{ minWidth: 180, display: 'inline-block' }}>
                          {t(`ai.settings.agent.types.${agent.type}`)}
                        </Text>
                      ) : (
                        <Input
                          style={{ minWidth: 180 }}
                          value={agent.name}
                          placeholder={t('ai.settings.agent.fields.namePlaceholder')}
                          onChange={(e) => onChange(idx, { name: e.target.value })}
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
                          onChange(idx, dec);
                        }}
                      />
                      <Button
                        size="small"
                        type="link"
                        onClick={() => onCollapsedChange(key)}
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
                        onChange={(e) => onChange(idx, { identity: e.target.value })}
                      />
                      <Input.TextArea
                        rows={2}
                        value={agent.inputHint}
                        placeholder={t('ai.settings.agent.fields.inputHintPlaceholder')}
                        onChange={(e) => onChange(idx, { inputHint: e.target.value })}
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
  );
}
