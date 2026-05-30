import { Alert, Tooltip } from 'antd';
import { ExclamationCircleOutlined } from '@ant-design/icons';
import type React from 'react';

export function Section({
  step,
  title,
  subtitle,
  children,
}: {
  step?: number;
  title: string;
  subtitle?: React.ReactNode;
  children: React.ReactNode;
}) {
  return (
    <section className="bg-white rounded-xl shadow-sm border border-gray-100">
      <header className="flex items-start justify-between gap-4 px-6 py-4">
        <div className="flex items-start gap-3">
          {typeof step === 'number' && (
            <span
              className="w-7 h-7 rounded-full text-sm font-semibold flex items-center justify-center shrink-0"
              style={{ backgroundColor: '#D4AF37', border: '1px solid #D4AF37', color: '#FFFFFF' }}
            >
              {step}
            </span>
          )}
          <div>
            <h2 className="text-base font-semibold text-gray-900">{title}</h2>
            {subtitle && <div className="text-xs text-gray-500 mt-0.5">{subtitle}</div>}
          </div>
        </div>
      </header>
      <div className="px-6 pb-6 pt-2">{children}</div>
    </section>
  );
}

export function Label({
  text,
  hint,
  badge,
}: {
  text: string;
  hint?: string;
  badge?: React.ReactNode;
}) {
  return (
    <div className="flex items-center justify-between mb-1.5">
      <div className="flex items-center gap-1.5">
        <span className="text-sm font-medium text-gray-700">{text}</span>
        {hint && (
          <Tooltip title={hint}>
            <ExclamationCircleOutlined className="w-3.5 h-3.5 text-gray-400 cursor-help" />
          </Tooltip>
        )}
      </div>
      {badge}
    </div>
  );
}

export function SoftTag({ children }: { children: React.ReactNode }) {
  return (
    <span
      className="inline-flex items-center rounded px-1.5 py-0.5 text-xs"
      style={{
        backgroundColor: 'rgba(212, 175, 55, 0.10)',
        border: '1px solid rgba(212, 175, 55, 0.32)',
        color: '#B8960B',
      }}
    >
      {children}
    </span>
  );
}

export function StatusBanner({
  tone: _tone,
  title,
  description,
  notice,
}: {
  tone: 'success' | 'warning' | 'error' | 'info';
  title: string;
  description: string;
  notice?: string;
}) {
  return (
    <Alert
      type="info"
      showIcon
      className="ai-gold-alert"
      message={<span className="font-semibold">{title}</span>}
      description={
        <div className="space-y-1">
          <div>{description}</div>
          {notice && <div className="text-xs text-slate-600">{notice}</div>}
        </div>
      }
    />
  );
}
