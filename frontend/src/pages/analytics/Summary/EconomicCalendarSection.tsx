import { memo } from 'react';
import { Row, Col } from 'antd';
import { ResponsiveContainer, LineChart, Line } from 'recharts';
import { useTranslation } from 'react-i18next';
import type { EconomicCalendarEvent, EconomicIndicator } from '@/gen/ant/v1/economic_data_pb';

interface Props {
  calendarEvents: EconomicCalendarEvent[];
  keyIndicators: EconomicIndicator[];
}

function EconomicCalendarSection({ calendarEvents, keyIndicators }: Props) {
  const { t } = useTranslation();
  return (
    <div className="rounded-2xl p-6 mt-6" style={{ background: 'var(--color-bg-card)', boxShadow: '0 4px 24px rgba(0, 0, 0, 0.08)' }}>
      <h2 className="text-lg font-semibold mb-4" style={{ color: 'var(--color-text)' }}>{t('analytics.summary.cards.economicCalendar')}</h2>
      <Row gutter={16}>
        <Col xs={24} md={14}>
          {calendarEvents.length === 0 ? (
            <div style={{ color: 'var(--color-text-muted)' }}>{t('analytics.summary.economicCalendar.empty') || 'No economic events available.'}</div>
          ) : (
            <div className="space-y-2 max-h-64 overflow-auto mt-2">
              {calendarEvents.map((event, index) => {
                const key = `${event.timestamp || ''}-${event.event || ''}-${event.country || ''}-${index}`;
                const dtLabel = event.time ? `${event.date || ''} ${event.time}` : (event.date || '');
                return (
                  <div key={key} className="flex justify-between gap-3 text-sm py-1 border-b border-gray-100 last:border-b-0">
                    <div className="flex-1 min-w-0">
                      <div className="font-medium truncate" style={{ color: 'var(--color-text)' }}>{event.localizedEvent || event.event || '-'}</div>
                      <div className="text-xs mt-1" style={{ color: 'var(--color-text-muted)' }}>{dtLabel}{event.country ? ` · ${event.country}` : ''}{event.impact ? ` · ${event.impact}` : ''}</div>
                    </div>
                    <div className="text-right text-xs" style={{ color: 'var(--color-text-muted)', minWidth: '120px' }}>
                      {event.actual && <div>{t('analytics.summary.economicCalendar.actual') || 'Actual'}: {event.actual}</div>}
                      {event.previous && <div>{t('analytics.summary.economicCalendar.previous') || 'Previous'}: {event.previous}</div>}
                      {event.estimate && <div>{t('analytics.summary.economicCalendar.estimate') || 'Estimate'}: {event.estimate}</div>}
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </Col>
        <Col xs={24} md={10}>
          <div className="mb-2 text-sm font-medium" style={{ color: 'var(--color-text)' }}>{t('analytics.summary.economicCalendar.keyIndicatorsTitle') || 'Key macro indicators'}</div>
          {keyIndicators.length === 0 ? (
            <div style={{ color: 'var(--color-text-muted)' }}>{t('analytics.summary.economicCalendar.empty') || 'No economic events available.'}</div>
          ) : (
            <div className="space-y-3 max-h-64 overflow-auto mt-1">
              {keyIndicators.map((ind) => {
                const history = Array.isArray(ind.history) ? [...ind.history].reverse() : [];
                return (
                  <div key={ind.code} className="text-xs p-1.5 rounded-lg" style={{ backgroundColor: '#F7F9FB' }}>
                    <div className="flex items-center justify-between mb-1">
                      <div className="font-medium truncate" style={{ color: 'var(--color-text)' }}>
                        {t(`analytics.summary.economicCalendar.indicators.${ind.code}`, { defaultValue: ind.name || ind.code })}
                      </div>
                      <div style={{ color: 'var(--color-text)' }}>{ind.latestValue?.toFixed ? ind.latestValue.toFixed(2) : ind.latestValue}{ind.units ? ` ${ind.units}` : ''}</div>
                    </div>
                    {history.length > 1 && (
                      <div style={{ height: 40 }}>
                        <ResponsiveContainer width="100%" height="100%">
                          <LineChart data={history}><Line type="monotone" dataKey="value" stroke="#D4AF37" strokeWidth={1.5} dot={false} /></LineChart>
                        </ResponsiveContainer>
                      </div>
                    )}
                  </div>
                );
              })}
            </div>
          )}
        </Col>
      </Row>
    </div>
  );
}

export default memo(EconomicCalendarSection);
