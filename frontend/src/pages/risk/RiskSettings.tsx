import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Card, Form, InputNumber, Switch, Button, message, Space, Typography } from 'antd'
import { SaveOutlined } from '@ant-design/icons'

const { Title, Text } = Typography

interface RiskProfile {
  max_positions: number
  daily_loss_limit: number | null
  max_drawdown_pct: number | null
  margin_level_min: number
  session_enabled: boolean
  slippage_max_points: number | null
  reject_rate_max: number | null
  killswitch_enabled: boolean
}

export default function RiskSettings() {
  const { t } = useTranslation()
  const [loading, setLoading] = useState(false)
  const [profile, setProfile] = useState<RiskProfile>({
    max_positions: 5,
    daily_loss_limit: null,
    max_drawdown_pct: null,
    margin_level_min: 150,
    session_enabled: true,
    slippage_max_points: null,
    reject_rate_max: null,
    killswitch_enabled: false,
  })

  const handleSave = async () => {
    setLoading(true)
    try {
      // TODO: connect to backend API when available
      message.success('Risk settings saved')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div style={{ maxWidth: 600, margin: '0 auto', padding: 24 }}>
      <Title level={3}>{t('risk.title', '风控偏好设置')}</Title>
      <Text type="secondary" style={{ marginBottom: 24, display: 'block' }}>
        {t('risk.description', '设置您的全局风控参数。所有策略将自动继承这些限制。')}
      </Text>

      <Card>
        <Form layout="vertical">
          <Form.Item label={t('risk.max_positions', '最大持仓数')} help={t('risk.max_positions_help', '同时持有的最大仓位数量')}>
            <InputNumber
              min={1} max={50}
              value={profile.max_positions}
              onChange={(v) => setProfile({ ...profile, max_positions: v ?? 5 })}
              style={{ width: '100%' }}
            />
          </Form.Item>

          <Form.Item label={t('risk.daily_loss', '日内最大亏损 (USD)')} help={t('risk.daily_loss_help', '当日累计亏损超过此值将暂停交易。留空 = 不限制')}>
            <InputNumber
              min={0}
              value={profile.daily_loss_limit}
              onChange={(v) => setProfile({ ...profile, daily_loss_limit: v })}
              placeholder="不限制"
              style={{ width: '100%' }}
            />
          </Form.Item>

          <Form.Item label={t('risk.max_drawdown', '最大回撤 (%)')} help={t('risk.max_drawdown_help', '从账户峰值回撤超过此比例将暂停交易')}>
            <InputNumber
              min={1} max={99}
              value={profile.max_drawdown_pct}
              onChange={(v) => setProfile({ ...profile, max_drawdown_pct: v })}
              placeholder="不限制"
              style={{ width: '100%' }}
            />
          </Form.Item>

          <Form.Item label={t('risk.margin_level', '最低保证金水平 (%)')} help={t('risk.margin_level_help', '净值/已用保证金低于此值将拒绝开仓')}>
            <InputNumber
              min={100} max={1000}
              value={profile.margin_level_min}
              onChange={(v) => setProfile({ ...profile, margin_level_min: v ?? 150 })}
              style={{ width: '100%' }}
            />
          </Form.Item>

          <Form.Item label={t('risk.killswitch', '紧急熔断')} help={t('risk.killswitch_help', '开启后将拒绝所有新订单，已有订单不受影响')}>
            <Switch
              checked={profile.killswitch_enabled}
              onChange={(v) => setProfile({ ...profile, killswitch_enabled: v })}
              checkedChildren="ON"
              unCheckedChildren="OFF"
            />
          </Form.Item>
        </Form>

        <Space style={{ marginTop: 16 }}>
          <Button type="primary" icon={<SaveOutlined />} loading={loading} onClick={handleSave}>
            {t('common.save', '保存')}
          </Button>
        </Space>
      </Card>
    </div>
  )
}
