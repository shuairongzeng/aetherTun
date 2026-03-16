import { useEffect, useMemo, useState } from "react";
import { BasicProxyConfigCard } from "./components/BasicProxyConfigCard";
import { getBackend } from "./backend";
import { FirstRunOnboarding } from "./components/FirstRunOnboarding";
import { OnboardingReminder } from "./components/OnboardingReminder";
import { PrimaryActionCard } from "./components/PrimaryActionCard";
import { QuickActionsCard } from "./components/QuickActionsCard";
import { RecentLogsCard } from "./components/RecentLogsCard";
import { StatusCard } from "./components/StatusCard";
import { useBasicProxySettings } from "./hooks/useBasicProxySettings";
import { useRuntimeState } from "./hooks/useRuntimeState";
import type { BasicProxySettings, OnboardingState, SaveBasicProxySettingsResult } from "./types";
import { getRuntimePhaseCopy } from "./components/runtimePhaseCopy";

type OnboardingStep = "welcome" | "config";

const hiddenOnboardingState: OnboardingState = {
  configExists: true,
  isDefaultProxyConfig: false,
  shouldShowOnboarding: false
};

function normalizeOnboardingState(raw?: Partial<Record<string, unknown>>): OnboardingState {
  return {
    configExists: Boolean(raw?.configExists ?? raw?.ConfigExists),
    isDefaultProxyConfig: Boolean(raw?.isDefaultProxyConfig ?? raw?.IsDefaultProxyConfig),
    shouldShowOnboarding: Boolean(raw?.shouldShowOnboarding ?? raw?.ShouldShowOnboarding)
  };
}

function deriveOnboardingStateFromSettings(settings: BasicProxySettings): OnboardingState {
  const isDefaultProxyConfig =
    settings.host === "127.0.0.1" && settings.port === 10808 && settings.type.toLowerCase() === "socks5";

  return {
    configExists: true,
    isDefaultProxyConfig,
    shouldShowOnboarding: isDefaultProxyConfig
  };
}

function nextStepText(phase: string, onboardingState: OnboardingState): string {
  if (onboardingState.shouldShowOnboarding) {
    return "先完成基础代理配置，再启动代理。";
  }

  if (phase === "running") {
    return "代理已运行；如需切换线路，保存配置后重启。";
  }

  if (phase === "starting") {
    return "正在等待核心上线，留意日志区的实时输出。";
  }

  if (phase === "stopping") {
    return "等待停止完成后，再进行下一次启动。";
  }

  return "确认代理地址和端口后，点击右上角主按钮启动。";
}

function logSummaryText(phase: string): string {
  if (phase === "running" || phase === "starting") {
    return "日志会持续同步到界面，不再依赖外部终端窗口。";
  }

  return "启动后日志会自动开始滚动；当前可先检查配置是否正确。";
}

