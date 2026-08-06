import { useState, useEffect } from 'react';
import { Tour } from 'antd';
import type { TourProps } from 'antd';
import { useTranslation } from 'react-i18next';

const STORAGE_KEY = 'alphaforge_ws_tour_done';

export default function WorkspaceTour() {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);

  useEffect(() => {
    if (!localStorage.getItem(STORAGE_KEY)) {
      const timer = setTimeout(() => setOpen(true), 500);
      return () => clearTimeout(timer);
    }
  }, []);

  const handleClose = () => {
    setOpen(false);
    localStorage.setItem(STORAGE_KEY, 'true');
  };

  const steps: TourProps['steps'] = [
    {
      title: t('strategy.workspace.tour.ai', { defaultValue: 'AI Assistant' }),
      description: t('strategy.workspace.tour.aiDesc', { defaultValue: 'Ask AI to generate, optimize, or debug your strategy. Applied code appears in the editor instantly.' }),
      target: () => document.querySelector('[data-tour="ai-assistant"]') as HTMLElement,
    },
    {
      title: t('strategy.workspace.tour.code', { defaultValue: 'Code Editor' }),
      description: t('strategy.workspace.tour.codeDesc', { defaultValue: 'Write or paste your MQL strategy code here. You can also import .mq4/.mq5 files from the Import MQL tab.' }),
      target: () => document.querySelector('[data-tour="code-editor"]') as HTMLElement,
    },
    {
      title: t('strategy.workspace.tour.backtest', { defaultValue: 'Backtest' }),
      description: t('strategy.workspace.tour.backtestDesc', { defaultValue: 'Run backtests with configurable parameters. View equity curve, trade statistics, and risk metrics.' }),
      target: () => document.querySelector('[data-tour="backtest"]') as HTMLElement,
    },
    {
      title: t('strategy.workspace.tour.save', { defaultValue: 'Save & Publish' }),
      description: t('strategy.workspace.tour.saveDesc', { defaultValue: 'Save your strategy as a template, publish to marketplace, or deploy to a live schedule.' }),
      target: () => document.querySelector('[data-tour="save"]') as HTMLElement,
    },
  ];

  return <Tour open={open} onClose={handleClose} steps={steps} />;
}
