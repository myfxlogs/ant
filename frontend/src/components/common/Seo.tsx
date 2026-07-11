import { Helmet } from 'react-helmet-async';

interface SeoProps {
  title?: string;
  description?: string;
  path?: string;
  noindex?: boolean;
}

const SITE_NAME = 'AlphaForge';
const BASE_URL = 'https://alfq.org';
const DEFAULT_DESCRIPTION = 'AI-powered platform for MT4/MT5 strategy backtesting, optimization, and automated trading.';

export default function Seo({ title, description, path, noindex }: SeoProps) {
  const fullTitle = title ? `${title} — ${SITE_NAME}` : `${SITE_NAME} — AI-Powered MT4/MT5 Strategy Platform`;
  const desc = description || DEFAULT_DESCRIPTION;
  const url = path ? `${BASE_URL}${path}` : BASE_URL;

  return (
    <Helmet>
      <title>{fullTitle}</title>
      <meta name="description" content={desc} />
      <link rel="canonical" href={url} />
      {noindex && <meta name="robots" content="noindex, nofollow" />}
      <meta property="og:title" content={fullTitle} />
      <meta property="og:description" content={desc} />
      <meta property="og:url" content={url} />
      <meta name="twitter:title" content={fullTitle} />
      <meta name="twitter:description" content={desc} />
    </Helmet>
  );
}
