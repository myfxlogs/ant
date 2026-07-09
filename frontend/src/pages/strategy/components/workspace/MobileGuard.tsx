import { Button, Result } from 'antd';
import { CodeOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';
import {
  MOBILE_TITLE_KEY, MOBILE_SUBTITLE_KEY, MOBILE_CTA_KEY,
} from '@/gen/ant/v1/i18n/strategy_workspace_keys';

/** Guard shown on narrow screens — the workspace needs ≥1200px to be usable. */
export default function MobileGuard() {
  const { t } = useTranslation();
  const navigate = useNavigate();

  return (
    <Result
      icon={<CodeOutlined style={{ color: '#D4AF37', fontSize: 64 }} />}
      title={t(MOBILE_TITLE_KEY)}
      subTitle={t(MOBILE_SUBTITLE_KEY)}
      extra={
        <Button type="primary" onClick={() => navigate('/strategy/templates')}>
          {t(MOBILE_CTA_KEY)}
        </Button>
      }
    />
  );
}
