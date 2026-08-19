import { useState } from 'react';
import { Modal, Button, Steps, Typography } from 'antd';
import { CloudServerOutlined, RobotOutlined, CrownOutlined, CheckCircleOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { useAuthStore } from '@/stores/authStore';

const DISMISS_KEY = 'ant-onboarding-dismissed';

interface Props {
  hasAccounts: boolean;
  hasStrategies: boolean;
}

export default function WelcomeModal({ hasAccounts, hasStrategies }: Props) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [open, setOpen] = useState(false);
  const { user } = useAuthStore();

  const [prevFlags, setPrevFlags] = useState({ hasAccounts, hasStrategies });
  if (hasAccounts !== prevFlags.hasAccounts || hasStrategies !== prevFlags.hasStrategies) {
    setPrevFlags({ hasAccounts, hasStrategies });
    try {
      const dismissed = localStorage.getItem(DISMISS_KEY);
      if (!dismissed && !hasAccounts && !hasStrategies) setOpen(true);
    } catch { /* ignore */ }
  }

  const dismiss = () => {
    try { localStorage.setItem(DISMISS_KEY, '1'); } catch { /* ignore */ }
    setOpen(false);
  };

  const stepItems = [
    {
      title: t('onboarding.step1.title', { defaultValue: 'Connect Your Account' }),
      description: t('onboarding.step1.desc', { defaultValue: 'Link your MT4/MT5 trading account to start.' }),
      icon: <CloudServerOutlined />,
      action: () => { dismiss(); navigate('/accounts/bind'); },
      actionLabel: t('onboarding.step1.action', { defaultValue: 'Bind Account' }),
      done: hasAccounts,
    },
    {
      title: t('onboarding.step2.title', { defaultValue: 'Create Your First Strategy' }),
      description: t('onboarding.step2.desc', { defaultValue: 'Use AI to generate a trading strategy from natural language.' }),
      icon: <RobotOutlined />,
      action: () => { dismiss(); navigate('/strategy/workspace'); },
      actionLabel: t('onboarding.step2.action', { defaultValue: 'Open Workspace' }),
      done: hasStrategies,
    },
    {
      title: t('onboarding.step3.title', { defaultValue: 'Upgrade Your Plan' }),
      description: t('onboarding.step3.desc', { defaultValue: 'Unlock more AI tokens, strategies, and live trading with Pro.' }),
      icon: <CrownOutlined />,
      action: () => { dismiss(); navigate('/subscription'); },
      actionLabel: t('onboarding.step3.action', { defaultValue: 'View Plans' }),
      done: false,
    },
  ];

  const currentStep = hasAccounts ? (hasStrategies ? 2 : 1) : 0;

  return (
    <Modal
      open={open}
      onCancel={dismiss}
      footer={null}
      width={520}
      centered
      closable
    >
      <div className="text-center mb-6">
        <h2 className="text-xl font-bold" style={{ color: 'var(--color-text)' }}>
          {t('onboarding.welcome', { defaultValue: 'Welcome to AlphaForge, {{name}}!', name: user?.email?.split('@')[0] || user?.nickname || '' })}
        </h2>
        <p className="mt-2 text-sm" style={{ color: 'var(--color-text-muted)' }}>
          {t('onboarding.subtitle', { defaultValue: 'Get started in 3 simple steps' })}
        </p>
      </div>

      <Steps
        current={currentStep}
        direction="vertical"
        size="small"
        items={stepItems.map((s) => ({
          title: (
            <div className="flex items-center gap-2">
              <span>{s.title}</span>
              {s.done && <CheckCircleOutlined style={{ color: 'var(--color-success)', fontSize: 14 }} />}
            </div>
          ),
          description: (
            <div className="mt-1">
              <Typography.Text type="secondary" style={{ fontSize: 12 }}>{s.description}</Typography.Text>
              {!s.done && (
                <Button
                  type="link"
                  size="small"
                  className="mt-1 p-0"
                  style={{ color: '#D4AF37' }}
                  onClick={s.action}
                >
                  {s.actionLabel} →
                </Button>
              )}
            </div>
          ),
          icon: s.icon,
        }))}
      />

      <div className="text-center mt-6">
        <Button onClick={dismiss} style={{ borderRadius: '8px' }}>
          {t('onboarding.dismiss', { defaultValue: 'Got it, dismiss' })}
        </Button>
      </div>
    </Modal>
  );
}
