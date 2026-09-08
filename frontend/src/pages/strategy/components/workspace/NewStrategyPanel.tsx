import { useState } from 'react';
import { Empty } from 'antd';
import { ImportOutlined, FileTextOutlined, RobotOutlined, EditOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { IMPORT_MQL_KEY, AI_GENERATE_KEY, USE_TEMPLATE_KEY } from '@/gen/ant/v1/i18n/strategy_workspace_keys';

export type WorkspaceSection = 'new' | 'strategies' | 'history';

interface Props {
  onAI: () => void;
  onManual: () => void;
  onImport: () => void;
  onTemplate: () => void;
  templateCount: number;
}

// 新建策略来源选择（主区面板）：分区切换后展示的第一步。
// 四个来源各自带完成态承诺；选择后由父级切换到对应工作流。
export default function NewStrategyPanel({ onAI, onManual, onImport, onTemplate, templateCount }: Props) {
  const { t } = useTranslation();
  const [hovered, setHovered] = useState('');

  const cardStyle = (key: string): React.CSSProperties => ({
    width: 220, padding: '18px 16px', borderRadius: 10, cursor: 'pointer', textAlign: 'left',
    border: `1px solid ${hovered === key ? 'var(--color-info)' : 'var(--ant-color-border)'}`,
    background: 'var(--ant-color-bg-container)',
    boxShadow: hovered === key ? '0 2px 8px rgba(0,0,0,0.09)' : 'none',
    transition: 'border-color 0.15s, box-shadow 0.15s',
    opacity: key === 'template' && templateCount === 0 ? 0.5 : 1,
  });
  const titleStyle: React.CSSProperties = { fontSize: 14, fontWeight: 600, color: 'var(--ant-color-text)', marginTop: 10 };
  const descStyle: React.CSSProperties = { fontSize: 12, color: 'var(--ant-color-text-secondary)', marginTop: 6, lineHeight: 1.6 };

  const cards = [
    {
      key: 'ai', icon: <RobotOutlined style={{ fontSize: 22, color: 'var(--color-ai-btn)' }} />,
      title: t(AI_GENERATE_KEY, { defaultValue: 'AI 生成' }),
      desc: t('strategy.workspace.new.aiDesc', { defaultValue: '描述你的想法，AI 会追问关键参数并自动生成、编译、回测。' }),
      action: onAI,
    },
    {
      key: 'manual', icon: <EditOutlined style={{ fontSize: 22, color: 'var(--color-info)' }} />,
      title: t('strategy.workspace.new.manual', { defaultValue: '手动编写' }),
      desc: t('strategy.workspace.new.manualDesc', { defaultValue: '打开代码编辑器，从零编写策略（Python 子集），保存后可回测。' }),
      action: onManual,
    },
    {
      key: 'import', icon: <ImportOutlined style={{ fontSize: 22, color: 'var(--color-success)' }} />,
      title: t(IMPORT_MQL_KEY, { defaultValue: '导入 MQL' }),
      desc: t('strategy.workspace.new.importDesc', { defaultValue: '上传/粘贴 MQL4/MQL5 源码，自动获得可运行度报告与盲区翻译建议。' }),
      action: onImport,
    },
    {
      key: 'template', icon: <FileTextOutlined style={{ fontSize: 22, color: 'var(--color-warning)' }} />,
      title: t(USE_TEMPLATE_KEY, { defaultValue: '使用模板' }),
      desc: t('strategy.workspace.new.templateDesc', { defaultValue: '从已验证的模板策略起步，改参数即可回测。' }),
      action: onTemplate,
      disabled: templateCount === 0,
    },
  ];

  return (
    <div style={{ flex: '1 1 0', display: 'flex', alignItems: 'center', justifyContent: 'center', overflow: 'auto' }}>
      <div>
        <div style={{ fontSize: 18, fontWeight: 600, color: 'var(--ant-color-text)', marginBottom: 4 }}>
          {t('strategy.workspace.new.title', { defaultValue: '新建策略' })}
        </div>
        <div style={{ fontSize: 13, color: 'var(--ant-color-text-secondary)', marginBottom: 20 }}>
          {t('strategy.workspace.new.subtitle', { defaultValue: '选择一种开始方式' })}
        </div>
        <div style={{ display: 'flex', gap: 14, flexWrap: 'wrap' }}>
          {cards.map((c) => (
            <div key={c.key} data-testid={`new-source-${c.key}`}
              style={cardStyle(c.key)}
              onClick={c.disabled ? undefined : c.action}
              onMouseEnter={() => setHovered(c.key)}
              onMouseLeave={() => setHovered('')}>
              {c.icon}
              <div style={titleStyle}>{c.title}</div>
              <div style={descStyle}>{c.desc}</div>
            </div>
          ))}
        </div>
        {templateCount === 0 && (
          <Empty description={t('strategy.workspace.new.noTemplates', { defaultValue: '暂无模板' })} style={{ marginTop: 16 }} />
        )}
      </div>
    </div>
  );
}