export default function App() {
  const backend = useMemo(() => getBackend(), []);
  const {
    status,
    recentLogs,
    logsHint,
    busy,
    startCore,
    stopCore,
    openConfigFile,
    openLogDirectory,
    toggleAutoStart
  } = useRuntimeState();
  const {
    value: basicProxySettings,
    dirty: basicProxyDirty,
    saving: basicProxySaving,
    errors: basicProxyErrors,
    status: basicProxyStatus,
    setStatus: setBasicProxyStatus,
    updateValue: updateBasicProxyValue,
    resetDefaults: resetBasicProxyDefaults,
    saveSettings: saveBasicProxySettings
  } = useBasicProxySettings();
  const [onboardingState, setOnboardingState] = useState<OnboardingState>(hiddenOnboardingState);
  const [onboardingDismissed, setOnboardingDismissed] = useState(false);
  const [onboardingStep, setOnboardingStep] = useState<OnboardingStep>("welcome");

  useEffect(() => {
    let cancelled = false;

    async function loadOnboardingState() {
      if (!backend?.GetOnboardingState) {
        return;
      }

      const state = normalizeOnboardingState(await backend.GetOnboardingState());
      if (cancelled) {
        return;
      }

      setOnboardingState(state);
      if (state.shouldShowOnboarding) {
        setOnboardingStep("welcome");
      }
    }

    void loadOnboardingState();

    return () => {
      cancelled = true;
    };
  }, [backend]);

  const applyRestartPromptIfNeeded = async (result: SaveBasicProxySettingsResult) => {
    if (!result.requiresRestart) {
      return;
    }

    if (!window.confirm("新配置已保存，是否立即重启代理以应用新配置？")) {
      setBasicProxyStatus({
        tone: "info",
        text: "配置已保存，当前运行中的代理仍使用旧配置，重启后生效。"
      });
      return;
    }

    setBasicProxyStatus({
      tone: "info",
      text: "配置已保存，正在重启代理以应用新配置…"
    });
    await stopCore();
    await startCore();
  };

  const handleProxySettingsSaved = async (result?: SaveBasicProxySettingsResult) => {
    if (!result) {
      return;
    }

    const nextOnboardingState = deriveOnboardingStateFromSettings(result.settings);
    setOnboardingState(nextOnboardingState);
    if (!nextOnboardingState.shouldShowOnboarding) {
      setOnboardingDismissed(false);
      setOnboardingStep("welcome");
    }

    await applyRestartPromptIfNeeded(result);
  };

  const handleSaveBasicProxySettings = async () => {
    const result = await saveBasicProxySettings();
    await handleProxySettingsSaved(result);
  };

  const handleSkipOnboarding = () => {
    setOnboardingDismissed(true);
    setOnboardingStep("welcome");
  };

  const handleResumeOnboarding = () => {
    setOnboardingDismissed(false);
    setOnboardingStep("welcome");
  };

  const handleSaveOnboarding = async () => {
    const result = await saveBasicProxySettings();
    await handleProxySettingsSaved(result);
  };

  const showOnboardingOverlay = onboardingState.shouldShowOnboarding && !onboardingDismissed;
  const showOnboardingReminder = onboardingState.shouldShowOnboarding && onboardingDismissed;
  const phaseCopy = getRuntimePhaseCopy(status.phase);

  return (
    <main className="app-shell">
      <div className="dashboard-frame">
        <section className="dashboard-overview">
          <StatusCard status={status} />
          <PrimaryActionCard
            phase={status.phase}
            busy={busy}
            errorText={status.lastErrorText}
            onStart={startCore}
            onStop={stopCore}
          />
        </section>

        <section className="summary-strip" aria-label="概览摘要">
          <article className="summary-chip">
            <span className="summary-chip__label">当前阶段</span>
            <strong className="summary-chip__value">{phaseCopy.badgeLabel}</strong>
          </article>
          <article className="summary-chip">
            <span className="summary-chip__label">下一步建议</span>
            <strong className="summary-chip__value">{nextStepText(status.phase, onboardingState)}</strong>
          </article>
          <article className="summary-chip">
            <span className="summary-chip__label">日志行为</span>
            <strong className="summary-chip__value">{logSummaryText(status.phase)}</strong>
          </article>
        </section>

        {showOnboardingReminder ? (
          <OnboardingReminder onContinue={handleResumeOnboarding} onOpenConfigFile={openConfigFile} />
        ) : null}

        <section className="workspace-layout">
          <div className="workspace-main">
            <BasicProxyConfigCard
              value={basicProxySettings}
              dirty={basicProxyDirty}
              saving={basicProxySaving}
              errors={basicProxyErrors}
              status={basicProxyStatus}
              onChange={updateBasicProxyValue}
              onSave={handleSaveBasicProxySettings}
              onResetDefaults={resetBasicProxyDefaults}
              onOpenConfigFile={openConfigFile}
            />
          </div>

          <aside className="workspace-side">
            <QuickActionsCard
              onOpenConfigFile={openConfigFile}
              onOpenLogDirectory={openLogDirectory}
              onToggleAutoStart={toggleAutoStart}
            />
          </aside>

          <div className="workspace-logs">
            <RecentLogsCard entries={recentLogs} emptyText={logsHint} />
          </div>
        </section>
      </div>

      {showOnboardingOverlay ? (
        <FirstRunOnboarding
          step={onboardingStep}
          value={basicProxySettings}
          errors={basicProxyErrors}
          saving={basicProxySaving}
          statusText={basicProxyStatus.text}
          onChange={updateBasicProxyValue}
          onStart={() => setOnboardingStep("config")}
          onBack={() => setOnboardingStep("welcome")}
          onSkip={handleSkipOnboarding}
          onSave={handleSaveOnboarding}
        />
      ) : null}
    </main>
  );
}
