import { Tag, Space, Tooltip } from 'antd'
import { CheckCircleOutlined, WarningOutlined, CloseCircleOutlined } from '@ant-design/icons'
import type { ValidateResult, ResolvedSymbol } from '@/types/symbol'
import { useTranslation } from 'react-i18next'

interface Props {
  result: ValidateResult | null
  loading?: boolean
}

export default function SymbolDetection({ result, loading }: Props) {
  const { t } = useTranslation()

  if (!result && !loading) return null

  const tradeModeLabel = (mode: number): string => {
    switch (mode) {
      case 0: return t('symbolDetection.tradeMode.disabled')
      case 1: return t('symbolDetection.tradeMode.longOnly')
      case 2: return t('symbolDetection.tradeMode.shortOnly')
      case 3: return t('symbolDetection.tradeMode.longShort')
      default: return t('symbolDetection.tradeMode.unknown', { mode })
    }
  }

  return (
    <div style={{ padding: '12px 0' }}>
      <div style={{ marginBottom: 8, fontWeight: 600, fontSize: 14 }}>
        {t('symbolDetection.label')}
      </div>

      {loading && (
        <div style={{ color: '#888' }}>{t('symbolDetection.loading')}</div>
      )}

      {result && result.extracted.length === 0 && (
        <Space>
          <WarningOutlined style={{ color: '#faad14' }} />
          <span style={{ color: '#888' }}>{t('symbolDetection.noSymbols')}</span>
        </Space>
      )}

      {result && result.extracted.length > 0 && (
        <Space wrap size={[8, 8]}>
          {result.extracted.map((sym, i) => {
            const resolved = result.resolved?.find((r) => r.canonical === sym.canonical)
            return (
              <Tooltip
                key={i}
                title={
                  resolved
                    ? t('symbolDetection.resolvedTooltip', { broker: resolved.symbol_raw || '—', mode: tradeModeLabel(resolved.trade_mode) })
                    : t('symbolDetection.unresolvedTooltip')
                }
              >
                <Tag
                  color={resolved ? (resolved.is_tradeable ? 'green' : 'orange') : 'blue'}
                  icon={
                    resolved ? (
                      resolved.is_tradeable ? <CheckCircleOutlined /> : <CloseCircleOutlined />
                    ) : undefined
                  }
                  style={{ fontSize: 13, padding: '2px 8px' }}
                >
                  {sym.canonical}
                  {sym.confidence > 0 && sym.confidence < 1 && (
                    <span style={{ marginLeft: 4, opacity: 0.6, fontSize: 11 }}>
                      {Math.round(sym.confidence * 100)}%
                    </span>
                  )}
                </Tag>
              </Tooltip>
            )
          })}
        </Space>
      )}

      {result && result.warnings && result.warnings.length > 0 && (
        <div style={{ marginTop: 8 }}>
          {result.warnings.map((w, i) => (
            <div key={i} style={{ color: '#faad14', fontSize: 12 }}>
              <WarningOutlined style={{ marginRight: 4 }} />
              {w}
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
