import { Progress, Tag, Typography, Collapse } from 'antd';
import { CheckCircleOutlined, WarningOutlined, CloseCircleOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';

const { Text } = Typography;

interface CoverageData {
  compiles: boolean;
  coverage_score: number;
  total_calls: number;
  supported_calls: number;
  exec_kind: string;
  version: string;
  entry_rules: number;
  exit_rules: number;
  indicators: string[];
  params: { name: string; type: string; default: string }[];
  fatal_blind_spots: string[];
  warn_blind_spots: string[];
  recommendation: string;
  error?: string;
}

export default function CoverageReportView({ json }: { json: Record<string, unknown> }) {
  const { t } = useTranslation();
  const data = json as unknown as CoverageData;

  if (!data.compiles) {
    return (
      <div style={{ marginTop: 6, padding: '6px 8px', background: '#fff2f0', borderRadius: 4, fontSize: 10 }}>
        <CloseCircleOutlined style={{ color: '#ff4d4f' }} /> <b>{t('strategy.validate.compilationFailed', { defaultValue: 'Compilation failed' })}</b>
        {data.error && <div style={{ color: '#cf1322', marginTop: 2 }}>{data.error}</div>}
        {data.recommendation && <div style={{ color: '#8c8c8c', marginTop: 2 }}>{data.recommendation}</div>}
      </div>
    );
  }

  const scorePct = Math.round((data.coverage_score ?? 0) * 100);
  const scoreColor = scorePct >= 90 ? '#52c41a' : scorePct >= 70 ? '#faad14' : '#ff4d4f';
  const recColor = data.recommendation === 'ready_to_run' ? 'success'
    : data.recommendation === 'needs_ai_translation' ? 'error' : 'warning';
  const recLabel = data.recommendation === 'ready_to_run' ? 'Ready to Run'
    : data.recommendation === 'needs_ai_translation' ? 'Needs AI Translation' : 'Needs Review';

  return (
    <div style={{ marginTop: 6 }}>
      {/* Coverage score + recommendation */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 4 }}>
        <Progress
          percent={scorePct}
          size="small"
          strokeColor={scoreColor}
          style={{ width: 120, margin: 0 }}
          format={(pct) => <span style={{ fontSize: 10 }}>{pct}%</span>}
        />
        <Tag color={recColor} style={{ fontSize: 10, margin: 0 }}>{recLabel}</Tag>
        <Text type="secondary" style={{ fontSize: 10 }}>
          {data.supported_calls}/{data.total_calls} calls · {data.version} · {data.exec_kind}
        </Text>
      </div>

      {/* Fatal blind spots */}
      {data.fatal_blind_spots?.length > 0 && (
        <div style={{ marginBottom: 4 }}>
          {data.fatal_blind_spots.map((bs, i) => (
            <Tag key={i} color="error" icon={<CloseCircleOutlined />} style={{ fontSize: 10, margin: 2 }}>
              {bs}
            </Tag>
          ))}
        </div>
      )}

      {/* Warning blind spots */}
      {data.warn_blind_spots?.length > 0 && (
        <div style={{ marginBottom: 4 }}>
          {data.warn_blind_spots.map((bs, i) => (
            <Tag key={i} color="warning" icon={<WarningOutlined />} style={{ fontSize: 10, margin: 2 }}>
              {bs}
            </Tag>
          ))}
        </div>
      )}

      {/* Indicators */}
      {data.indicators?.length > 0 && (
        <div style={{ marginBottom: 4 }}>
          <Text type="secondary" style={{ fontSize: 10 }}>{t('strategy.chat.indicators', { defaultValue: 'Indicators:' })} </Text>
          {data.indicators.map((ind, i) => (
            <Tag key={i} style={{ fontSize: 10, margin: 1 }}>{ind}</Tag>
          ))}
        </div>
      )}

      {/* Parameters */}
      {data.params?.length > 0 && (
        <Collapse
          size="small"
          ghost
          style={{ marginTop: 2 }}
          items={[{
            key: 'params',
            label: <Text type="secondary" style={{ fontSize: 10 }}>
              <CheckCircleOutlined style={{ color: '#52c41a' }} /> {data.params.length} parameters
            </Text>,
            children: (
              <div style={{ display: 'flex', flexWrap: 'wrap', gap: 4 }}>
                {data.params.map((p, i) => (
                  <Tag key={i} style={{ fontSize: 10, margin: 1 }}>
                    {p.name}: {p.type}{p.default && ` = ${p.default}`}
                  </Tag>
                ))}
              </div>
            ),
          }]}
        />
      )}

      {/* Entry/exit rules */}
      <Text type="secondary" style={{ fontSize: 10 }}>
        Entry rules: {data.entry_rules} · Exit rules: {data.exit_rules}
      </Text>
    </div>
  );
}
