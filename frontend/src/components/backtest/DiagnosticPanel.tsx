import { Alert, Tag, Button, Tooltip, Switch } from 'antd';
import { WarningOutlined, BugOutlined, InfoCircleOutlined, RobotOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { useMemo, useCallback } from 'react';
import {
  BACKTEST_DEGRADED_KEY, BACKTEST_DEGRADED_DESC_KEY,
} from '@/gen/ant/v1/i18n/strategy_workspace_keys';
import type { BacktestBlindSpotItem } from './backtestRunnerWatch';

const SILENCE_KEY_PREFIX = 'bt_silence_';

function loadSilencedSignatures(strategyId: string): Set<string> {
  try {
    const raw = localStorage.getItem(SILENCE_KEY_PREFIX + strategyId);
    return raw ? new Set(JSON.parse(raw)) : new Set();
  } catch {
    return new Set();
  }
}

function saveSilencedSignatures(strategyId: string, signatures: Set<string>): void {
  try {
    localStorage.setItem(SILENCE_KEY_PREFIX + strategyId, JSON.stringify([...signatures]));
  } catch { /* quota */ }
}

function severityLevel(severity: string): 'fatal' | 'warning' | 'info' {
  if (severity === '致命' || severity === 'fatal') return 'fatal';
  if (severity === '警告' || severity === 'warning') return 'warning';
  return 'info';
}

function severityColor(level: 'fatal' | 'warning' | 'info'): string {
  switch (level) {
    case 'fatal': return '#ff4d4f';
    case 'warning': return '#faad14';
    case 'info': return '#8c8c8c';
  }
}

function severityIcon(level: 'fatal' | 'warning' | 'info') {
  switch (level) {
    case 'fatal': return <BugOutlined style={{ color: severityColor(level) }} />;
    case 'warning': return <WarningOutlined style={{ color: severityColor(level) }} />;
    case 'info': return <InfoCircleOutlined style={{ color: severityColor(level) }} />;
  }
}

function categoryLabel(category: string, t: (k: string, d?: string) => string): string {
  switch (category) {
    case 'invariant': return t('strategy.backtest.diagnostic.invariant', 'Invariant Violation');
    case 'defense_a': return t('strategy.backtest.diagnostic.defenseA', 'Structural Validation');
    case 'lookahead': return t('strategy.backtest.diagnostic.lookahead', 'Lookahead Bias');
    case 'statistical': return t('strategy.backtest.diagnostic.statistical', 'Statistical Hint');
    default: return category || t('strategy.backtest.diagnostic.unknown', 'Diagnostic');
  }
}

interface DiagnosticPanelProps {
  blindSpots: BacktestBlindSpotItem[];
  strategyId?: string;
  onAIFix?: (blindSpots: BacktestBlindSpotItem[]) => void;
  aiFixing?: boolean;
}

export function DiagnosticPanel({ blindSpots, strategyId, onAIFix, aiFixing }: DiagnosticPanelProps) {
  const { t } = useTranslation();
  const silenced = useMemo(() => strategyId ? loadSilencedSignatures(strategyId) : new Set<string>(), [strategyId]);

  const grouped = useMemo(() => {
    const fatal: BacktestBlindSpotItem[] = [];
    const warning: BacktestBlindSpotItem[] = [];
    const info: BacktestBlindSpotItem[] = [];
    for (const bs of blindSpots) {
      const level = severityLevel(bs.severity);
      if (level === 'fatal') fatal.push(bs);
      else if (level === 'warning') warning.push(bs);
      else info.push(bs);
    }
    return { fatal, warning, info };
  }, [blindSpots]);

  const visibleWarning = useMemo(() => {
    if (!strategyId) return grouped.warning;
    return grouped.warning.filter(bs => !silenced.has(bs.id));
  }, [grouped.warning, silenced, strategyId]);

  const handleSilenceToggle = useCallback((bsId: string, checked: boolean) => {
    if (!strategyId) return;
    const current = loadSilencedSignatures(strategyId);
    if (checked) current.add(bsId);
    else current.delete(bsId);
    saveSilencedSignatures(strategyId, current);
  }, [strategyId]);

  const hasFatal = grouped.fatal.length > 0;
  const hasFixable = hasFatal || grouped.info.length > 0;
  const allSilenced = strategyId && grouped.warning.length > 0 && visibleWarning.length === 0;

  if (blindSpots.length === 0) return null;

  const alertType = hasFatal ? 'error' : 'warning';
  const alertIcon = hasFatal ? <BugOutlined /> : <WarningOutlined />;

  return (
    <Alert
      type={alertType}
      showIcon
      icon={alertIcon}
      message={t(BACKTEST_DEGRADED_KEY)}
      description={
        <div>
          <div style={{ marginBottom: 8 }}>{t(BACKTEST_DEGRADED_DESC_KEY)}</div>

          {grouped.fatal.length > 0 && (
            <DiagnosticGroup
              title={t('strategy.backtest.diagnostic.fatal', 'Critical Issues')}
              items={grouped.fatal}
              color={severityColor('fatal')}
              icon={severityIcon('fatal')}
              t={t}
            />
          )}

          {visibleWarning.length > 0 && (
            <DiagnosticGroup
              title={t('strategy.backtest.diagnostic.warning', 'Risk Warnings')}
              items={visibleWarning}
              color={severityColor('warning')}
              icon={severityIcon('warning')}
              t={t}
              renderItem={(bs) => (
                <div key={bs.id} style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: 4 }}>
                  <div style={{ flex: 1 }}>
                    <Tag style={{ fontSize: 10, marginRight: 4 }}>{categoryLabel(bs.category, t)}</Tag>
                    <span>{bs.description}</span>
                    {bs.location && <span style={{ color: '#8c8c8c', fontSize: 11, marginLeft: 4 }}>@ {bs.location}</span>}
                  </div>
                  {strategyId && (
                    <Tooltip title={t('strategy.backtest.diagnostic.silenceHint', 'Acknowledge as intentional — hide this warning')}>
                      <Switch
                        size="small"
                        onChange={(checked) => handleSilenceToggle(bs.id, checked)}
                      />
                    </Tooltip>
                  )}
                </div>
              )}
            />
          )}

          {allSilenced && (
            <div style={{ fontSize: 11, color: '#8c8c8c', marginBottom: 4 }}>
              {t('strategy.backtest.diagnostic.allSilenced', 'All warnings acknowledged as intentional')}
            </div>
          )}

          {grouped.info.length > 0 && (
            <DiagnosticGroup
              title={t('strategy.backtest.diagnostic.info', 'Quality Hints')}
              items={grouped.info}
              color={severityColor('info')}
              icon={severityIcon('info')}
              t={t}
            />
          )}

          {hasFixable && onAIFix && (
            <div style={{ marginTop: 8 }}>
              <Button
                size="small"
                type="dashed"
                icon={<RobotOutlined />}
                loading={aiFixing}
                onClick={() => onAIFix(grouped.fatal.length > 0 ? grouped.fatal : grouped.info)}
              >
                {t('strategy.backtest.diagnostic.aiFix', 'AI Fix')}
              </Button>
            </div>
          )}
        </div>
      }
      style={{ marginBottom: 12 }}
    />
  );
}

function DiagnosticGroup({
  title,
  items,
  color,
  icon,
  t,
  renderItem,
}: {
  title: string;
  items: BacktestBlindSpotItem[];
  color: string;
  icon: React.ReactNode;
  t: (k: string, d?: string) => string;
  renderItem?: (bs: BacktestBlindSpotItem) => React.ReactNode;
}) {
  if (items.length === 0) return null;
  return (
    <div style={{ marginBottom: 8 }}>
      <div style={{ fontWeight: 600, fontSize: 12, marginBottom: 4, color, display: 'flex', alignItems: 'center', gap: 4 }}>
        {icon}
        {title} ({items.length})
      </div>
      <ul style={{ margin: 0, paddingLeft: 20, fontSize: 12 }}>
        {items.map((bs) => renderItem ? renderItem(bs) : (
          <li key={bs.id} style={{ marginBottom: 2 }}>
            <Tag style={{ fontSize: 10, marginRight: 4 }}>{categoryLabel(bs.category, t)}</Tag>
            <span>{bs.description}</span>
            {bs.location && <span style={{ color: '#8c8c8c', fontSize: 11, marginLeft: 4 }}>@ {bs.location}</span>}
          </li>
        ))}
      </ul>
    </div>
  );
}
