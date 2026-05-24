import React, {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { Form, message, Space } from "antd";
import dayjs from "dayjs";
import "dayjs/locale/zh-cn";
import { strategyTemplateApi } from "@/client/strategy";
import type {
  StrategyTemplate,
  CreateTemplateRequest,
} from "@/client/strategy";
import { DEFAULT_TEMPLATES } from "./StrategyTemplatePage.defaults";
import { copyToClipboard } from "@/utils/clipboard";
import { accountApi } from "@/client/account";
import { marketApi } from "@/client/market";
import { pythonStrategyApi } from "@/client/pythonStrategy";
import { getDeviceLocale } from "@/utils/date";
import { codeAssistApi, type RequiredParamSpec } from "@/client/codeAssist";
import { wrapStrategyCodeWithParams } from "./backtestParamInjection";
import StrategyTemplateBacktestRunsPanel from "./components/StrategyTemplateBacktestRunsPanel";
import StrategyTemplateDialogs from "./components/StrategyTemplateDialogs";
import StrategyTemplateHeaderCard from "./components/StrategyTemplateHeaderCard";
import StrategyTemplateListCard from "./components/StrategyTemplateListCard";
import { useTranslation } from "react-i18next";
import { useLocation } from "react-router-dom";
import {
  type QuickRangeKey,
  quickRangeLabel,
  saveRunTitle,
} from "./StrategyTemplatePage.utils";
import { useStrategyTemplateRuns } from "./hooks/useStrategyTemplateRuns";
import { buildStrategyTemplateColumns } from "./StrategyTemplateColumns";

const StrategyTemplatePage: React.FC = () => {
  const { t, i18n } = useTranslation();
  const location = useLocation();
  dayjs.locale(
    getDeviceLocale().toLowerCase().startsWith("zh") ? "zh-cn" : "en",
  );

  const [templates, setTemplates] = useState<StrategyTemplate[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalVisible, setModalVisible] = useState(false);
  const [editingTemplate, setEditingTemplate] =
    useState<StrategyTemplate | null>(null);
  const [codeModalVisible, setCodeModalVisible] = useState(false);
  const [viewingCode, setViewingCode] = useState<string>("");
  const [codeValidating, setCodeValidating] = useState(false);
  // The exact code string that last passed extended strategy validation.
  // Save is blocked unless the current code === this value, forcing the user
  // to (re-)run validation after every edit.
  const [lastValidatedCode, setLastValidatedCode] = useState<string>("");

  // Required-parameter form state lives with the Backtest modal — strategies
  // do not request these at code-write time (symbol/timeframe/account come
  // from the schedule context, and user-defined params are filled at
  // backtest/schedule submit). See `submitBacktest` for how the values are
  // injected into the code that gets sent to the engine.
  const [backtestRequiredParams, setBacktestRequiredParams] = useState<
    RequiredParamSpec[]
  >([]);
  const [backtestParamValues, setBacktestParamValues] = useState<
    Record<string, unknown>
  >({});
  const [form] = Form.useForm();

  const [backtestModalOpen, setBacktestModalOpen] = useState(false);
  const [backtestForm] = Form.useForm();
  const [backtestSubmitting, setBacktestSubmitting] = useState(false);
  const [backtestTemplate, setBacktestTemplate] =
    useState<StrategyTemplate | null>(null);
  const [accounts, setAccounts] = useState<any[]>([]);
  const [symbols, setSymbols] = useState<{ value: string; label: string }[]>(
    [],
  );
  const [symbolsLoading, setSymbolsLoading] = useState(false);

  const runState = useStrategyTemplateRuns(t);
  const {
    runs,
    setRuns,
    runsLoading,
    fetchRuns,
    runDrawerOpen,
    setRunDrawerOpen,
    selectedRunId,
    setSelectedRunId,
    canceling,
    cancelRun,
    deleteRun,
    scoreOpen,
    setScoreOpen,
    scoreLoading,
    setScoreLoading,
    scoreRunId,
    setScoreRunId,
    scoreSnapshot,
    setScoreSnapshot,
    scoreValue,
    scheduleFlow,
    setScheduleFlow,
  } = runState;

  const [highlightTemplateId, setHighlightTemplateId] = useState<string>("");
  const deepLinkNotifiedRef = useRef<boolean>(false);

  // Which template group is visible: 'system' = preset (read-only), 'user' = user-created.
  const [templateGroup, setTemplateGroup] = useState<"system" | "user">(
    "system",
  );

  const [quickRange, setQuickRange] = useState<QuickRangeKey>("CUSTOM");
  const watchedRange = Form.useWatch("range", backtestForm) as
    | [dayjs.Dayjs, dayjs.Dayjs]
    | undefined;

  useEffect(() => {
    if (!backtestModalOpen) return;
    const nowText = dayjs().format("YYYY-MM-DD HH:mm");
    backtestForm.setFieldsValue({
      title: `${nowText} ${quickRangeLabel(t, quickRange)}`,
    });
  }, [backtestModalOpen, quickRange, backtestForm, t]);

  const fetchTemplates = useCallback(async () => {
    setLoading(true);
    try {
      const response = await strategyTemplateApi.list();
      setTemplates((response as any) || []);
    } catch (_error) {
      message.error(t("strategy.templates.messages.fetchTemplateListFailed"));
    } finally {
      setLoading(false);
    }
  }, [t]);

  const fetchAccounts = async () => {
    try {
      const data = (await accountApi.list()) as any[];
      setAccounts(data || []);
    } catch (_e) {
      setAccounts([]);
    }
  };

  const loadSymbols = async (accountId: string) => {
    if (!accountId) {
      setSymbols([]);
      return;
    }
    setSymbolsLoading(true);
    try {
      const list = await marketApi.getSymbols(accountId);
      const seen = new Set<string>();
      const opts = (list || [])
        .map((s) => String((s as any)?.symbol || "").trim())
        .filter((v) => v)
        .filter((v) => {
          if (seen.has(v)) return false;
          seen.add(v);
          return true;
        })
        .map((v) => ({ value: v, label: v }));
      setSymbols(opts);
    } catch (_e) {
      setSymbols([]);
    } finally {
      setSymbolsLoading(false);
    }
  };

  useEffect(() => {
    fetchTemplates();
    fetchAccounts();
    fetchRuns();
  }, [fetchTemplates, fetchRuns]);

  // Re-fetch templates whenever the UI language changes so backend-provided
  // i18n name/description for preset strategies update in place. Mirrors the
  // pattern used by ai/debate where `locale = i18n.language` is re-read per
  // request; here we refresh the cached list instead.
  useEffect(() => {
    const onLang = () => {
      void fetchTemplates();
    };
    i18n.on("languageChanged", onLang);
    return () => {
      i18n.off("languageChanged", onLang);
    };
  }, [i18n, fetchTemplates]);

  useEffect(() => {
    const search = new URLSearchParams(location.search || "");
    const tid = String(search.get("templateId") || "").trim();
    const rid = String(search.get("runId") || "").trim();
    const openLatest = search.get("openLatestRun") === "1";
    const groupParam = String(search.get("group") || "")
      .trim()
      .toLowerCase();
    if (groupParam === "user" || groupParam === "system") {
      setTemplateGroup(groupParam);
    }
    if (tid) {
      setHighlightTemplateId(tid);
      // Jump to the tab that contains the highlighted template so the row is visible.
      // Skip the auto-detect when the caller already specified the group explicitly,
      // since the templates list may not be loaded yet at this point.
      if (groupParam !== "user" && groupParam !== "system") {
        const found = (templates || []).find(
          (x: any) => String(x?.id || "") === tid,
        );
        if (found) {
          const tags = Array.isArray((found as any)?.tags)
            ? (found as any).tags
            : [];
          const isSystem =
            Boolean((found as any)?.isSystem) ||
            tags.includes("preset") ||
            tid.startsWith("default-");
          setTemplateGroup(isSystem ? "system" : "user");
        }
      }
    }

    if (rid) {
      setSelectedRunId(rid);
      setRunDrawerOpen(true);
      if (!deepLinkNotifiedRef.current) {
        deepLinkNotifiedRef.current = true;
        message.info(t("strategy.templates.messages.navigatedFromDebate"));
      }
      return;
    }
    if (openLatest && tid) {
      const timer = window.setTimeout(() => {
        const latest = (runs || [])
          .filter((r) => String(r?.templateId || r?.template_id || "") === tid)
          .sort(
            (a, b) =>
              new Date(String(b?.createdAt || b?.created_at || "")).getTime() -
              new Date(String(a?.createdAt || a?.created_at || "")).getTime(),
          )[0];
        if (latest?.id) {
          setSelectedRunId(String(latest.id));
          setRunDrawerOpen(true);
        }
        if (!deepLinkNotifiedRef.current) {
          deepLinkNotifiedRef.current = true;
          message.info(t("strategy.templates.messages.navigatedFromDebate"));
        }
      }, 300);
      return () => window.clearTimeout(timer);
    }
  }, [location.search, runs, t, setRunDrawerOpen, setSelectedRunId, templates]);

  const handleCreate = () => {
    setEditingTemplate(null);
    form.resetFields();
    setLastValidatedCode("");
    setModalVisible(true);
  };

  const handleEdit = (template: StrategyTemplate) => {
    setEditingTemplate(template);
    setLastValidatedCode("");
    form.setFieldsValue({
      name: template.name,
      description: template.description,
      code: template.code,
      isPublic: template.isPublic,
    });
    setModalVisible(true);
  };

  const validateTemplateCode = async () => {
    const code = String(form.getFieldValue("code") || "");
    if (!code.trim()) {
      message.warning(t("strategy.templates.messages.enterStrategyCode"));
      return false;
    }
    setCodeValidating(true);
    try {
      const ext = await codeAssistApi.validateExtended(code);
      if (!ext.valid) {
        message.error(
          ext.errors?.[0] ||
            ext.warnings?.[0] ||
            t("strategy.templates.messages.codeValidationNotPassed"),
        );
        return false;
      }
      setLastValidatedCode(code);
      message.success(t("strategy.templates.messages.codeValidationPassed"));
      return true;
    } catch (e: any) {
      message.error(
        String(
          e?.message ||
            e ||
            t("strategy.templates.messages.codeValidationFailed"),
        ),
      );
      return false;
    } finally {
      setCodeValidating(false);
    }
  };

  const handleSave = async (values: any) => {
    try {
      setCodeValidating(true);
      const ext = await codeAssistApi.validateExtended(
        String(values.code || ""),
      );
      if (!ext.valid) {
        message.error(
          ext.errors?.[0] ||
            ext.warnings?.[0] ||
            t("strategy.templates.messages.codeValidationNotPassed"),
        );
        return;
      }

      const data: CreateTemplateRequest = {
        name: values.name,
        description: values.description || "",
        code: values.code,
        parameters: [],
        isPublic: values.isPublic || false,
        tags: [],
      };

      if (editingTemplate) {
        await strategyTemplateApi.update({ id: editingTemplate.id, ...data });
        message.success(t("strategy.templates.messages.templateUpdated"));
      } else {
        await strategyTemplateApi.create(data);
        message.success(t("strategy.templates.messages.templateCreated"));
        // A freshly created template is always user-created; jump the user to
        // the "自建模板" tab so the new row is visible.
        setTemplateGroup("user");
      }
      setModalVisible(false);
      fetchTemplates();
    } catch (_error) {
      message.error(t("common.saveFailed"));
    } finally {
      setCodeValidating(false);
    }
  };

  const handleDelete = async (id: string) => {
    try {
      await strategyTemplateApi.delete(id);
      message.success(t("strategy.templates.messages.templateDeleted"));
      fetchTemplates();
    } catch (error: any) {
      // 后端对系统模板返回 permission_denied，给出明确提示。
      const code = String(error?.code || "").toLowerCase();
      const msg = String(error?.rawMessage || error?.message || "");
      if (
        code.includes("permission") ||
        msg.toLowerCase().includes("system template")
      ) {
        message.error(
          t(
            "strategy.templates.messages.systemTemplateReadOnly",
            "系统模板不可删除或修改",
          ),
        );
        return;
      }
      message.error(t("common.deleteFailed"));
    }
  };

  const fetchTemplateCodeIfNeeded = async (
    tpl: StrategyTemplate,
  ): Promise<StrategyTemplate> => {
    const id = String((tpl as any)?.id || "");
    const isDefault = id.startsWith("default-");
    if (isDefault) return tpl;
    const code = String((tpl as any)?.code || "");
    if (code) return tpl;
    const full: any = await strategyTemplateApi.get(id);
    return {
      ...tpl,
      ...(full as any),
      code: String((full as any)?.code || ""),
    } as StrategyTemplate;
  };

  const handleViewCode = async (tpl: StrategyTemplate) => {
    try {
      const full = await fetchTemplateCodeIfNeeded(tpl);
      setViewingCode(String((full as any)?.code || ""));
      setCodeModalVisible(true);
    } catch (_e) {
      message.error(t("strategy.templates.messages.readStrategyCodeFailed"));
    }
  };

  const openBacktestModal = async (template: StrategyTemplate) => {
    let full: StrategyTemplate;
    try {
      full = await fetchTemplateCodeIfNeeded(template);
      setBacktestTemplate(full);
    } catch (_e) {
      message.error(t("strategy.templates.messages.readStrategyCodeFailed"));
      return;
    }
    // Ask the static analyser which `params['xxx']` keys the strategy reads
    // so we can render a form section in the backtest modal. Failure is
    // non-fatal — the user can still submit a backtest with no extra params.
    setBacktestRequiredParams([]);
    setBacktestParamValues({});
    try {
      const ext = await codeAssistApi.validateExtended(
        String((full as any)?.code || ""),
      );
      if (ext.valid) {
        setBacktestRequiredParams(ext.parameters || []);
      }
    } catch {
      /* ignore — show an empty params form */
    }
    setBacktestModalOpen(true);
    setQuickRange("1D");
    const defaultAccountId = accounts?.[0]?.id ? String(accounts[0].id) : "";
    const defaultTo = dayjs();
    const defaultFrom = dayjs().add(-1, "day");
    const nowText = dayjs().format("YYYY-MM-DD HH:mm");
    backtestForm.setFieldsValue({
      title: `${nowText} ${quickRangeLabel(t, "1D")}`,
      accountId: defaultAccountId,
      symbol: "",
      timeframe: "H1",
      initialCapital: 10000,
      range: [defaultFrom, defaultTo],
      extraSymbols: [],
    });
    if (defaultAccountId) {
      await loadSymbols(defaultAccountId);
    }
  };

  const applyQuickRange = (key: QuickRangeKey) => {
    setQuickRange(key);
    if (key === "CUSTOM") return;
    const to = dayjs();
    const from =
      key === "1D"
        ? to.subtract(1, "day")
        : key === "3D"
          ? to.subtract(3, "day")
          : key === "1W"
            ? to.subtract(1, "week")
            : to.subtract(1, "year");
    backtestForm.setFieldsValue({ range: [from, to] });
  };

  const submitBacktest = async () => {
    if (!backtestTemplate) return;
    const values = await backtestForm.validateFields();
    setBacktestSubmitting(true);
    try {
      const fullTemplate = await fetchTemplateCodeIfNeeded(backtestTemplate);
      if (!String((fullTemplate as any)?.code || "")) {
        message.error(
          t("strategy.templates.messages.strategyCodeEmptyCannotBacktest"),
        );
        return;
      }
      const range = values.range as [dayjs.Dayjs, dayjs.Dayjs] | undefined;
      if (
        !range ||
        !range?.[0] ||
        !range?.[1] ||
        typeof (range?.[0] as any)?.toDate !== "function"
      ) {
        message.error(t("strategy.templates.messages.selectBacktestRange"));
        return;
      }
      const fromDate = range[0].toDate();
      const toDate = range[1].toDate();
      if (
        !(fromDate instanceof Date) ||
        isNaN(fromDate.getTime()) ||
        !(toDate instanceof Date) ||
        isNaN(toDate.getTime())
      ) {
        message.error(t("strategy.templates.messages.backtestRangeInvalid"));
        return;
      }
      const extraSymbols = Array.isArray(values.extraSymbols)
        ? (values.extraSymbols as any[])
            .map((s) => String(s))
            .filter((s) => !!s && s !== String(values.symbol))
        : [];

      // Enforce the strategy-defined required params at submit time. The
      // grpc StartBacktestRunRequest does not carry a `params` field today,
      // so we splice the chosen values into the code as a wrapper that
      // mutates ``context['params']`` before delegating to the user's
      // ``run``. Saved templates are NOT modified.
      const missingParams = backtestRequiredParams
        .filter((p) => p.required)
        .filter((p) => {
          const v = backtestParamValues[p.key];
          return v === undefined || v === null || v === "";
        });
      if (missingParams.length > 0) {
        message.error(
          t("strategy.codeAssist.fillRequiredParams", {
            defaultValue: "Please fill the required parameters: {{keys}}",
            keys: missingParams.map((m) => m.key).join(", "),
          }),
        );
        return;
      }
      const codeToSubmit = wrapStrategyCodeWithParams(
        String((fullTemplate as any)?.code || ""),
        backtestParamValues,
      );

      const resp = await pythonStrategyApi.startBacktestRun({
        code: codeToSubmit,
        accountId: String(values.accountId),
        symbol: String(values.symbol),
        timeframe: String(values.timeframe),
        initialCapital: Number(values.initialCapital || 10000),
        mode: "KLINE_RANGE",
        from: fromDate,
        to: toDate,
        templateId: String(backtestTemplate?.id || "").startsWith("default-")
          ? undefined
          : String(backtestTemplate?.id || ""),
        extraSymbols,
      });
      saveRunTitle(
        String(resp?.runId || ""),
        String(values.title || dayjs().format("YYYY-MM-DD")),
      );
      message.success(t("strategy.templates.messages.backtestSubmitted"));
      setBacktestModalOpen(false);
      setSelectedRunId(resp.runId);
      setRunDrawerOpen(true);
      await fetchRuns();
    } catch (e: any) {
      const errMsg =
        String(
          e?.rawMessage ||
            (e?.code !== undefined ? `code=${String(e.code)} ` : "") +
              (e?.message || "") ||
            e,
        ) || t("strategy.templates.messages.backtestSubmitFailed");
      message.error(errMsg);
    } finally {
      setBacktestSubmitting(false);
    }
  };

  const handleCopyCode = async (code: string) => {
    const ok = await copyToClipboard(code);
    if (ok) {
      message.success(t("strategy.templates.messages.codeCopied"));
      return;
    }
    message.error(t("strategy.templates.messages.copyFailed"));
  };

  const columns = useMemo(
    () =>
      buildStrategyTemplateColumns({
        t,
        onBacktest: openBacktestModal,
        onViewCode: handleViewCode,
        onCopyToCreate: (record) => {
          form.setFieldsValue({
            name: record.name + t("strategy.templates.copySuffix"),
            description: record.description,
            code: record.code,
            isPublic: false,
          });
          setEditingTemplate(null);
          setModalVisible(true);
        },
        onEdit: (record) => {
          void (async () => {
            try {
              const full = await fetchTemplateCodeIfNeeded(record);
              handleEdit(full);
            } catch (_e) {
              message.error(
                t("strategy.templates.messages.readStrategyCodeFailed"),
              );
            }
          })();
        },
        onDelete: handleDelete,
        onLaunchSchedule: (record) => {
          setScoreRunId("");
          setScoreSnapshot(null);
          setScheduleFlow({
            publishing: false,
            creating: false,
            enableAfterCreate: true,
            templateId: String(record?.id || ""),
            templateDraftId: undefined,
          });
          setScoreOpen(true);
          const defaultAccountId = accounts?.[0]?.id
            ? String(accounts[0].id)
            : "";
          if (defaultAccountId) {
            void loadSymbols(defaultAccountId);
          }
        },
      }),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [t, accounts, backtestTemplate],
  );

  const dataSource = useMemo(() => {
    if (templates.length > 0) return templates;
    return (DEFAULT_TEMPLATES || []).map((tpl: any) => ({
      ...tpl,
      name: tpl?.nameKey ? t(String(tpl.nameKey)) : tpl?.name,
      description: tpl?.descriptionKey
        ? t(String(tpl.descriptionKey))
        : tpl?.description,
    }));
  }, [templates, t]);

  const runPanelActions = {
    setSelectedRunId,
    setRunDrawerOpen,
    setScoreRunId,
    setScoreOpen,
    setScoreLoading,
    setScoreSnapshot,
    setScheduleFlow,
    setRuns,
  };
  const dialogEdit = {
    modalVisible,
    editingTemplate,
    form,
    codeValidating,
    lastValidatedCode,
    setModalVisible,
    setLastValidatedCode,
    validateTemplateCode,
    handleSave,
  };
  const dialogSchedule = {
    scoreOpen,
    scoreLoading,
    scoreRunId,
    scoreSnapshot,
    scoreValue,
    scheduleFlow,
    setScheduleFlow,
    setRuns,
    accounts,
    symbols,
    symbolsLoading,
    loadSymbols,
    setScoreOpen,
    setScoreRunId,
    setScoreSnapshot,
    fetchTemplates,
    fetchRuns,
  };
  const dialogBacktest = {
    backtestModalOpen,
    backtestTemplate,
    backtestForm,
    backtestSubmitting,
    accounts,
    symbols,
    symbolsLoading,
    quickRange,
    watchedRange,
    backtestRequiredParams,
    backtestParamValues,
    setBacktestModalOpen,
    setBacktestRequiredParams,
    setBacktestParamValues,
    submitBacktest,
    applyQuickRange,
    setQuickRange,
    loadSymbols,
  };

  return (
    <div style={{ padding: 24 }}>
      <Space orientation="vertical" size="large" style={{ width: "100%" }}>
        <StrategyTemplateHeaderCard
          onRefresh={fetchTemplates}
          onCreate={handleCreate}
        />

        {/* Templates list first (system / user tabs), then backtest reports. */}
        <StrategyTemplateListCard
          dataSource={dataSource as StrategyTemplate[]}
          templatesCount={templates.length}
          templateGroup={templateGroup}
          loading={loading}
          columns={columns}
          highlightTemplateId={highlightTemplateId}
          onTemplateGroupChange={setTemplateGroup}
        />

        <StrategyTemplateBacktestRunsPanel
          runs={runs}
          loading={runsLoading}
          onRefresh={fetchRuns}
          actions={runPanelActions}
          onDelete={deleteRun}
        />

        <StrategyTemplateDialogs
          edit={dialogEdit}
          schedule={dialogSchedule}
          code={{
            codeModalVisible,
            viewingCode,
            setCodeModalVisible,
            handleCopyCode,
          }}
          backtest={dialogBacktest}
          drawer={{
            runDrawerOpen,
            selectedRunId,
            setRunDrawerOpen,
            cancelRun,
            canceling,
          }}
        />
      </Space>
    </div>
  );
};

export default StrategyTemplatePage;
