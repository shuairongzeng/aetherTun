import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { getBackend } from "../backend";
import { getRuntimePhaseCopy } from "../components/runtimePhaseCopy";
import type { BackendApi, LogEntry, RuntimeStatus } from "../types";
import { LOG_FETCH_LIMIT, mergeLogEntries } from "./runtimeLogBuffer";

const fallbackStatus: RuntimeStatus = { phase: "stopped" };
const launchTimeoutMs = 15000;

type PendingAction = "starting" | "stopping" | undefined;

function normalizeStatus(raw: Record<string, unknown> | undefined): RuntimeStatus {
  if (!raw) {
    return fallbackStatus;
  }

  return {
    phase: String(raw.phase ?? raw.Phase ?? "stopped"),
    proxyEndpoint: String(raw.proxyEndpoint ?? raw.ProxyEndpoint ?? ""),
    tunAdapterName: String(raw.tunAdapterName ?? raw.TunAdapterName ?? ""),
    lastErrorCode: String(raw.lastErrorCode ?? raw.LastErrorCode ?? ""),
    lastErrorText: String(raw.lastErrorText ?? raw.LastErrorText ?? "")
  };
}

function errorMessage(error: unknown): string {
  if (error instanceof Error) {
    return error.message;
  }

  if (typeof error === "string") {
    return error;
  }

  return "操作失败，请检查后台核心进程是否存在。";
}

