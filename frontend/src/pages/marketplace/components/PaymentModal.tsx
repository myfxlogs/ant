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
        message.success(t('marketplace.payment.couponApplied', { defaultValue: 'Coupon applied! Discount: ' + resp.discountAmount }));
      } else {
        setCouponError(resp.errorMessage || 'Invalid coupon');
        setDiscountedAmount(null);
      }
    } catch {
      setCouponError(t('marketplace.payment.couponError', { defaultValue: 'Failed to validate coupon' }));
      setDiscountedAmount(null);
    } finally {
      setCouponLoading(false);
    }
  };

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
              : t('marketplace.payment.oneTimePurchase', '¥{{amount}} 一次性买断', { amount: effectivePrice.toFixed(2) })}
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
              placeholder={t('marketplace.payment.couponPlaceholder', { defaultValue: 'Enter coupon code...' })}
              value={couponCode}
              onChange={e => { setCouponCode(e.target.value); setDiscountedAmount(null); setCouponError(''); }}
              onPressEnter={handleValidateCoupon}
            />
            <Button loading={couponLoading} onClick={handleValidateCoupon}>
              {t('marketplace.payment.applyCoupon', { defaultValue: 'Apply' })}
            </Button>
          </Space.Compact>
          {couponError && <Text type="danger" style={{ fontSize: 12 }}>{couponError}</Text>}
        </div>
      )}

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

      {/* Risk disclaimer */}
      {strategy.disclaimer && (
        <Alert
          type="warning"
          showIcon
          icon={<WarningOutlined />}
          style={{ marginBottom: 16 }}
          message={t('marketplace.payment.riskWarning', { defaultValue: 'Risk Disclaimer' })}
          description={strategy.disclaimer}
        />
      )}

      {/* Risk acknowledgment checkbox */}
      <div style={{ marginBottom: 16 }}>
        <Checkbox
          checked={riskAcknowledged}
          onChange={e => setRiskAcknowledged(e.target.checked)}
        >
          {t('marketplace.payment.riskAck', { defaultValue: 'I understand the risks associated with trading strategies and accept full responsibility for my investment decisions.' })}
        </Checkbox>
      </div>

      <div style={{ textAlign: 'right' }}>
        <Space>
          <Button onClick={onCancel}>{t('marketplace.payment.cancel', '取消')}</Button>
          <Button
            type="primary"
            icon={<ShoppingCartOutlined />}
            loading={loading}
            disabled={(!sufficient && !isFree) || !riskAcknowledged}
            onClick={() => onConfirm(couponCode.trim() || undefined)}
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
