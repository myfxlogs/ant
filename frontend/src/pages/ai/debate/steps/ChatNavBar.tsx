import { Button, Divider, Typography } from 'antd';
import { useTranslation } from 'react-i18next';

const { Text } = Typography;

export function ChatNavBar(props: {
  sending: boolean;
  canBack: boolean;
  isLastAgent: boolean;
  onBack: () => void;
  onNext: () => void;
}) {
  const { t } = useTranslation();
  const { sending, canBack, isLastAgent, onBack, onNext } = props;

  return (
    <>
      <Divider style={{ margin: '16px 0 12px' }} />
      <div style={{ display: 'grid', gridTemplateColumns: '1fr auto 1fr', alignItems: 'center', gap: 12 }}>
        <div>
          <Button onClick={onBack} disabled={!canBack}>
            {t('ai.debate.v2.back', { defaultValue: 'Back' })}
          </Button>
        </div>
        <div style={{ textAlign: 'center' }}>
          <Button
            type="primary" size="large" onClick={onNext} loading={sending} disabled={sending}
            title={sending ? t('ai.debate.v2.nextDisabledWhileSending', {
              defaultValue: 'Please wait until the assistant finishes the current reply.',
            }) : undefined}
          >
            {isLastAgent
              ? t('ai.debate.v2.generateCode', { defaultValue: 'Generate code' })
              : t('ai.debate.v2.next', { defaultValue: 'Next' })}
          </Button>
        </div>
        <div style={{ textAlign: 'right' }}>
          <Text type="secondary" style={{ fontSize: 12 }}>
            {t('ai.debate.v2.nextHint', {
              defaultValue: 'Typing "next" or "continue" also advances the flow.',
            })}
          </Text>
        </div>
      </div>
    </>
  );
}
