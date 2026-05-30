import { useMemo } from 'react';
import { Alert, Button, Empty, Space, Spin, Tag, Typography, message as antMessage } from 'antd';
import { useTranslation } from 'react-i18next';
import type { AIAgentDefinitionView } from '@/client/ai';
import { useAgentLabel } from '../helpers';

const { Text } = Typography;

export function AgentSelectionStep(props: {
  agentDefs: AIAgentDefinitionView[];
  agentsLoading: boolean;
  selectedAgents: AIAgentDefinitionView[];
  onChange: (agents: AIAgentDefinitionView[]) => void;
  onNext: () => void;
}) {
  const { t } = useTranslation();
  const labelOf = useAgentLabel();
  const { agentDefs, agentsLoading, selectedAgents, onChange, onNext } = props;

  // 隐藏 code 类型：代码生成器固定参与最终步骤，不需要用户勾选。
  const selectable = useMemo(() => agentDefs.filter((a) => a.type !== 'code'), [agentDefs]);

  const selectedKeys = useMemo(() => new Set(selectedAgents.map((a) => a.agentKey || a.type)), [selectedAgents]);

  function toggle(a: AIAgentDefinitionView) {
    if (!a.enabled) {
      antMessage.warning(t('ai.debate.messages.enableAgentFirst', { defaultValue: 'This expert is disabled. Please enable it in AI Settings first.' }));
      return;
    }
    const key = a.agentKey || a.type;
    if (selectedKeys.has(key)) {
      onChange(selectedAgents.filter((x) => (x.agentKey || x.type) !== key));
    } else {
      onChange([...selectedAgents, a]);
    }
  }

  return (
    <div>
      <Alert
        className="ai-gold-alert"
        type="info"
        showIcon
        style={{ marginBottom: 12 }}
        message={t('ai.debate.v2.selectTitle', { defaultValue: 'Choose the experts for this session' })}
        description={t('ai.debate.v2.selectDesc', {
          defaultValue: 'Optional: if no expert is chosen, the system will skip directly to code generation after intent clarification. The selection order is also the speaking order.',
        })}
      />

      {agentsLoading && agentDefs.length === 0 ? (
        <div style={{ textAlign: 'center', padding: '32px 0' }}>
          <Spin />
          <div style={{ marginTop: 8 }}>
            <Text type="secondary">{t('ai.debate.messages.loadingAgents')}</Text>
          </div>
        </div>
      ) : selectable.length === 0 ? (
        <Empty description={t('ai.debate.messages.noAgentsHint')} />
      ) : (
        <div
          style={{
            display: 'grid',
            gridTemplateColumns: 'repeat(auto-fill, minmax(200px, 1fr))',
            gap: 12,
          }}
        >
          {selectable.map((a) => {
            const key = a.agentKey || a.type;
            const selected = selectedKeys.has(key);
            const order = selected
              ? selectedAgents.findIndex((x) => (x.agentKey || x.type) === key) + 1
              : 0;
            return (
              <div
                key={key}
                onClick={() => toggle(a)}
                style={{
                  cursor: 'pointer',
                  borderRadius: 8,
                  border: selected
                    ? '2px solid #d4af37'
                    : a.enabled
                    ? '1px solid #e5e7eb'
                    : '1px solid #f0f0f0',
                  background: selected
                    ? 'rgba(212, 175, 55, 0.08)'
                    : a.enabled
                    ? '#ffffff'
                    : '#fafafa',
                  padding: 12,
                  minHeight: 84,
                  display: 'flex',
                  flexDirection: 'column',
                  gap: 6,
                }}
              >
                {(() => {
                  const { name: label, hint } = labelOf(a);
                  return (
                    <>
                      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                        <Text strong>{label}</Text>
                        {selected ? <Tag color="gold">#{order}</Tag> : null}
                      </div>
                      <Text type="secondary" style={{ fontSize: 12 }}>
                        {hint || label}
                      </Text>
                    </>
                  );
                })()}
              </div>
            );
          })}
        </div>
      )}

      <div style={{ marginTop: 16, textAlign: 'right' }}>
        <Space>
          <Text type="secondary">
            {t('ai.debate.v2.selectedCount', {
              defaultValue: '{{count}} expert(s) selected',
              count: selectedAgents.length,
            })}
          </Text>
          <Button type="primary" onClick={onNext}>
            {selectedAgents.length === 0
              ? t('ai.debate.v2.nextNoAgents', { defaultValue: 'No expert needed, next' })
              : t('ai.debate.v2.next', { defaultValue: 'Next' })}
          </Button>
        </Space>
      </div>
    </div>
  );
}
