import { useCallback, useEffect, useMemo, useState } from "react";
import { getBackend } from "../backend";
import type {
  BackendApi,
  BasicProxySettings,
  BasicProxyStatus,
  SaveBasicProxySettingsResult
} from "../types";

const defaultBasicProxySettings: BasicProxySettings = {
  host: "127.0.0.1",
  port: 10808,
  type: "socks5"
};

type BasicProxyValidationErrors = Partial<Record<keyof BasicProxySettings, string>>;

function normalizeBasicProxySettings(raw?: Partial<Record<string, unknown>>): BasicProxySettings {
  return {
    host: String(raw?.host ?? raw?.Host ?? defaultBasicProxySettings.host),
    port: Number(raw?.port ?? raw?.Port ?? defaultBasicProxySettings.port),
    type: String(raw?.type ?? raw?.Type ?? defaultBasicProxySettings.type).toLowerCase()
  };
}

function normalizeSaveResult(raw?: Partial<Record<string, unknown>>): SaveBasicProxySettingsResult {
  const settings = normalizeBasicProxySettings((raw?.settings ?? raw?.Settings) as Partial<Record<string, unknown>>);

  return {
    settings,
    requiresRestart: Boolean(raw?.requiresRestart ?? raw?.RequiresRestart)
  };
}

function getErrorMessage(error: unknown): string {
  if (error instanceof Error) {
    return error.message;
  }

  if (typeof error === "string") {
    return error;
  }

  return "保存配置失败，请检查配置文件后重试。";
}

function validateBasicProxySettings(value: BasicProxySettings): BasicProxyValidationErrors {
  const errors: BasicProxyValidationErrors = {};

  if (!value.host.trim()) {
    errors.host = "代理地址不能为空";
  }

  if (!Number.isInteger(value.port) || value.port < 1 || value.port > 65535) {
    errors.port = "代理端口必须在 1-65535";
  }

  if (!["socks5", "http"].includes(value.type)) {
    errors.type = "代理类型必须是 SOCKS5 或 HTTP";
  }

  return errors;
}

export function useBasicProxySettings() {
  const backend = useMemo(() => getBackend(), []);
  const [value, setValue] = useState<BasicProxySettings>(defaultBasicProxySettings);
  const [savedValue, setSavedValue] = useState<BasicProxySettings>(defaultBasicProxySettings);
  const [saving, setSaving] = useState(false);
  const [status, setStatus] = useState<BasicProxyStatus>({ tone: "info", text: "" });
  const [errors, setErrors] = useState<BasicProxyValidationErrors>({});

  useEffect(() => {
    let cancelled = false;

    async function loadSettings() {
      if (!backend?.GetBasicProxySettings) {
        return;
      }

      try {
        const loaded = normalizeBasicProxySettings(await backend.GetBasicProxySettings());
        if (cancelled) {
          return;
        }

        setValue(loaded);
        setSavedValue(loaded);
        setErrors(validateBasicProxySettings(loaded));
      } catch (error) {
        if (!cancelled) {
          setStatus({ tone: "error", text: getErrorMessage(error) });
        }
      }
    }

    void loadSettings();

    return () => {
      cancelled = true;
    };
  }, [backend]);

  const updateValue = useCallback((next: BasicProxySettings) => {
    setValue(next);
    setErrors(validateBasicProxySettings(next));
  }, []);

  const resetDefaults = useCallback(() => {
    setValue(defaultBasicProxySettings);
    setErrors(validateBasicProxySettings(defaultBasicProxySettings));
    setStatus({ tone: "info", text: "已恢复默认值，点击“保存配置”后生效。" });
  }, []);

  const saveSettings = useCallback(async () => {
    const nextErrors = validateBasicProxySettings(value);
    setErrors(nextErrors);

    if (Object.keys(nextErrors).length > 0) {
      return undefined;
    }

    if (!backend?.SaveBasicProxySettings) {
      setStatus({ tone: "error", text: "当前环境不支持保存配置。" });
      return undefined;
    }

    setSaving(true);
    try {
      const result = normalizeSaveResult(await backend.SaveBasicProxySettings({
        host: value.host.trim(),
        port: value.port,
        type: value.type
      }));

      setValue(result.settings);
      setSavedValue(result.settings);
      setStatus({
        tone: result.requiresRestart ? "info" : "success",
        text: result.requiresRestart ? "配置已保存，重启后生效。" : "配置已保存。"
      });
      return result;
    } catch (error) {
      setStatus({ tone: "error", text: getErrorMessage(error) });
      return undefined;
    } finally {
      setSaving(false);
    }
  }, [backend, value]);

  const dirty =
    value.host !== savedValue.host || value.port !== savedValue.port || value.type !== savedValue.type;

  return {
    value,
    dirty,
    saving,
    errors,
    status,
    setStatus,
    updateValue,
    resetDefaults,
    saveSettings
  };
}
