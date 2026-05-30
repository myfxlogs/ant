import { useCallback, useEffect, useMemo, useState } from "react";
import { Button, Card, Form, Space, Typography, message } from "antd";
import { StatusResult } from "@/components/common/StatusResult";
import type { Timestamp } from "@bufbuild/protobuf/wkt";
import { timestampDate } from "@bufbuild/protobuf/wkt";
import { pythonStrategyApi } from "../../client/pythonStrategy";
import {
  strategyScheduleV2Api,
  strategyTemplateApi,
} from "../../client/strategy";
import { useAccountsAndSymbols } from "./hooks/useAccountsAndSymbols";
import { tradingApi } from "../../client/trading";
import { scheduleHealthApi } from "../../client/scheduleHealth";
import { getTradingRiskToastMessage } from "../../utils/tradingRiskError";
import EditScheduleModal, {
  type ScheduleFormValues,
} from "./components/EditScheduleModal";
import TriggerModal from "./components/TriggerModal";
import ScheduleHealthModal from "./components/ScheduleHealthModal";
import ScheduleTable from "./components/ScheduleTable";
import { DEFAULT_TEMPLATES } from "./StrategyTemplatePage.defaults";
import { getDeviceLocale, getDeviceTimeZone } from "@/utils/date";
import {
  buildParametersFromForm,
  parseParametersToForm,
} from "./StrategyScheduleParams";
import { useTranslation } from "react-i18next";
const { Title } = Typography;

// ── Local helper types ──────────────────────────────────────────────────────

/** Signal-shaped object returned from Python strategy execution. */
interface SignalLike {
  type?: unknown;
  signalType?: unknown;
  signal?: unknown;
  volume?: unknown;
  price?: unknown;
  stopLoss?: unknown;
  takeProfit?: unknown;
  comment?: unknown;
  magicNumber?: unknown;
  symbol?: string;
}

/** Record with optional id field (created schedules, etc.). */
interface WithId {
  id?: unknown;
}

/** Record with optional symbol field (symbol lists). */
interface WithSymbol {
  symbol?: unknown;
}

/** Template with optional code field. */
interface WithCode {
  code?: unknown;
}

type ScheduleType = 'interval' | 'kline_close' | 'hf_quote';

function buildSymbolOptions(list: WithSymbol[]) {
  return Array.from(
    new Set((list || []).map((s) => String(s?.symbol || "").trim()).filter(Boolean)),
  ).map((value) => ({ value, label: value }));
}