export function useRuntimeState() {
  const backend = useMemo(() => getBackend(), []);
  const [status, setStatus] = useState<RuntimeStatus>(fallbackStatus);
  const [recentLogs, setRecentLogs] = useState<LogEntry[]>([]);
  const [logsHint, setLogsHint] = useState(getRuntimePhaseCopy("stopped").logsEmptyText);
  const [busy, setBusy] = useState(false);
  const [actionErrorText, setActionErrorText] = useState<string>();
  const [pendingAction, setPendingAction] = useState<PendingAction>();
  const recentLogsRef = useRef<LogEntry[]>([]);
  const pendingActionRef = useRef<PendingAction>();
  const pendingTimeoutRef = useRef<number>();

  const clearPendingAction = useCallback(() => {
    if (pendingTimeoutRef.current !== undefined) {
      window.clearTimeout(pendingTimeoutRef.current);
      pendingTimeoutRef.current = undefined;
    }

    pendingActionRef.current = undefined;
    setPendingAction(undefined);
    setBusy(false);
  }, []);

  const beginPendingAction = useCallback((action: Exclude<PendingAction, undefined>) => {
    if (pendingTimeoutRef.current !== undefined) {
      window.clearTimeout(pendingTimeoutRef.current);
    }

    pendingActionRef.current = action;
    setPendingAction(action);
    setBusy(true);
    pendingTimeoutRef.current = window.setTimeout(() => {
      pendingActionRef.current = undefined;
      setPendingAction(undefined);
      setBusy(false);
      setActionErrorText(action === "starting" ? "启动超时，请重试。" : "停止超时，请重试。");
      setStatus((current) => ({
        ...current,
        phase: "error",
        lastErrorCode: action === "starting" ? "start_timeout" : "stop_timeout"
      }));
      setLogsHint(getRuntimePhaseCopy("error").logsEmptyText);
    }, launchTimeoutMs);
  }, []);

  const refresh = useCallback(async () => {
    if (!backend?.GetStatus) {
      setStatus(fallbackStatus);
      recentLogsRef.current = [];
      setRecentLogs([]);
      setLogsHint(getRuntimePhaseCopy("stopped").logsEmptyText);
      return;
    }

    const [nextStatus, nextLogs] = await Promise.all([
      backend.GetStatus(),
      backend.GetRecentLogs ? backend.GetRecentLogs(LOG_FETCH_LIMIT) : Promise.resolve([])
    ]);

    const normalizedStatus = normalizeStatus(nextStatus);
    setStatus(normalizedStatus);

    if (normalizedStatus.phase === "running" || normalizedStatus.phase === "starting") {
      setActionErrorText(undefined);
    }

    if (
      pendingActionRef.current === "starting" &&
      (normalizedStatus.phase === "running" || normalizedStatus.phase === "error")
    ) {
      clearPendingAction();
    }

    if (
      pendingActionRef.current === "stopping" &&
      (normalizedStatus.phase === "stopped" || normalizedStatus.phase === "error")
    ) {
      clearPendingAction();
    }

    const mergedLogs = mergeLogEntries(recentLogsRef.current, nextLogs);
    recentLogsRef.current = mergedLogs;
    setRecentLogs(mergedLogs);
    setLogsHint(mergedLogs.length === 0 ? getRuntimePhaseCopy(normalizedStatus.phase).logsEmptyText : undefined);
  }, [backend, clearPendingAction]);

  useEffect(() => {
    void refresh();
    const timer = window.setInterval(() => {
      void refresh();
    }, 1500);

    return () => window.clearInterval(timer);
  }, [refresh]);

  useEffect(() => {
    return () => {
      if (pendingTimeoutRef.current !== undefined) {
        window.clearTimeout(pendingTimeoutRef.current);
      }
    };
  }, []);

  const startCore = useCallback(async () => {
    if (!backend?.StartCore || busy || pendingAction) {
      return;
    }

    beginPendingAction("starting");
    setActionErrorText(undefined);
    setStatus((current) => ({
      ...current,
      phase: "starting",
      lastErrorCode: "",
      lastErrorText: ""
    }));
    setLogsHint(getRuntimePhaseCopy("starting").logsEmptyText);

    try {
      await backend.StartCore();
      await refresh();
    } catch (error) {
      clearPendingAction();
      setActionErrorText(errorMessage(error));
      setStatus((current) => ({
        ...current,
        phase: "error",
        lastErrorCode: "start_failed"
      }));
      setLogsHint(getRuntimePhaseCopy("error").logsEmptyText);
    }
  }, [backend, beginPendingAction, busy, clearPendingAction, pendingAction, refresh]);

  const stopCore = useCallback(async () => {
    if (!backend?.StopCore || busy || pendingAction) {
      return;
    }

    beginPendingAction("stopping");
    setActionErrorText(undefined);
    setStatus((current) => ({
      ...current,
      phase: "stopping",
      lastErrorCode: "",
      lastErrorText: ""
    }));
    setLogsHint(getRuntimePhaseCopy("stopping").logsEmptyText);

    try {
      await backend.StopCore();
      await refresh();
    } catch (error) {
      clearPendingAction();
      setActionErrorText(errorMessage(error));
      setStatus((current) => ({
        ...current,
        lastErrorCode: "stop_failed"
      }));
      setLogsHint(getRuntimePhaseCopy("error").logsEmptyText);
    }
  }, [backend, beginPendingAction, busy, clearPendingAction, pendingAction, refresh]);

  const openConfigFile = useCallback(async () => {
    await backend?.OpenConfigFile?.();
  }, [backend]);

  const openLogDirectory = useCallback(async () => {
    await backend?.OpenLogDirectory?.();
  }, [backend]);

  const toggleAutoStart = useCallback(async () => {
    await backend?.ToggleAutoStart?.();
  }, [backend]);

  const resolvedStatus = useMemo(() => {
    const nextStatus = actionErrorText
      ? {
          ...status,
          phase: status.phase === "running" ? status.phase : "error",
          lastErrorCode: status.lastErrorCode || "action_failed",
          lastErrorText: actionErrorText
        }
      : status;

    if (pendingAction === "starting" && nextStatus.phase !== "running" && nextStatus.phase !== "error") {
      return {
        ...nextStatus,
        phase: "starting",
        lastErrorCode: "",
        lastErrorText: ""
      };
    }

    if (pendingAction === "stopping" && nextStatus.phase !== "stopped" && nextStatus.phase !== "error") {
      return {
        ...nextStatus,
        phase: "stopping",
        lastErrorCode: "",
        lastErrorText: ""
      };
    }

    return nextStatus;
  }, [actionErrorText, pendingAction, status]);

  return {
    status: resolvedStatus,
    recentLogs,
    logsHint,
    busy,
    refresh,
    startCore,
    stopCore,
    openConfigFile,
    openLogDirectory,
    toggleAutoStart
  };
}
