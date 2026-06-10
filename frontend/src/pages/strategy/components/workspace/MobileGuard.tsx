import { Button, Result } from 'antd';
import { CodeOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';

/** Guard shown on narrow screens — the workspace needs ≥1200px to be usable. */
export default function MobileGuard() {
  const { t } = useTranslation();
  const navigate = useNavigate();

  return (
    <Result
      icon={<CodeOutlined style={{ color: '#D4AF37', fontSize: 64 }} />}
      title={t('strategy.workspace.mobileTitle', { defaultValue: 'Desktop Required' })}
      subTitle={t('strategy.workspace.mobileSubtitle', { defaultValue: 'The Strategy Workspace needs a larger screen. Please switch to a desktop device, or browse strategy templates instead.' })}
      extra={
        <Button type="primary" onClick={() => navigate('/strategy/templates')}>
          {t('strategy.workspace.mobileCta', { defaultValue: 'Go to Strategy Templates' })}
        </Button>
      }
    />
  );
}
