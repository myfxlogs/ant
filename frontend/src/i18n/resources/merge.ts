export function mergeResources<T extends Record<string, unknown>>(...items: T[]): T {
  const out: Record<string, unknown> = {};
  for (const item of items) mergeInto(out, item);
  return out as T;
}

function mergeInto(target: Record<string, unknown>, source: Record<string, unknown>) {
  for (const [key, value] of Object.entries(source)) {
    const current = target[key];
    if (isRecord(current) && isRecord(value)) {
      mergeInto(current, value);
      continue;
    }
    target[key] = value;
  }
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return !!value && typeof value === 'object' && !Array.isArray(value);
}
