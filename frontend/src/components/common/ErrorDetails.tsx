import { useEffect, useMemo, useState } from 'react';
import { Button, message } from 'antd';
import { useTranslation } from 'react-i18next';
import { translateTextWithLLM } from '@/utils/llmTranslate';

export interface ErrorDetailsProps {
  detail?: string;
}

export default function ErrorDetails(props: ErrorDetailsProps) {
  const { t } = useTranslation();
  const detail = String(props.detail || '').trim();

  const [detailVisible, setDetailVisible] = useState(false);
  const [translatingDetail, setTranslatingDetail] = useState(false);
  const [translatedDetail, setTranslatedDetail] = useState('');
  const [showOriginalDetail, setShowOriginalDetail] = useState(false);

  const displayText = useMemo(() => {
    if (!translatedDetail) return detail;
    if (showOriginalDetail) return detail;
    return translatedDetail;
  }, [detail, showOriginalDetail, translatedDetail]);

  useEffect(() => {
    setDetailVisible(false);
    setTranslatingDetail(false);
    setTranslatedDetail('');
    setShowOriginalDetail(false);
  }, [detail]);

  if (!detail) return null;

  return (
    <div className="mb-4 text-left">
      <div className="flex items-center justify-between">
        <Button type="link" onClick={() => setDetailVisible((v) => !v)} style={{ padding: 0 }}>
          {detailVisible ? t('common.hideDetails') : t('common.showDetails')}
        </Button>

        {detailVisible ? (
          <div className="flex items-center gap-2">
            <Button
              size="small"
              onClick={async () => {
                try {
                  await navigator.clipboard.writeText(displayText);
                  message.success(t('common.copied'));
                } catch {
                  message.error(t('common.copyFailed'));
                }
              }}
            >
              {t('common.copy')}
            </Button>

            <Button
              size="small"
              loading={translatingDetail}
              onClick={async () => {
                if (translatedDetail) {
                  setShowOriginalDetail(false);
                  return;
                }
                setTranslatingDetail(true);
                try {
                  const out = await translateTextWithLLM({ text: detail, purpose: 'error_detail' });
                  setTranslatedDetail(out);
                  setShowOriginalDetail(false);
                } catch (e: any) {
                  const msg = String(e?.message || e || '').trim();
                  if (msg === 'errors.ai.not_configured') {
                    message.warning(t('errors.ai.not_configured'));
                  } else {
                    message.error(t('errors.translate_failed'));
                  }
                } finally {
                  setTranslatingDetail(false);
                }
              }}
            >
              {translatedDetail ? t('common.viewTranslation') : t('common.translate')}
            </Button>

            {translatedDetail ? (
              <Button size="small" onClick={() => setShowOriginalDetail((v) => !v)}>
                {showOriginalDetail ? t('common.viewTranslation') : t('common.viewOriginal')}
              </Button>
            ) : null}
          </div>
        ) : null}
      </div>

      {detailVisible ? (
        <pre
          className="text-xs whitespace-pre-wrap mt-2"
          style={{
            background: '#F5F7F9',
            border: '1px solid rgba(185, 201, 223, 0.4)',
            borderRadius: '10px',
            padding: '10px 12px',
            color: '#5A6B75',
            maxHeight: 160,
            overflow: 'auto',
          }}
        >
          {displayText}
        </pre>
      ) : null}
    </div>
  );
}
