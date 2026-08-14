import { fnv1a32 } from './fnv1a32';

export function strategyMagic(scheduleId: string): number {
  if (!scheduleId) return 0;
  const bytes = new Uint8Array(16);
  const hex = scheduleId.replace(/-/g, '');
  for (let i = 0; i < 16; i++) {
    bytes[i] = parseInt(hex.slice(i * 2, i * 2 + 2), 16);
  }
  return fnv1a32(bytes) | 0;
}
