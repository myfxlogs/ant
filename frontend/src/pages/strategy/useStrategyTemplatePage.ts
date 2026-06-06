import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Form, message } from "antd";
import dayjs from "dayjs";
import { useLocation } from "react-router-dom";
import { useTranslation } from "react-i18next";
import type { StrategyTemplate } from "@/client/strategy";
import { strategyTemplateApi, type CreateTemplateRequest } from "@/client/strategy-schedules";
import { DEFAULT_TEMPLATES } from "./StrategyTemplatePage.defaults";
import { copyToClipboard } from "@/utils/clipboard";
import { getDeviceLocale } from "@/utils/date";
import { codeAssistApi, type RequiredParamSpec } from "@/client/codeAssist";
import { type QuickRangeKey, quickRangeLabel } from "./StrategyTemplatePage.utils";
import { applyQuickRange, doSubmitBacktest, openBacktestModal } from "./submitBacktest";
import { useStrategyTemplateRuns } from "./hooks/useStrategyTemplateRuns";
import { useAccountsAndSymbols } from "./hooks/useAccountsAndSymbols";
import { buildStrategyTemplateColumns } from "./StrategyTemplateColumns";

export function useStrategyTemplatePage() {
  const { t, i18n } = useTranslation();
  const location = useLocation();
  dayjs.locale(getDeviceLocale().toLowerCase().startsWith("zh") ? "zh-cn" : "en");

  const [templates, setTemplates] = useState<StrategyTemplate[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [modalVisible, setModalVisible] = useState(false);
  const [editingTemplate, setEditingTemplate] = useState<StrategyTemplate | null>(null);
  const [codeModalVisible, setCodeModalVisible] = useState(false);
  const [viewingCode, setViewingCode] = useState<string>("");
  const [codeValidating, setCodeValidating] = useState(false);
  const [lastValidatedCode, setLastValidatedCode] = useState<string>("");
  const [backtestRequiredParams, setBacktestRequiredParams] = useState<RequiredParamSpec[]>([]);
  const [backtestParamValues, setBacktestParamValues] = useState<Record<string, unknown>>({});
  const [form] = Form.useForm();
  const [highlightTemplateId, setHighlightTemplateId] = useState<string>("");
  const deepLinkNotifiedRef = useRef<boolean>(false);
  const [templateGroup, setTemplateGroup] = useState<"system" | "user">("system");
  const [quickRange, setQuickRange] = useState<QuickRangeKey>("CUSTOM");

  const { accounts, symbols, symbolsLoading, fetchAccounts, loadSymbols } = useAccountsAndSymbols();
  const [backtestModalOpen, setBacktestModalOpen] = useState(false);
  const [backtestForm] = Form.useForm();
  const [backtestSubmitting, setBacktestSubmitting] = useState(false);
  const [backtestTemplate, setBacktestTemplate] = useState<StrategyTemplate | null>(null);

  const runState = useStrategyTemplateRuns(t);
  const { runs, setRuns, runsLoading, fetchRuns, runDrawerOpen, setRunDrawerOpen,
    selectedRunId, setSelectedRunId, canceling, cancelRun, deleteRun,
    scoreOpen, setScoreOpen, scoreLoading, setScoreLoading, scoreRunId, setScoreRunId,
    scoreSnapshot, setScoreSnapshot, scoreValue, scheduleFlow, setScheduleFlow } = runState;

  const watchedRange = Form.useWatch("range", backtestForm) as [dayjs.Dayjs, dayjs.Dayjs] | undefined;

  useEffect(() => {
    if (!backtestModalOpen) return;
    backtestForm.setFieldsValue({ title: `${dayjs().format("YYYY-MM-DD HH:mm")} ${quickRangeLabel(t, quickRange)}` });
  }, [backtestModalOpen, quickRange, backtestForm, t]);

  const fetchTemplates = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const response = await strategyTemplateApi.list();
      setTemplates(response || []);
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : t("strategy.templates.messages.fetchTemplateListFailed");
      setError(msg); message.error(msg);
    } finally { setLoading(false); }
  }, [t]);

  useEffect(() => { fetchTemplates(); fetchAccounts(); fetchRuns(); }, [fetchTemplates, fetchAccounts, fetchRuns]);

  useEffect(() => {
    const onLang = () => { void fetchTemplates(); };
    i18n.on("languageChanged", onLang);
    return () => { i18n.off("languageChanged", onLang); };
  }, [i18n, fetchTemplates]);

  // Deep-link URL param handling
  useEffect(() => {
    const search = new URLSearchParams(location.search || "");
    const tid = String(search.get("templateId") || "").trim();
    const rid = String(search.get("runId") || "").trim();
    const openLatest = search.get("openLatestRun") === "1";
    const groupParam = String(search.get("group") || "").trim().toLowerCase();
    if (groupParam === "user" || groupParam === "system") setTemplateGroup(groupParam);
    if (tid) {
      setHighlightTemplateId(tid);
      if (groupParam !== "user" && groupParam !== "system") {
        const found = (templates || []).find((x) => String(x?.id || "") === tid);
        if (found) {
          const tags = Array.isArray(found?.tags) ? found.tags : [];
          const isSystem = Boolean(found?.isSystem) || tags.includes("preset") || tid.startsWith("default-");
          setTemplateGroup(isSystem ? "system" : "user");
        }
      }
    }
    if (rid) {
      setSelectedRunId(rid); setRunDrawerOpen(true);
      if (!deepLinkNotifiedRef.current) { deepLinkNotifiedRef.current = true; message.info(t("strategy.templates.messages.deepLinkNavigate")); }
      return;
    }
    if (openLatest && tid) {
      const timer = window.setTimeout(() => {
        const latest = (runs || []).filter((r) => String(r?.templateId || r?.template_id || "") === tid)
          .sort((a, b) => new Date(String(b?.createdAt || b?.created_at || "")).getTime() - new Date(String(a?.createdAt || a?.created_at || "")).getTime())[0];
        if (latest?.id) { setSelectedRunId(String(latest.id)); setRunDrawerOpen(true); }
        if (!deepLinkNotifiedRef.current) { deepLinkNotifiedRef.current = true; message.info(t("strategy.templates.messages.deepLinkNavigate")); }
      }, 300);
      return () => window.clearTimeout(timer);
    }
  }, [location.search, runs, t, setRunDrawerOpen, setSelectedRunId, templates]);

  const handleCreate = () => {
    setEditingTemplate(null); form.resetFields(); setLastValidatedCode(""); setModalVisible(true);
  };

  const handleEdit = (template: StrategyTemplate) => {
    setEditingTemplate(template); setLastValidatedCode("");
    form.setFieldsValue({ name: template.name, description: template.description, code: template.code, isPublic: template.isPublic });
    setModalVisible(true);
  };

  const validateTemplateCode = async () => {
    const code = String(form.getFieldValue("code") || "");
    if (!code.trim()) { message.warning(t("strategy.templates.messages.enterStrategyCode")); return false; }
    setCodeValidating(true);
    try {
      const ext = await codeAssistApi.validateExtended(code);
      if (!ext.valid) { message.error(ext.errors?.[0] || ext.warnings?.[0] || t("strategy.templates.messages.codeValidationNotPassed")); return false; }
      setLastValidatedCode(code); message.success(t("strategy.templates.messages.codeValidationPassed")); return true;
    } catch (e: unknown) {
      const err = e as { message?: string };
      message.error(String(err?.message || e || t("strategy.templates.messages.codeValidationFailed"))); return false;
    } finally { setCodeValidating(false); }
  };

  const handleSave = async (values: Record<string, unknown>) => {
    try {
      setCodeValidating(true);
      const ext = await codeAssistApi.validateExtended(String(values.code || ""));
      if (!ext.valid) { message.error(ext.errors?.[0] || ext.warnings?.[0] || t("strategy.templates.messages.codeValidationNotPassed")); return; }
      const data: CreateTemplateRequest = { name: String(values.name || ""), description: String(values.description || ""), code: String(values.code || ""), parameters: [], isPublic: Boolean(values.isPublic) || false, tags: [] };
      if (editingTemplate) { await strategyTemplateApi.update({ id: editingTemplate.id, ...data }); message.success(t("strategy.templates.messages.templateUpdated")); }
      else { await strategyTemplateApi.create(data); message.success(t("strategy.templates.messages.templateCreated")); setTemplateGroup("user"); }
      setModalVisible(false); fetchTemplates();
    } catch { message.error(t("common.saveFailed")); }
    finally { setCodeValidating(false); }
  };

  const handleDelete = async (id: string) => {
    try {
      await strategyTemplateApi.delete(id); message.success(t("strategy.templates.messages.templateDeleted")); fetchTemplates();
    } catch (error: unknown) {
      const err = error as { code?: string; rawMessage?: string; message?: string };
      const code = String(err?.code || "").toLowerCase();
      const msg = String(err?.rawMessage || err?.message || "");
      if (code.includes("permission") || msg.toLowerCase().includes("system template")) {
        message.error(t("strategy.templates.messages.systemTemplateReadOnly", "系统模板不可删除或修改")); return;
      }
      message.error(t("common.deleteFailed"));
    }
  };

  const fetchTemplateCodeIfNeeded = async (tpl: StrategyTemplate): Promise<StrategyTemplate> => {
    const id = String(tpl?.id || "");
    if (id.startsWith("default-")) return tpl;
    if (String(tpl?.code || "")) return tpl;
    const full = await strategyTemplateApi.get(id);
    return { ...tpl, ...full, code: String(full?.code || "") };
  };

  const handleViewCode = async (tpl: StrategyTemplate) => {
    try { const full = await fetchTemplateCodeIfNeeded(tpl); setViewingCode(String(full?.code || "")); setCodeModalVisible(true); }
    catch { message.error(t("strategy.templates.messages.readStrategyCodeFailed")); }
  };

  const handleOpenBacktestModal = async (template: StrategyTemplate) => {
    await openBacktestModal(t, template, fetchTemplateCodeIfNeeded, accounts, loadSymbols, backtestForm, setBacktestTemplate, setBacktestRequiredParams, setBacktestParamValues, setBacktestModalOpen, setQuickRange);
  };

  const handleApplyQuickRange = (key: QuickRangeKey) => {
    applyQuickRange(key, setQuickRange, backtestForm);
  };

  const submitBacktest = async () => {
    setBacktestSubmitting(true);
    try {
      const runId = await doSubmitBacktest(t, backtestTemplate, backtestForm, backtestRequiredParams, backtestParamValues, accounts, loadSymbols);
      if (runId) { setBacktestModalOpen(false); setSelectedRunId(runId); setRunDrawerOpen(true); await fetchRuns(); }
    } catch (e: unknown) {
      const err = e as { rawMessage?: string; code?: string | number; message?: string };
      message.error(String(err?.rawMessage || (err?.code !== undefined ? `code=${String(err.code)} ` : "") + (err?.message || "") || e) || t("strategy.templates.messages.backtestSubmitFailed"));
    } finally { setBacktestSubmitting(false); }
  };

  const handleCopyCode = async (code: string) => {
    const ok = await copyToClipboard(code);
    if (ok) message.success(t("strategy.templates.messages.codeCopied"));
    else message.error(t("strategy.templates.messages.copyFailed"));
  };

  const columns = useMemo(() => buildStrategyTemplateColumns({
    t, onBacktest: handleOpenBacktestModal, onViewCode: handleViewCode,
    onCopyToCreate: (record) => { form.setFieldsValue({ name: record.name + t("strategy.templates.copySuffix"), description: record.description, code: record.code, isPublic: false }); setEditingTemplate(null); setModalVisible(true); },
    onEdit: (record) => { void (async () => { try { const full = await fetchTemplateCodeIfNeeded(record); handleEdit(full); } catch { message.error(t("strategy.templates.messages.readStrategyCodeFailed")); } })(); },
    onDelete: handleDelete,
    onLaunchSchedule: (record) => { setScoreRunId(""); setScoreSnapshot(null); setScheduleFlow({ publishing: false, creating: false, enableAfterCreate: true, templateId: String(record?.id || ""), templateDraftId: undefined }); setScoreOpen(true); const defaultAccountId = accounts?.[0]?.id ? String(accounts[0].id) : ""; if (defaultAccountId) void loadSymbols(defaultAccountId); },
  }), [t, accounts, backtestTemplate]);

  const dataSource = useMemo(() => {
    if (templates.length > 0) return templates;
    interface DefaultTemplateItem { nameKey?: string; descriptionKey?: string; name?: string; description?: string; [key: string]: unknown; }
    return (DEFAULT_TEMPLATES as DefaultTemplateItem[] || []).map((tpl) => ({ ...tpl, name: tpl?.nameKey ? t(String(tpl.nameKey)) : tpl?.name, description: tpl?.descriptionKey ? t(String(tpl.descriptionKey)) : tpl?.description }));
  }, [templates, t]);

  const dialogProps = {
    edit: { modalVisible, editingTemplate, form, codeValidating, lastValidatedCode, setModalVisible, setLastValidatedCode, validateTemplateCode, handleSave },
    schedule: { scoreOpen, scoreLoading, scoreRunId, scoreSnapshot, scoreValue, scheduleFlow, setScheduleFlow, setRuns, accounts, symbols, symbolsLoading, loadSymbols, setScoreOpen, setScoreRunId, setScoreSnapshot, fetchTemplates, fetchRuns },
    backtest: { backtestModalOpen, backtestTemplate, backtestForm, backtestSubmitting, accounts, symbols, symbolsLoading, quickRange, watchedRange, backtestRequiredParams, backtestParamValues, setBacktestModalOpen, setBacktestRequiredParams, setBacktestParamValues, submitBacktest, applyQuickRange: handleApplyQuickRange, setQuickRange, loadSymbols },
    drawer: { runDrawerOpen, selectedRunId, setRunDrawerOpen, cancelRun, canceling },
    code: { codeModalVisible, viewingCode, setCodeModalVisible, handleCopyCode },
    runPanelActions: { setSelectedRunId, setRunDrawerOpen, setScoreRunId, setScoreOpen, setScoreLoading, setScoreSnapshot, setScheduleFlow, setRuns },
  };

  return {
    fetchTemplates, handleCreate, templates, loading, error,
    templateGroup, setTemplateGroup, highlightTemplateId,
    columns, dataSource, dialogProps, runs, runsLoading, deleteRun, fetchRuns,
  };
}
