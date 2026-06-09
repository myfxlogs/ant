import { Modal, Form, InputNumber, Space, Button, ColorPicker } from 'antd';
import { useTranslation } from 'react-i18next';
import { useChartIndicatorsStore, type ActiveIndicator, type IndicatorDef } from '@/stores/chartIndicatorsStore';

interface Props {
  visible: boolean;
  indicator: ActiveIndicator;
  def: IndicatorDef;
  onClose: () => void;
}

export default function IndicatorSettingsModal({ visible, indicator, def, onClose }: Props) {
  const { t } = useTranslation();
  const { updateParams, removeIndicator } = useChartIndicatorsStore();
  const [form] = Form.useForm();

  const handleSave = () => {
    const values = form.getFieldsValue();
    const params: Record<string, number> = {};
    for (const p of def.params) {
      params[p.key] = Number(values[p.key] ?? indicator.params[p.key] ?? p.default);
    }
    updateParams(indicator.instanceId, params);
    onClose();
  };

  const handleRemove = () => {
    removeIndicator(indicator.instanceId);
    onClose();
  };

  return (
    <Modal
      title={t('common.indicatorSettings', { name: def.name })}
      open={visible}
      onCancel={onClose}
      onOk={handleSave}
      okText={t('common.save', 'Save')}
      cancelText={t('common.cancel', 'Cancel')}
      width={360}
      destroyOnClose
      footer={(_, { OkBtn, CancelBtn }) => (
        <div style={{ display: 'flex', justifyContent: 'space-between' }}>
          <Button danger onClick={handleRemove}>{t('common.delete', 'Remove')}</Button>
          <Space>
            <CancelBtn />
            <OkBtn />
          </Space>
        </div>
      )}
    >
      <Form
        form={form}
        layout="vertical"
        initialValues={indicator.params}
        style={{ marginTop: 16 }}
      >
        {def.params.map((p) => (
          <Form.Item
            key={p.key}
            label={p.label}
            name={p.key}
            rules={[
              { required: true },
              { type: 'number', min: p.min, max: p.max, message: `${p.min} – ${p.max}` },
            ]}
          >
            <InputNumber
              style={{ width: '100%' }}
              min={p.min}
              max={p.max}
              step={p.step}
              placeholder={String(p.default)}
            />
          </Form.Item>
        ))}

        {/* Line color */}
        <Form.Item label={t('common.lineColor')} name="_color">
          <ColorPicker showText style={{ width: '100%' }} />
        </Form.Item>
      </Form>
    </Modal>
  );
}
