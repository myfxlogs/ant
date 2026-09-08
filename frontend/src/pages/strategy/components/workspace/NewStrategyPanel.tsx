import { useState } from 'react';
import { ImportOutlined, RobotOutlined, EditOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { IMPORT_MQL_KEY, AI_GENERATE_KEY } from '@/gen/ant/v1/i18n/strategy_workspace_keys';
import type { NewSource } from './WorkspaceSidebar';

export type WorkspaceSection = 'new' | 'strategies' | 'history';

interface Props {
  onNewSource: (source: NewSource) => void;
}

// 新建策略来源选择（主区面板）：分区切换后展示的第一步。
// 四个来源各自带完成态承诺；选择后由父级切换到对应工作流。
export default function NewStrategyPanel({ onNewSource }: Props) {
  const { t } = useTranslation();
  const [hovered, setHovered] = useState('');

  const cardStyle = (key: string): React.CSSProperties => ({
    width: 220, padding: '18px 16px', borderRadius: 10, cursor: 'pointer', textAlign: 'left',
    border: `1px solid ${hovered === key ? 'var(--color-info)' : 'var(--ant-color-border)'}`,
    background: 'var(--ant-color-bg-container)',
    boxShadow: hovered === key ? '0 2px 8px rgba(0,0,0,0.09)' : 'none',
    transition: 'border-color 0.15s, box-shadow 0.15s',
  });
  const titleStyle: React.CSSProperties = { fontSize: 14, fontWeight: 600, color: 'var(--ant-color-text)', marginTop: 10 };
  const descStyle: React.CSSProperties = { fontSize: 12, color: 'var(--ant-color-text-secondary)', marginTop: 6, lineHeight: 1.6 };

  const cards = [
    {
      key: 'ai', icon: <RobotOutlined style={{ fontSize: 22, color: 'var(--color-ai-btn)' }} />,
      title: t(AI_GENERATE_KEY, { defaultValue: 'AI 生成' }),
      desc: t('strategy.workspace.new.aiDesc', { defaultValue: '描述你的想法，AI 会追问关键参数并自动生成、编译、回测。' }),
      action: () => onNewSource('ai'),
    },
    {
      key: 'manual', icon: <EditOutlined style={{ fontSize: 22, color: 'var(--color-info)' }} />,
      title: t('strategy.workspace.new.manual', { defaultValue: '手动编写' }),
      desc: t('strategy.workspace.new.manualDesc', { defaultValue: '打开代码编辑器，从零编写策略（Python 子集），保存后可回测。' }),
      action: () => onNewSource('manual'),
    },
    {
      key: 'import', icon: <ImportOutlined style={{ fontSize: 22, color: 'var(--color-success)' }} />,
      title: t(IMPORT_MQL_KEY, { defaultValue: '导入 MQL' }),
      desc: t('strategy.workspace.new.importDesc', { defaultValue: '上传/粘贴 MQL4/MQL5 源码，自动获得可运行度报告与盲区翻译建议。' }),
      action: () => onNewSource('import'),
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
              onClick={c.action}
              onMouseEnter={() => setHovered(c.key)}
              onMouseLeave={() => setHovered('')}>
              {c.icon}
              <div style={titleStyle}>{c.title}</div>
              <div style={descStyle}>{c.desc}</div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
