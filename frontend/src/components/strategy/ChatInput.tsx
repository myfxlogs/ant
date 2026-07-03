import { Input, Button } from 'antd';
import { SendOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { PLACEHOLDER_KEY, SEND_KEY, REGENERATE_KEY } from '@/gen/ant/v1/i18n/strategy_gen_keys';

interface Props {
  value: string;
  onChange: (v: string) => void;
  onSend: () => void;
  disabled?: boolean;
  hasResult?: boolean;
}

export default function ChatInput({ value, onChange, onSend, disabled, hasResult }: Props) {
  const { t } = useTranslation();

  return (
    <div style={{ display: 'flex', gap: 8, padding: '8px 16px 12px', borderTop: '1px solid var(--ant-color-border)', flexShrink: 0 }}>
      <Input.TextArea
        rows={3}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        onPressEnter={(e) => { e.preventDefault(); onSend(); }}
        placeholder={t(PLACEHOLDER_KEY)}
        style={{ flex: 1, fontSize: 13, resize: 'none' }}
      />
      <Button
        type="primary"
        icon={<SendOutlined />}
        onClick={onSend}
        disabled={disabled || !value.trim()}
        style={{ whiteSpace: 'nowrap' }}
      >
        {hasResult ? t(REGENERATE_KEY) : t(SEND_KEY)}
      </Button>
    </div>
  );
}