export default function StrategySchedulePage() {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [schedules, setSchedules] = useState<any[]>([]);
  const [templates, setTemplates] = useState<any[]>([]);
  const { accounts, symbols, symbolsLoading, fetchAccounts, loadSymbols } =
    useAccountsAndSymbols();

  const [openEdit, setOpenEdit] = useState(false);
  const [editing, setEditing] = useState<any | null>(null);

  const [triggering, setTriggering] = useState(false);
  const [openTrigger, setOpenTrigger] = useState(false);
  const [triggerResult, setTriggerResult] = useState<{
    logs: string[];
    signal: any;
    meta: any;
  } | null>(null);
  const [triggerContext, setTriggerContext] = useState<{
    schedule: any;
    accountId: string;
  } | null>(null);
  const [healthOpen, setHealthOpen] = useState(false);
  const [healthLoading, setHealthLoading] = useState(false);
  const [healthTarget, setHealthTarget] = useState<any | null>(null);
  const [healthSummary, setHealthSummary] = useState<any | null>(null);

  const [form] = Form.useForm<ScheduleFormValues>();

  const formatTime = (v: unknown) => {
    if (!v) return "-";
    const locale = getDeviceLocale();
    const timeZone = getDeviceTimeZone();
    if (typeof v === "object") {
      // Check for @bufbuild/protobuf Timestamp-like ({ seconds, nanos })
      const ts = v as Partial<Timestamp>;
      const seconds = ts.seconds;
      const secNum =
        typeof seconds === "number"
          ? seconds
          : typeof seconds === "bigint"
            ? Number(seconds)
            : undefined;
      if (typeof secNum === "number" && Number.isFinite(secNum)) {
        try {
          const d = timestampDate(v as Timestamp);
          if (d instanceof Date && !Number.isNaN(d.getTime())) {
            return d.toLocaleString(locale, { timeZone, hour12: false });
          }
        } catch (_e) {
          // ignore
        }
      }
    }
    if (v instanceof Date) {
      return v.toLocaleString(locale, { timeZone, hour12: false });
    }
    const s = String(v);
    const d = new Date(s);
    if (!Number.isNaN(d.getTime())) {
      return d.toLocaleString(locale, { timeZone, hour12: false });
    }
    return s;
  };

  const loadScheduleHealth = useCallback(
    async (row: any) => {
      if (!row?.id) return;
      setHealthLoading(true);
      try {
        setHealthSummary(await scheduleHealthApi.getScheduleHealth(row.id));
      } catch (e: any) {
        message.error(
          e?.message || t("strategy.schedules.health.messages.loadFailed"),
        );
        setHealthSummary(null);
      } finally {
        setHealthLoading(false);
      }
    },
    [t],
  );

  const refresh = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [tpls, schs] = await Promise.all([
        strategyTemplateApi.list(),
        strategyScheduleV2Api.list(),
      ]);
      setTemplates(tpls as Record<string, unknown>[]);
      setSchedules(schs as Record<string, unknown>[]);
      void fetchAccounts();
    } catch (e: any) {
      const msg = e?.message || t("common.loadingFailed");
      setError(msg);
      message.error(msg);
    } finally {
      setLoading(false);
    }
  }, [t]);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  useEffect(() => {
    const t = setInterval(() => {
      void refresh();
    }, 10_000);
    return () => clearInterval(t);
  }, [refresh]);

  const templatesForSelect = useMemo(() => {
    const out: any[] = [];
    const seen = new Set<string>();
    (templates || []).forEach((t: any) => {
      if (!t?.id) return;
      seen.add(String(t.id));
      out.push(t);
    });
    (DEFAULT_TEMPLATES || []).forEach((t: any) => {
      if (!t?.id) return;
      const id = String(t.id);
      if (seen.has(id)) return;
      out.push(t);
    });
    return out;
  }, [templates]);

  const loadSymbols = useCallback(
    async (accountId: string, keepSymbol?: string) => {
      if (!accountId) {
        setSymbols([]);
        form.setFieldValue("symbol", "");
        return;
      }

      setSymbolsLoading(true);
      setSymbols([]);
      if (!keepSymbol) {
        form.setFieldValue("symbol", "");
      }
      try {
        const list = await marketApi.getSymbols(accountId);
        const opts = buildSymbolOptions(list);
        setSymbols(opts);

        const nextSymbol = keepSymbol || form.getFieldValue("symbol");
        const exists = opts.some((o) => o.value === nextSymbol);
        if (opts.length > 0 && (!nextSymbol || !exists)) {
          form.setFieldValue("symbol", opts[0].value);
        }
      } catch (_e) {
        setSymbols([]);
        form.setFieldValue("symbol", "");
      } finally {
        setSymbolsLoading(false);
      }
    },
    [form],
  );

  const openCreate = useCallback(() => {
    setEditing(null);
    form.resetFields();
    form.setFieldsValue({
      isActive: true,
      timeframe: "H1",
      symbol: "",
      scheduleType: "kline_close",
      intervalMs: 300_000,
      hfCooldownMs: 1_000,
      parametersJson: "{}",
    });
    setSymbols([]);
    setOpenEdit(true);
  }, [form]);

  useEffect(() => {
    const qs = new URLSearchParams(window.location.search || "");
    const accountId = String(qs.get("accountId") || "").trim();
    const symbol = String(qs.get("symbol") || "").trim();
    const timeframe = String(qs.get("timeframe") || "").trim();
    if (!accountId && !symbol && !timeframe) return;
    openCreate();
    if (accountId) form.setFieldValue("accountId", accountId);
    if (timeframe) form.setFieldValue("timeframe", timeframe);
    if (accountId) void loadSymbols(accountId, symbol);
    if (symbol) form.setFieldValue("symbol", symbol);
  }, [form, openCreate, loadSymbols]);

  const openUpdate = (row: any) => {
    setEditing(row);
    const conf = row?.scheduleConfig || {};
    // 归一化后端 scheduleType → 前端三选一：interval | kline_close | hf_quote。
    const rawType = String(row?.scheduleType || "").toLowerCase();
    const triggerMode = String(conf?.triggerMode || "stable_kline");
    let scheduleType: ScheduleType;
    if (rawType === "interval" || rawType === "cron") {
      scheduleType = "interval";
    } else if (triggerMode === "hf_quote_stream" || rawType === "hf_quote") {
      scheduleType = "hf_quote";
    } else {
      scheduleType = "kline_close";
    }
    const intervalMs =
      typeof conf?.intervalMs === "number"
        ? conf.intervalMs
        : typeof conf?.intervalMs === "bigint"
          ? Number(conf.intervalMs)
          : 300_000;
    const hfCooldownMs =
      typeof conf?.hfCooldownMs === "number"
        ? conf.hfCooldownMs
        : typeof conf?.hfCooldownMs === "bigint"
          ? Number(conf.hfCooldownMs)
          : 1_000;
    const parametersJson = row?.parameters
      ? JSON.stringify(row.parameters, null, 2)
      : "{}";
    const parsedParams = parseParametersToForm(row?.parameters || {});
    form.setFieldsValue({
      id: row?.id,
      templateId: row?.templateId,
      accountId: row?.accountId,
      name: row?.name,
      symbol: row?.symbol,
      timeframe: row?.timeframe,
      defaultVolume: parsedParams.defaultVolume,
      maxPositions: parsedParams.maxPositions,
      stopLossPriceOffset: parsedParams.stopLossPriceOffset,
      takeProfitPriceOffset: parsedParams.takeProfitPriceOffset,
      maxDrawdownPct: parsedParams.maxDrawdownPct,
      scheduleType,
      intervalMs,
      hfCooldownMs,
      parametersJson,
    });
    void loadSymbols(row?.accountId, row?.symbol);
    setOpenEdit(true);
  };

  const parseParameters = (raw?: string): Record<string, string> => {
    if (!raw || raw.trim() === "") return {};
    const obj = JSON.parse(raw);
    if (obj == null || typeof obj !== "object" || Array.isArray(obj)) {
      throw new Error(
        t("strategy.schedules.validation.parametersMustBeJsonObject"),
      );
    }
    const out: Record<string, string> = {};
    Object.entries(obj).forEach(([k, v]) => {
      out[String(k)] = typeof v === "string" ? v : JSON.stringify(v);
    });
    return out;
  };

  const submitEdit = async () => {
    const v = await form.validateFields();
    let params: Record<string, string> = {};
    try {
      params = parseParameters(v.parametersJson);
    } catch (e: any) {
      message.error(
        e?.message || t("strategy.schedules.messages.parametersParseFailed"),
      );
      return;
    }

    // Merge JSON params with structured fields to keep behaviour consistent
    // with the schedule-launch form (risk keys + lot etc.). Structured fields win.
    const merged = {
      ...params,
      ...buildParametersFromForm(v),
    };

    // 构建后端 scheduleConfig：
    //   - interval     → intervalMs 有效，triggerMode=stable_kline
    //   - kline_close  → intervalMs=0, triggerMode=stable_kline
    //   - hf_quote     → hfCooldownMs 有效, triggerMode=hf_quote_stream
    const sType: ScheduleType = (v.scheduleType ||
      "kline_close") as ScheduleType;
    const scheduleConfig: Record<string, unknown> = {
      cronExpression: "",
      intervalMs: 0n,
      eventTrigger: "",
      triggerMode: sType === "hf_quote" ? "hf_quote_stream" : "stable_kline",
      stableOverrideIntervalMs: 0n,
      hfCooldownMs: 0n,
    };
    if (sType === "interval") {
      const ms = Math.max(1000, Math.floor(Number(v.intervalMs || 300_000)));
      scheduleConfig.intervalMs = BigInt(ms);
    }
    if (sType === "hf_quote") {
      const cd = Math.max(100, Math.floor(Number(v.hfCooldownMs || 1_000)));
      scheduleConfig.hfCooldownMs = BigInt(cd);
    }
    // 后端存储 scheduleType 仍沿用 interval/cron 两种兼容值（kline_close 与 hf_quote 走 cron 分支）。
    const backendScheduleType = sType === "interval" ? "interval" : "cron";

    setLoading(true);
    try {
      if (editing?.id) {
        await strategyScheduleV2Api.update({
          id: editing.id,
          name: v.name,
          symbol: v.symbol,
          timeframe: v.timeframe,
          scheduleType: backendScheduleType,
          scheduleConfig,
          parameters: merged,
        });
        message.success(t("common.updated"));
      } else {
        let templateId = v.templateId;
        if (
          typeof templateId === "string" &&
          templateId.startsWith("default-")
        ) {
          const def = (DEFAULT_TEMPLATES || []).find(
            (t: any) => String(t?.id) === String(templateId),
          );
          if (!def) {
            throw new Error(
              t("strategy.schedules.messages.defaultTemplateNotFound"),
            );
          }
          const created: any = await strategyTemplateApi.create({
            name: String(def?.name || ""),
            description: String(def?.description || ""),
            code: String(def?.code || ""),
            isPublic: !!def?.isPublic,
            parameters: [],
            tags: [],
          });
          if (!created?.id) {
            throw new Error(
              t("strategy.schedules.messages.importDefaultTemplateFailedNoId"),
            );
          }
          templateId = String(created.id);
        }
        const createdSchedule: any = await strategyScheduleV2Api.create({
          templateId,
          accountId: v.accountId,
          name: v.name,
          symbol: v.symbol,
          timeframe: v.timeframe,
          scheduleType: backendScheduleType,
          scheduleConfig,
          parameters: merged,
        });
        if (v.isActive) {
          const scheduleId = String((createdSchedule as WithId)?.id || "");
          if (scheduleId) {
            await strategyScheduleV2Api.toggle(scheduleId, true);
          }
        }
        message.success(t("common.created"));
      }
      setOpenEdit(false);
      setEditing(null);
      form.resetFields();
      await refresh();
    } catch (e: any) {
      message.error(e?.message || t("common.saveFailed"));
    } finally {
      setLoading(false);
    }
  };

  const onToggleActive = async (row: any, next: boolean) => {
    setLoading(true);
    try {
      await strategyScheduleV2Api.toggle(row.id, next);
      message.success(next ? t("common.enabled") : t("common.disabled"));
      await refresh();
    } catch (e: any) {
      message.error(e?.message || t("common.operationFailed"));
    } finally {
      setLoading(false);
    }
  };

  const onDelete = async (row: any) => {
    setLoading(true);
    try {
      await strategyScheduleV2Api.delete(row.id);
      message.success(t("common.deleted"));
      await refresh();
    } catch (e: any) {
      message.error(e?.message || t("common.deleteFailed"));
    } finally {
      setLoading(false);
    }
  };

  const onManualTrigger = async (row: any) => {
    setTriggering(true);
    setTriggerResult(null);
    setTriggerContext({ schedule: row, accountId: row.accountId });
    setOpenTrigger(true);

    try {
      const tpl: WithCode = await strategyTemplateApi.get(row.templateId);
      const code = String(tpl?.code || "");
      if (!code) {
        throw new Error(
          t("strategy.schedules.messages.templateCodeEmptyCannotExecute"),
        );
      }
      const exec = await pythonStrategyApi.execute({
        code,
        accountId: row.accountId,
        symbol: row.symbol,
        timeframe: row.timeframe,
      });
      if (!exec.success) {
        throw new Error(
          exec.error || t("strategy.schedules.messages.strategyExecuteFailed"),
        );
      }
      setTriggerResult({
        logs: exec.logs || [],
        signal: exec.signal,
        meta: { templateId: row.templateId, scheduleId: row.id },
      });
    } catch (e: any) {
      setTriggerResult({
        logs: [],
        signal: null,
        meta: {
          error: e?.message || t("strategy.schedules.messages.executeFailed"),
        },
      });
    } finally {
      setTriggering(false);
    }
  };

  const doOrderSend = async () => {
    if (!triggerContext?.schedule) return;
    const schedule = triggerContext.schedule;
    const raw = triggerResult?.signal as SignalLike | null;
    if (!raw) {
      message.error(t("strategy.schedules.messages.noOrderableSignal"));
      return;
    }

    const signal = raw;

    const rawAction = String(
      signal?.type ??
        signal?.signalType ??
        signal?.signal ??
        "",
    )
      .trim()
      .toLowerCase();
    const action = rawAction === "buy" || rawAction === "sell" ? rawAction : "";

    const volumeNum =
      typeof signal?.volume === "number"
        ? signal.volume
        : Number(signal?.volume);
    const volume = Number.isFinite(volumeNum) ? volumeNum : 0;

    if (!action || (action as string) === "hold") {
      message.error(t("strategy.schedules.messages.signalHoldCannotOrder"));
      return;
    }
    if (!(volume > 0)) {
      message.error(t("strategy.schedules.messages.volumeInvalid"));
      return;
    }

    const payload: {
      accountId: string;
      symbol: string;
      type: string;
      volume: number;
      price?: number;
      stopLoss?: number;
      takeProfit?: number;
      comment?: string;
      magicNumber?: bigint;
    } = {
      accountId: schedule.accountId,
      symbol: signal.symbol || schedule.symbol,
      type: action,
      volume,
      price:
        typeof signal?.price === "number"
          ? signal.price
          : Number(signal?.price || 0),
      stopLoss:
        typeof signal?.stopLoss === "number"
          ? signal.stopLoss
          : Number(signal?.stopLoss || 0),
      takeProfit:
        typeof signal?.takeProfit === "number"
          ? signal.takeProfit
          : Number(signal?.takeProfit || 0),
      comment: String(signal?.comment || ""),
      magicNumber: (typeof signal?.magicNumber === 'bigint'
        ? signal.magicNumber
        : typeof signal?.magicNumber === 'number'
          ? BigInt(Math.floor(signal.magicNumber as number))
          : undefined) as bigint | undefined,
    };

    setTriggering(true);
    try {
      const res = await tradingApi.orderSend(payload);
      if (res.error) {
        message.error(
          getTradingRiskToastMessage({
            riskCode: res.riskError?.code,
            error: res.error,
            message: res.message,
            fallback: res.error || t("strategy.schedules.messages.orderFailed"),
          }),
        );
        return;
      }
      message.success(t("strategy.schedules.messages.orderSubmitted"));
      setOpenTrigger(false);
      setTriggerContext(null);
      setTriggerResult(null);
    } catch (e: any) {
      message.error(e?.message || t("strategy.schedules.messages.orderFailed"));
    } finally {
      setTriggering(false);
    }
  };

  const scheduleType = Form.useWatch("scheduleType", form);
  const accountIdWatch = Form.useWatch("accountId", form);

  useEffect(() => {
    if (!openEdit) return;
    if (editing?.id) return;
    if (!accountIdWatch) {
      setSymbols([]);
      return;
    }
    void loadSymbols(accountIdWatch);
  }, [accountIdWatch, openEdit, editing?.id, loadSymbols]);

  return (
    <div className="p-4">
      <Card>
        <Space orientation="vertical" style={{ width: "100%" }}>
          <Space style={{ width: "100%", justifyContent: "space-between" }}>
            <Title level={4} style={{ margin: 0 }}>
              {t("strategy.schedules.title")}
            </Title>
            <Space>
              <Button onClick={() => void refresh()} loading={loading}>
                {t("common.refresh")}
              </Button>
              <Button type="primary" onClick={openCreate}>
                {t("strategy.schedules.actions.create")}
              </Button>
            </Space>
          </Space>

          <StatusResult error={error} onRetry={() => refresh()}>
            <ScheduleTable
              schedules={schedules}
              templates={templates}
              accounts={accounts}
              loading={loading}
              triggering={triggering}
              triggerContext={triggerContext}
              formatTime={formatTime}
              onEdit={openUpdate}
              onToggleActive={(row, next) => void onToggleActive(row, next)}
              onHealthCheck={(row) => {
                setHealthTarget(row);
                setHealthOpen(true);
                void loadScheduleHealth(row);
              }}
              onManualTrigger={(row) => void onManualTrigger(row)}
              onDelete={(row) => void onDelete(row)}
            />
          </StatusResult>
        </Space>
      </Card>

      <EditScheduleModal
        editing={editing}
        open={openEdit}
        loading={loading}
        form={form}
        templates={templatesForSelect}
        accounts={accounts}
        symbols={symbols}
        symbolsLoading={symbolsLoading}
        accountIdWatch={accountIdWatch}
        scheduleType={scheduleType}
        onCancel={() => {
          setOpenEdit(false);
          setEditing(null);
        }}
        onOk={() => void submitEdit()}
      />

      <TriggerModal
        open={openTrigger}
        triggering={triggering}
        triggerContext={triggerContext}
        triggerResult={triggerResult}
        onClose={() => {
          setOpenTrigger(false);
          setTriggerResult(null);
          setTriggerContext(null);
        }}
        onRerun={() => {
          if (triggerContext?.schedule) {
            void onManualTrigger(triggerContext.schedule);
          }
        }}
        onConfirmOrder={() => void doOrderSend()}
      />

      <ScheduleHealthModal
        open={healthOpen}
        target={healthTarget}
        loading={healthLoading}
        summary={healthSummary}
        onRefresh={() => {
          if (healthTarget) void loadScheduleHealth(healthTarget);
        }}
        onClose={() => {
          setHealthOpen(false);
          setHealthTarget(null);
          setHealthSummary(null);
        }}
        formatTime={formatTime}
      />
    </div>
  );
}
