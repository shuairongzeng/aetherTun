import { useEffect, useMemo, useState } from "react";
import { getBackend } from "./backend";
import { FirstRunOnboarding } from "./components/FirstRunOnboarding";
import { OnboardingReminder } from "./components/OnboardingReminder";
import { TabBar, type TabId } from "./components/TabBar";
import { useBasicProxySettings } from "./hooks/useBasicProxySettings";
import { useRuntimeState } from "./hooks/useRuntimeState";
import { OverviewPage } from "./pages/OverviewPage";
import { SettingsPage } from "./pages/SettingsPage";
import { LogsPage } from "./pages/LogsPage";
import type { BasicProxySettings, OnboardingState, SaveBasicProxySettingsResult } from "./types";

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

export default function App() {
  const backend = useMemo(() => getBackend(), []);
  const [activeTab, setActiveTab] = useState<TabId>("overview");
  const {
    status,
    recentLogs,
    logsHint,
    busy,
    startCore,
    stopCore,
    openConfigFile,
    openLogDirectory,
    autoStartEnabled,
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
    if (result) {
      // 首启向导保存成功后，无论配置是否与默认值相同，
      // 都直接关闭遮罩层进入主界面
      setOnboardingState({
        configExists: true,
        isDefaultProxyConfig: false,
        shouldShowOnboarding: false
      });
      setOnboardingDismissed(false);
      setOnboardingStep("welcome");
      await applyRestartPromptIfNeeded(result);
    }
  };

  const showOnboardingOverlay = onboardingState.shouldShowOnboarding && !onboardingDismissed;
  const showOnboardingReminder = onboardingState.shouldShowOnboarding && onboardingDismissed;

  return (
    <>
      <TabBar activeTab={activeTab} onTabChange={setActiveTab} statusPhase={status.phase} />

      <main className="app-shell">
        {showOnboardingReminder ? (
          <OnboardingReminder onContinue={handleResumeOnboarding} onOpenConfigFile={openConfigFile} />
        ) : null}

        {activeTab === "overview" ? (
          <OverviewPage
            status={status}
            busy={busy}
            autoStartEnabled={autoStartEnabled}
            proxySettings={basicProxySettings}
            onStart={startCore}
            onStop={stopCore}
            onOpenConfigFile={openConfigFile}
            onOpenLogDirectory={openLogDirectory}
            onToggleAutoStart={toggleAutoStart}
          />
        ) : null}

        {activeTab === "settings" ? (
          <SettingsPage
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
        ) : null}

        {activeTab === "logs" ? (
          <LogsPage entries={recentLogs} emptyText={logsHint} onOpenLogDirectory={openLogDirectory} />
        ) : null}
      </main>

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
    </>
  );
}
