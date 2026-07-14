import * as Sentry from '@sentry/react'

const SENTRY_DSN = import.meta.env.VITE_SENTRY_DSN || ''

export function initSentry() {
  if (!SENTRY_DSN) {
    return
  }

  Sentry.init({
    dsn: SENTRY_DSN,
    environment: import.meta.env.VITE_SENTRY_ENVIRONMENT || import.meta.env.MODE || 'development',
    release: import.meta.env.VITE_SENTRY_RELEASE || undefined,
    tracesSampleRate: 0,
    attachStacktrace: true,
  })
}
