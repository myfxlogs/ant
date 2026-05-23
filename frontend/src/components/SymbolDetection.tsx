import { Tag, Space, Tooltip } from 'antd'
import { CheckCircleOutlined, WarningOutlined, CloseCircleOutlined } from '@ant-design/icons'
import type { ValidateResult, ResolvedSymbol } from '@/types/symbol'

interface Props {
  result: ValidateResult | null
  loading?: boolean
}

export default function SymbolDetection({ result, loading }: Props) {
  if (!result && !loading) return null

  return (
    <div style={{ padding: '12px 0' }}>
      <div style={{ marginBottom: 8, fontWeight: 600, fontSize: 14 }}>
        识别到的交易品种
      </div>

      {loading && (
        <div style={{ color: '#888' }}>正在解析…</div>
      )}

      {result && result.extracted.length === 0 && (
        <Space>
          <WarningOutlined style={{ color: '#faad14' }} />
          <span style={{ color: '#888' }}>未识别到交易品种，请尝试包含具体的品种名称（如"比特币"、"EURUSD"、"黄金"）</span>
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
                    ? `broker: ${resolved.symbol_raw || '—'} | 模式: ${tradeModeLabel(resolved.trade_mode)}`
                    : '尚未绑定交易账户，无法解析'
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

function tradeModeLabel(mode: number): string {
  switch (mode) {
    case 0: return '已禁用'
    case 1: return '仅做多'
    case 2: return '仅做空'
    case 3: return '多空均可'
    default: return `未知(${mode})`
  }
}
