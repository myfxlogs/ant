import { useState, useEffect, useCallback } from 'react';
import { ThumbsUp, ThumbsDown, MessageSquare } from 'lucide-react';
import { aiApi } from '../../client/ai';
import { useTranslation } from 'react-i18next';

interface SessionFeedbackProps {
  sessionId: string;
}

export function SessionFeedback({ sessionId }: SessionFeedbackProps) {
  const { t } = useTranslation();
  const [rating, setRating] = useState<'good' | 'bad' | null>(null);
  const [showReason, setShowReason] = useState(false);
  const [reason, setReason] = useState('');
  const [submitting, setSubmitting] = useState(false);

  const loadExisting = useCallback(async () => {
    if (!sessionId) return;
    try {
      const existing = await aiApi.getSessionFeedback(sessionId);
      if (existing) {
        setRating(existing.rating as 'good' | 'bad');
        setReason(existing.reason);
      }
    } catch {
      // ignore — not critical
    }
  }, [sessionId]);

  useEffect(() => {
    loadExisting();
  }, [loadExisting]);

  const submit = async (newRating: 'good' | 'bad') => {
    setRating(newRating);
    setShowReason(true);
    setSubmitting(true);
    try {
      await aiApi.submitSessionFeedback(sessionId, newRating, reason);
    } catch {
      // revert on failure
      setRating(null);
    } finally {
      setSubmitting(false);
    }
  };

  const submitReason = async () => {
    setSubmitting(true);
    try {
      await aiApi.submitSessionFeedback(sessionId, rating!, reason);
    } finally {
      setSubmitting(false);
      setShowReason(false);
    }
  };

  if (!sessionId) return null;

  return (
    <div className="flex flex-col gap-2 px-3 py-2 border-t border-gray-200 dark:border-gray-700">
      <div className="flex items-center gap-2 text-sm text-gray-500 dark:text-gray-400">
        <span>{t('feedback.rateSession')}</span>
        <button
          onClick={() => submit('good')}
          disabled={submitting}
          className={`p-1.5 rounded transition-colors ${
            rating === 'good'
              ? 'bg-green-100 text-green-600 dark:bg-green-900/30 dark:text-green-400'
              : 'hover:bg-gray-100 dark:hover:bg-gray-700'
          }`}
          aria-label={t('feedback.good')}
        >
          <ThumbsUp size={16} />
        </button>
        <button
          onClick={() => submit('bad')}
          disabled={submitting}
          className={`p-1.5 rounded transition-colors ${
            rating === 'bad'
              ? 'bg-red-100 text-red-600 dark:bg-red-900/30 dark:text-red-400'
              : 'hover:bg-gray-100 dark:hover:bg-gray-700'
          }`}
          aria-label={t('feedback.bad')}
        >
          <ThumbsDown size={16} />
        </button>
        {rating && !showReason && (
          <button
            onClick={() => setShowReason(true)}
            className="ml-1 text-xs text-blue-500 hover:underline flex items-center gap-1"
          >
            <MessageSquare size={12} />
            {t('feedback.addReason')}
          </button>
        )}
      </div>
      {showReason && (
        <div className="flex gap-2">
          <input
            type="text"
            value={reason}
            onChange={(e) => setReason(e.target.value)}
            placeholder={t('feedback.reasonPlaceholder')}
            className="flex-1 px-2 py-1 text-sm border rounded dark:bg-gray-800 dark:border-gray-600 dark:text-gray-200"
            maxLength={500}
          />
          <button
            onClick={submitReason}
            disabled={submitting}
            className="px-3 py-1 text-sm bg-blue-500 text-white rounded hover:bg-blue-600 disabled:opacity-50"
          >
            {t('common.submit')}
          </button>
        </div>
      )}
    </div>
  );
}
