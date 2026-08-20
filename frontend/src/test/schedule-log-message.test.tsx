import { describe, expect, it } from 'vitest';
import type { ReactElement, ReactNode } from 'react';
import type { TFunction } from 'i18next';
import type { ScheduleRunLog } from '@/gen/ant/v1/log_schedule_pb';
import { buildLogColumns } from '@/pages/strategy/components/live/expandedRowColumns';

describe('schedule execution log message column', () => {
  const messageColumn = () => buildLogColumns(((key: string) => key) as TFunction)[2] as {
    render: (value: unknown, row: ScheduleRunLog) => ReactNode;
  };

  it('shows and makes error_message copyable', () => {
    const row = { errorMessage: 'authoritative account snapshot unavailable or stale: account=acct-1' } as ScheduleRunLog;
    const element = messageColumn().render(undefined, row) as ReactElement<{ copyable?: { text?: string }; children?: ReactNode }>;
    expect(element.props.children).toBe(row.errorMessage);
    expect(element.props.copyable?.text).toBe(row.errorMessage);
  });

  it('shows lifecycle context when error_message is empty', () => {
    const row = { kind: 'signal', action: 'received', signalType: 'buy' } as ScheduleRunLog;
    const element = messageColumn().render(undefined, row) as ReactElement<{ children?: ReactNode }>;
    expect(element.props.children).toBe('signal / received / buy');
  });
});
