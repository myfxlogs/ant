import { Modal, Input, Select, Space, Typography } from 'antd';
import type { TFunction } from 'i18next';

const { Text } = Typography;

interface Props {
  t: TFunction;
  open: boolean;
  priceModel: string;
  setPriceModel: (v: string) => void;
  priceAmount: string;
  setPriceAmount: (v: string) => void;
  saving: boolean;
  onSave: () => void;
  onCancel: () => void;
}

export default function PricingModal({
  t, open, priceModel, setPriceModel, priceAmount, setPriceAmount, saving, onSave, onCancel,
}: Props) {
  return (
    <Modal
      title={t('marketplace.autogen.editPricing')}
      open={open}
      onCancel={onCancel}
      onOk={onSave}
      confirmLoading={saving}
      okText={t('marketplace.autogen.save')}
    >
      <Space direction="vertical" style={{ width: '100%' }}>
        <div>
          <Text type="secondary" style={{ display: 'block', marginBottom: 4 }}>{t('marketplace.autogen.priceModel')}</Text>
          <Select value={priceModel} onChange={setPriceModel} style={{ width: '100%' }}
            options={[
              { value: 'free', label: t('marketplace.autogen.pricingFree') },
              { value: 'once', label: t('marketplace.autogen.pricingOnce') },
              { value: 'subscription', label: t('marketplace.autogen.pricingSubscription') },
            ]}
          />
        </div>
        {priceModel !== 'free' && (
          <div>
            <Text type="secondary" style={{ display: 'block', marginBottom: 4 }}>{t('marketplace.autogen.priceAmount')}</Text>
            <Input value={priceAmount} onChange={e => setPriceAmount(e.target.value)} type="number" prefix="$" />
          </div>
        )}
      </Space>
    </Modal>
  );
}
