import { Helmet } from 'react-helmet-async';

interface SeoProps {
  title?: string;
  description?: string;
  path?: string;
  noindex?: boolean;
  keywords?: string[];
  ogImage?: string;
  ogType?: string;
  twitterCard?: 'summary' | 'summary_large_image';
}

const SITE_NAME = 'AlphaForge';
const BASE_URL = 'https://alfq.org';
const DEFAULT_DESCRIPTION = 'AI-powered platform for MT4/MT5 strategy backtesting, optimization, and automated trading.';
const DEFAULT_OG_IMAGE = '/og-image.svg';

export default function Seo({
  title,
  description,
  path,
  noindex,
  keywords,
  ogImage,
  ogType = 'website',
  twitterCard = 'summary_large_image',
}: SeoProps) {
  const fullTitle = title ? `${title} — ${SITE_NAME}` : `${SITE_NAME} — AI-Powered MT4/MT5 Strategy Platform`;
  const desc = description || DEFAULT_DESCRIPTION;
  const url = path ? `${BASE_URL}${path}` : BASE_URL;
  const rawImage = ogImage || DEFAULT_OG_IMAGE;
  const image = rawImage.startsWith('http') ? rawImage : `${BASE_URL}${rawImage}`;

  return (
    <Helmet>
      <title>{fullTitle}</title>
      <meta name="description" content={desc} />
      {keywords && keywords.length > 0 && (
        <meta name="keywords" content={keywords.join(', ')} />
      )}
      <link rel="canonical" href={url} />
      {noindex && <meta name="robots" content="noindex, nofollow" />}
      <meta property="og:type" content={ogType} />
      <meta property="og:site_name" content={SITE_NAME} />
      <meta property="og:title" content={fullTitle} />
      <meta property="og:description" content={desc} />
      <meta property="og:url" content={url} />
      <meta property="og:image" content={image} />
      <meta property="og:image:width" content="1200" />
      <meta property="og:image:height" content="630" />
      <meta property="og:image:alt" content={fullTitle} />
      <meta name="twitter:card" content={twitterCard} />
      <meta name="twitter:title" content={fullTitle} />
      <meta name="twitter:description" content={desc} />
      <meta name="twitter:image" content={image} />
    </Helmet>
  );
}
