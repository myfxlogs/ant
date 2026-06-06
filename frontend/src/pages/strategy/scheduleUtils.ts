import type { Timestamp } from "@bufbuild/protobuf/wkt";
import { timestampDate } from "@bufbuild/protobuf/wkt";
import { getDeviceLocale, getDeviceTimeZone } from "@/utils/date";

interface WithSymbol { symbol?: unknown; }

export function buildSymbolOptions(list: WithSymbol[]) {
  return Array.from(new Set((list || []).map((s) => String(s?.symbol || "").trim()).filter(Boolean)))
    .map((value) => ({ value, label: value }));
}

export function formatTime(v: unknown): string {
  if (!v) return "-";
  const locale = getDeviceLocale(); const timeZone = getDeviceTimeZone();
  if (typeof v === "object") {
    const ts = v as Partial<Timestamp>;
    const seconds = ts.seconds;
    const secNum = typeof seconds === "number" ? seconds : typeof seconds === "bigint" ? Number(seconds) : undefined;
    if (typeof secNum === "number" && Number.isFinite(secNum)) {
      try { const d = timestampDate(v as Timestamp); if (d instanceof Date && !Number.isNaN(d.getTime())) return d.toLocaleString(locale, { timeZone, hour12: false }); }
      catch { /* ignore */ }
    }
  }
  if (v instanceof Date) return v.toLocaleString(locale, { timeZone, hour12: false });
  const s = String(v); const d = new Date(s);
  if (!Number.isNaN(d.getTime())) return d.toLocaleString(locale, { timeZone, hour12: false });
  return s;
}
