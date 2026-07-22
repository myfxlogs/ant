import { Modal, Descriptions, Tag, Typography, Button, Space, Alert, Checkbox, Input, message } from 'antd';
import { WalletOutlined, ShoppingCartOutlined, WarningOutlined, TagOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';
import { useState } from 'react';
import type { PublishedStrategy } from '@/gen/ant/v1/marketplace_service_pb';
import { marketplaceClient } from '@/client/connect';

const { Text } = Typography;

interface Props {
  strategy: PublishedStrategy | null;
  walletBalance: string;
  open: boolean;
  loading: boolean;
  onConfirm: (couponCode?: string) => void;
  onCancel: () => void;
}

function parseBalance(s: string): number {
  return Number(s) || 0;
}

export default function PaymentModal({ strategy, walletBalance, open, loading, onConfirm, onCancel }: Props) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [riskAcknowledged, setRiskAcknowledged] = useState(false);
  const [couponCode, setCouponCode] = useState('');
  const [couponLoading, setCouponLoading] = useState(false);
  const [discountedAmount, setDiscountedAmount] = useState<string | null>(null);
  const [couponError, setCouponError] = useState('');
  if (!strategy) return null;

  const name = strategy.strategyName || strategy.title || 'Unknown';
  const originalPrice = Number(strategy.priceAmount || 0);
  const effectivePrice = discountedAmount ? Number(discountedAmount) : originalPrice;
  const isFree = effectivePrice <= 0;
  const balanceNum = parseBalance(walletBalance);
  const sufficient = balanceNum >= effectivePrice;
  const afterBalance = balanceNum - effectivePrice;

  const handleValidateCoupon = async () => {
    if (!couponCode.trim()) return;
    setCouponLoading(true);
    setCouponError('');
    try {
      const resp = await marketplaceClient.validateCoupon({
        code: couponCode.trim(),
        strategyId: strategy.strategyId,
        purchaseAmount: strategy.priceAmount || '0',
      });
      if (resp.valid) {
        setDiscountedAmount(resp.finalAmount);
        message.success(t('marketplace.payment.couponApplied', { discount: resp.discountAmount }));
      } else {
        setCouponError(resp.errorMessage || 'Invalid coupon');
        setDiscountedAmount(null);
      }
    } catch {
      setCouponError(t('marketplace.payment.couponError'));
      setDiscountedAmount(null);
    } finally {
      setCouponLoading(false);
    }
  };

  return (
    <Modal
      title={t('marketplace.payment.title')}
      open={open}
      onCancel={onCancel}
      width={480}
      footer={null}
      destroyOnClose
    >
      <Descriptions column={1} size="small" bordered style={{ marginBottom: 16 }}>
        <Descriptions.Item label={t('marketplace.payment.strategyName')}>
          <Text strong>{name}</Text>
        </Descriptions.Item>
        <Descriptions.Item label={t('marketplace.payment.price')}>
          <Tag color={isFree ? 'green' : 'gold'} style={{ fontSize: 14, fontWeight: 600 }}>
            {isFree
              ? t('marketplace.card.free')
              : t('marketplace.payment.oneTimePurchase', { amount: effectivePrice.toFixed(2) })}
          </Tag>
          {discountedAmount && originalPrice > effectivePrice && (
            <Text delete type="secondary" style={{ fontSize: 12, marginLeft: 8 }}>
              ¥{originalPrice.toFixed(2)}
            </Text>
          )}
        </Descriptions.Item>
      </Descriptions>

      {/* Coupon input */}
      {!isFree && (
        <div style={{ marginBottom: 16 }}>
          <Space.Compact style={{ width: '100%' }}>
            <Input
              prefix={<TagOutlined />}
              placeholder={t('marketplace.payment.couponPlaceholder')}
              value={couponCode}
              onChange={e => { setCouponCode(e.target.value); setDiscountedAmount(null); setCouponError(''); }}
              onPressEnter={handleValidateCoupon}
            />
            <Button loading={couponLoading} onClick={handleValidateCoupon}>
              {t('marketplace.payment.applyCoupon')}
            </Button>
          </Space.Compact>
          {couponError && <Text type="danger" style={{ fontSize: 12 }}>{couponError}</Text>}
        </div>
      )}

      <Descriptions column={1} size="small" bordered style={{ marginBottom: 16 }}>
        <Descriptions.Item label={<><WalletOutlined /> {t('marketplace.payment.walletBalance')}</>}>
          <Text strong style={{ color: sufficient ? '#52c41a' : '#ff4d4f', fontSize: 16 }}>
            ¥{balanceNum.toFixed(2)}
          </Text>
        </Descriptions.Item>
        {sufficient && !isFree && (
          <Descriptions.Item label={t('marketplace.payment.balanceAfter')}>
            <Text type="secondary">¥{afterBalance.toFixed(2)}</Text>
          </Descriptions.Item>
        )}
      </Descriptions>

      {!sufficient && !isFree && (
        <Alert
          type="error"
          style={{ marginBottom: 16 }}
          message={t('marketplace.payment.insufficientBalance')}
          description={
            <span>
              {t('marketplace.payment.depositPrompt')}{' '}
              <Button type="link" size="small" onClick={() => navigate('/wallet')} style={{ padding: 0 }}>
                {t('marketplace.payment.goToDeposit')} →
              </Button>
            </span>
          }
        />
      )}

      {/* Risk disclaimer */}
      {strategy.disclaimer && (
        <Alert
          type="warning"
          showIcon
          icon={<WarningOutlined />}
          style={{ marginBottom: 16 }}
          message={t('marketplace.payment.riskWarning')}
          description={strategy.disclaimer}
        />
      )}

      {/* Risk acknowledgment checkbox */}
      <div style={{ marginBottom: 16 }}>
        <Checkbox
          checked={riskAcknowledged}
          onChange={e => setRiskAcknowledged(e.target.checked)}
        >
          {t('marketplace.payment.riskAck')}
        </Checkbox>
      </div>

      <div style={{ textAlign: 'right' }}>
        <Space>
          <Button onClick={onCancel}>{t('marketplace.payment.cancel')}</Button>
          <Button
            type="primary"
            icon={<ShoppingCartOutlined />}
            loading={loading}
            disabled={(!sufficient && !isFree) || !riskAcknowledged}
            onClick={() => onConfirm(couponCode.trim() || undefined)}
          >
            {loading
              ? t('marketplace.payment.purchasing')
              : t('marketplace.payment.confirm')}
          </Button>
        </Space>
      </div>
    </Modal>
  );
}
