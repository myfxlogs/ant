import { useEffect, useState } from 'react';
import { Alert, Button } from 'antd';
import { useNavigate } from 'react-router-dom';
import { listSystemAIConfigs } from '@/pages/ai/systemai/api';
import { useTranslation } from 'react-i18next';

// 自 060 起：「是否配好了 AI」唯一信源是 system_ai_configs。任意一行
// `enabled && has_secret && default_model 非空` 即视为可用，路由/Debate 等
// 入口只看这一个条件。
export default function RequireAIConfig(props: { children: React.ReactNode }) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [loading, setLoading] = useState(true);
  const [ok, setOk] = useState(false);

  useEffect(() => {
    let mounted = true;
    (async () => {
      try {
        const { items } = await listSystemAIConfigs();
        const pass = items.some(
          (it) => it.enabled && it.has_secret && (it.default_model || '').trim() !== '',
        );
        if (!mounted) return;
        setOk(pass);
      } catch {
        if (!mounted) return;
        setOk(false);
      } finally {
        if (mounted) {
          setLoading(false);
        }
      }
    })();
    return () => {
      mounted = false;
    };
  }, []);

  if (loading) return null;

  if (!ok) {
    return (
      <Alert
        type="warning"
        showIcon
        message={t('ai.requireConfig.title')}
        description={t('ai.requireConfig.description')}
        action={
          <Button type="primary" size="small" onClick={() => navigate('/ai/settings')}>
            {t('ai.requireConfig.actions.goSettings')}
          </Button>
        }
      />
    );
  }

  return <>{props.children}</>;
}
