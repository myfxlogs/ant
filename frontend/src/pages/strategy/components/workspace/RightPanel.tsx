import { useState } from 'react';
import { Button, Tooltip, Drawer } from 'antd';
import { BulbOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import StrategyChat from '@/components/strategy/StrategyChat';
import MemoryContent from './MemoryContent';

interface Props {
  symbol?: string;
  timeframe?: string;
  accountId?: string;
  onApplyCode: (code: string) => void;
  onValidateResult?: (result: import('@/client/codeAssist').ValidateExtendedResult) => void;
  width?: number;
}

export default function RightPanel(props: Props) {
  const { t } = useTranslation();
  const width = props.width ?? 380;
  const [memoryOpen, setMemoryOpen] = useState(false);

  return (
    <div style={{
      width, minWidth: width, flexShrink: 0,
      background: 'var(--ant-color-bg-container)',
      borderLeft: '1px solid var(--ant-color-border)',
      display: 'flex', flexDirection: 'column', overflow: 'hidden',
    }}>
      {/* Header with Memory button */}
      <div style={{
        display: 'flex', alignItems: 'center', justifyContent: 'space-between',
        padding: '4px 12px', flexShrink: 0,
        borderBottom: '1px solid var(--ant-color-border)',
        fontSize: 11, fontWeight: 700, color: 'var(--ant-color-text-tertiary)',
      }}>
        <span>🤖 {t('strategy.workspace.aiAssistant')}</span>
        <Tooltip title={t('strategy.memory.title', 'Agent Memory')}>
          <Button size="small" type="text" icon={<BulbOutlined />} onClick={() => setMemoryOpen(true)} />
        </Tooltip>
      </div>

      {/* Chat — fills remaining space */}
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
        <StrategyChat
          symbol={props.symbol}
          timeframe={props.timeframe}
          accountId={props.accountId}
          onApplyCode={props.onApplyCode}
          onValidateResult={props.onValidateResult}
        />
      </div>

      <Drawer
        title={t('strategy.memory.title', 'Agent Memory')}
        open={memoryOpen}
        onClose={() => setMemoryOpen(false)}
        width={560}
        styles={{ body: { overflowY: 'auto' } }}
      >
        <MemoryContent />
      </Drawer>
    </div>
  );
}
