import { Button, Modal } from 'antd';
import { useTranslation } from 'react-i18next';
import ScheduleHealthContent from './ScheduleHealthContent';

type Props = {
  open: boolean;
  target: Record<string, unknown> | null;
  loading: boolean;
  summary: Record<string, unknown> | null;
  onRefresh: () => void;
  onClose: () => void;
  formatTime: (v: unknown) => string;
};

export default function ScheduleHealthModal({ open, target, loading, summary, onRefresh, onClose, formatTime }: Props) {
  const { t } = useTranslation();
  return (
    <Modal
      title={t('strategy.schedules.health.title', { name: (target?.name as string) || '' })}
      open={open} onCancel={onClose} width={980}
      footer={[
        <Button key="refresh" onClick={onRefresh} loading={loading}>{t('common.refresh')}</Button>,
        <Button key="close" onClick={onClose}>{t('common.close')}</Button>,
      ]}>
      <ScheduleHealthContent summary={summary} loading={loading} formatTime={formatTime} />
    </Modal>
  );
}
