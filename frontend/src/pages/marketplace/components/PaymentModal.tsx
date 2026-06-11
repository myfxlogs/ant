import { Modal, Descriptions, Tag, Typography, Button, Space, Alert } from 'antd';
import { WalletOutlined, ShoppingCartOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';
import type { PublishedStrategy } from '@/gen/ant/v1/marketplace_service_pb';

const { Text } = Typography;

interface Props {
  strategy: PublishedStrategy | null;
  walletBalance: string;
  open: boolean;
  loading: boolean;
  onConfirm: () => void;
  onCancel: () => void;
}

function parseBalance(s: string): number {
  return Number(s) || 0;
}

export default function PaymentModal({ strategy, walletBalance, open, loading, onConfirm, onCancel }: Props) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  if (!strategy) return null;

  const name = strategy.strategyName || strategy.title || 'Unknown';
  const priceAmount = Number(strategy.priceAmount || 0);
  const isFree = priceAmount <= 0;
  const balanceNum = parseBalance(walletBalance);
  const sufficient = balanceNum >= priceAmount;
  const afterBalance = balanceNum - priceAmount;

  return (
    <Modal
      title={t('marketplace.payment.title', '确认购买')}
      open={open}
      onCancel={onCancel}
      width={480}
      footer={null}
      destroyOnClose
    >
      <Descriptions column={1} size="small" bordered style={{ marginBottom: 16 }}>
        <Descriptions.Item label={t('marketplace.payment.strategyName', '策略')}>
          <Text strong>{name}</Text>
        </Descriptions.Item>
        <Descriptions.Item label={t('marketplace.payment.price', '价格')}>
          <Tag color={isFree ? 'green' : 'gold'} style={{ fontSize: 14, fontWeight: 600 }}>
            {isFree
              ? t('marketplace.card.free', '免费')
              : t('marketplace.payment.oneTimePurchase', '¥{{amount}} 一次性买断', { amount: priceAmount.toFixed(2) })}
          </Tag>
        </Descriptions.Item>
      </Descriptions>

      <Descriptions column={1} size="small" bordered style={{ marginBottom: 16 }}>
        <Descriptions.Item label={<><WalletOutlined /> {t('marketplace.payment.walletBalance', '我的余额')}</>}>
          <Text strong style={{ color: sufficient ? '#52c41a' : '#ff4d4f', fontSize: 16 }}>
            ¥{balanceNum.toFixed(2)}
          </Text>
        </Descriptions.Item>
        {sufficient && !isFree && (
          <Descriptions.Item label={t('marketplace.payment.balanceAfter', '购买后余额')}>
            <Text type="secondary">¥{afterBalance.toFixed(2)}</Text>
          </Descriptions.Item>
        )}
      </Descriptions>

      {!sufficient && !isFree && (
        <Alert
          type="error"
          style={{ marginBottom: 16 }}
          message={t('marketplace.payment.insufficientBalance', '余额不足')}
          description={
            <span>
              {t('marketplace.payment.depositPrompt', '请先充值后再购买。')}{' '}
              <Button type="link" size="small" onClick={() => navigate('/wallet')} style={{ padding: 0 }}>
                {t('marketplace.payment.goToDeposit', '去充值')} →
              </Button>
            </span>
          }
        />
      )}

      <div style={{ textAlign: 'right' }}>
        <Space>
          <Button onClick={onCancel}>{t('marketplace.payment.cancel', '取消')}</Button>
          <Button
            type="primary"
            icon={<ShoppingCartOutlined />}
            loading={loading}
            disabled={!sufficient && !isFree}
            onClick={onConfirm}
          >
            {loading
              ? t('marketplace.payment.purchasing', '处理中...')
              : t('marketplace.payment.confirm', '确认购买')}
          </Button>
        </Space>
      </div>
    </Modal>
  );
}
