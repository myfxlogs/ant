/**
 * Format a monetary amount for display — always shows at least 2 decimal places,
 * strips trailing zeros beyond the 2nd decimal.
 *
 * "1234" → "1234.00"
 * "1234.5" → "1234.50"
 * "1234.56" → "1234.56"
 * "1234.567" → "1234.567"
 * "1234.560" → "1234.56"
 */
export function formatAmount(value: string | number | null | undefined): string {
  const num = typeof value === 'string' ? parseFloat(value) : value;
  if (num == null || isNaN(num)) return '0.00';

  const parts = num.toFixed(10).split('.');
  const intPart = parts[0];
  let decPart = parts[1] || '';

  // Strip trailing zeros beyond position 2
  while (decPart.length > 2 && decPart.endsWith('0')) {
    decPart = decPart.slice(0, -1);
  }

  return `${intPart}.${decPart}`;
}
