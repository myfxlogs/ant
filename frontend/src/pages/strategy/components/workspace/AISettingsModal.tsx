import { Suspense, lazy } from 'react';
import { Modal, Spin } from 'antd';
import { SettingOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next'
import { PAGE_TITLE_KEY } from '@/gen/ant/v1/i18n/ai_settings_keys';

;

const SystemAI = lazy(() => import('@/pages/ai/SystemAI'));

interface Props {
  open: boolean;
  onClose: () => void;
}

export default function AISettingsModal({ open, onClose }: Props) {
  const { t } = useTranslation();

  return (
    <Modal title={<><SettingOutlined style={{ marginRight: 8 }} />{t(PAGE_TITLE_KEY, 'AI Settings')}</>}
      open={open} onCancel={onClose} footer={null} width={900} style={{ top: 8 }}>
      <div style={{ maxHeight: '80vh', overflowY: 'auto' }}>
        <style>{`
          .aisettings-modal .grid { grid-template-columns: repeat(2, minmax(0, 1fr)) !important; }
          .aisettings-modal .space-y-6 > .grid { grid-template-columns: repeat(3, minmax(0, 1fr)) !important; }
        `}</style>
        <div className="aisettings-modal">
        <Suspense fallback={<div style={{ textAlign: 'center', padding: 40 }}><Spin /></div>}>
          {open && <SystemAI />}
        </Suspense>
        </div>
      </div>
    </Modal>
  );
}
