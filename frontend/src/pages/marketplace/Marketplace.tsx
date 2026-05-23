import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Card, List, Tag, Input, Select, Typography, Space, Empty } from 'antd'
import { SearchOutlined, StarOutlined, DollarOutlined } from '@ant-design/icons'

const { Title, Text } = Typography
const { Search } = Input

interface MarketplaceItem {
  id: string
  title: string
  description: string
  asset_class: string
  risk_level: string
  price_model: string
  total_subscribers: number
  win_rate: number | null
  symbols: string[]
}

export default function Marketplace() {
  const { t } = useTranslation()
  const [items] = useState<MarketplaceItem[]>([])
  const [assetFilter, setAssetFilter] = useState<string>('')

  return (
    <div style={{ maxWidth: 900, margin: '0 auto', padding: 24 }}>
      <Title level={3}>{t('marketplace.title', '策略市场')}</Title>
      <Text type="secondary" style={{ marginBottom: 24, display: 'block' }}>
        {t('marketplace.description', '浏览社区分享的交易策略，订阅或购买你感兴趣的策略')}
      </Text>

      <Space style={{ marginBottom: 16, width: '100%' }} direction="vertical">
        <Search
          placeholder={t('marketplace.search', '搜索策略名称或品种...')}
          prefix={<SearchOutlined />}
          style={{ maxWidth: 400 }}
        />
        <Select
          value={assetFilter}
          onChange={setAssetFilter}
          placeholder={t('marketplace.filter_asset', '按资产类别筛选')}
          allowClear
          style={{ minWidth: 160 }}
          options={[
            { value: 'forex', label: '外汇' },
            { value: 'crypto', label: '加密货币' },
            { value: 'commodity', label: '商品' },
            { value: 'index', label: '指数' },
          ]}
        />
      </Space>

      {items.length === 0 ? (
        <Empty description={t('marketplace.empty', '还没有已发布的策略。成为第一个分享策略的人！')} />
      ) : (
        <List
          itemLayout="vertical"
          dataSource={items}
          renderItem={(item) => (
            <Card hoverable style={{ marginBottom: 12 }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <div>
                  <Title level={5} style={{ margin: 0 }}>{item.title}</Title>
                  <Text type="secondary">{item.description}</Text>
                </div>
                <Space>
                  {item.price_model !== 'free' && (
                    <Tag icon={<DollarOutlined />} color="gold">
                      {item.price_model === 'monthly' ? '月租' : '买断'}
                    </Tag>
                  )}
                  <Tag color={item.risk_level === 'high' ? 'red' : item.risk_level === 'medium' ? 'orange' : 'green'}>
                    {item.risk_level === 'high' ? '高风险' : item.risk_level === 'medium' ? '中风险' : '低风险'}
                  </Tag>
                </Space>
              </div>
              <div style={{ marginTop: 8 }}>
                {item.symbols.map((s) => (
                  <Tag key={s} color="blue" style={{ marginBottom: 4 }}>{s}</Tag>
                ))}
              </div>
              <div style={{ marginTop: 8, display: 'flex', gap: 24 }}>
                <Text type="secondary">
                  <StarOutlined /> {item.total_subscribers} 订阅
                </Text>
                {item.win_rate != null && (
                  <Text type="secondary">胜率: {item.win_rate}%</Text>
                )}
              </div>
            </Card>
          )}
        />
      )}
    </div>
  )
}
