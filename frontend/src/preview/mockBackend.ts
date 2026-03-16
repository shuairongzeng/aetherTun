import type { BackendApi, BasicProxySettings, LogEntry, OnboardingState, RuntimeStatus } from "../types";

type PreviewScenario = "running" | "onboarding";

const defaultSettings: BasicProxySettings = {
  host: "127.0.0.1",
  port: 10808,
  type: "socks5"
};

const runningSettings: BasicProxySettings = {
  host: "demo.aether.local",
  port: 7890,
  type: "socks5"
};

const runningStatus: RuntimeStatus = {
  phase: "running",
  proxyEndpoint: "socks5://demo.aether.local:7890",
  tunAdapterName: "Aether-TUN"
};

const runningLogs: LogEntry[] = [
  {
    time: "2026-03-08T00:20:11.000Z",
    level: "info",
    source: "core",
    message: "代理握手完成，TUN 路由已接管目标流量。"
  },
  {
    time: "2026-03-08T00:20:08.000Z",
    level: "info",
    source: "dns",
    message: "FakeIP DNS 已开始处理首批查询。"
  },
  {
    time: "2026-03-08T00:20:05.000Z",
    level: "info",
    source: "launcher",
    message: "后台核心提权成功，运行状态已同步到控制面板。"
  }
];

function createOnboardingState(settings: BasicProxySettings): OnboardingState {
  const isDefaultProxyConfig =
    settings.host === defaultSettings.host &&
    settings.port === defaultSettings.port &&
    settings.type.toLowerCase() === defaultSettings.type;

  return {
    configExists: true,
    isDefaultProxyConfig,
    shouldShowOnboarding: isDefaultProxyConfig
  };
}

function readScenario(): PreviewScenario | undefined {
  if (typeof window === "undefined") {
    return undefined;
  }

  const value = new URLSearchParams(window.location.search).get("preview");
  if (value === "running" || value === "onboarding") {
    return value;
  }

  return undefined;
}

function buildScenarioState(scenario: PreviewScenario) {
  if (scenario === "onboarding") {
    return {
      status: { phase: "stopped" } satisfies RuntimeStatus,
      settings: { ...defaultSettings },
      onboarding: createOnboardingState(defaultSettings),
      logs: [] as LogEntry[]
    };
  }

  return {
    status: { ...runningStatus },
    settings: { ...runningSettings },
    onboarding: createOnboardingState(runningSettings),
    logs: [...runningLogs]
  };
}

export function getPreviewBackend(): BackendApi | undefined {
  const scenario = readScenario();
  if (!scenario || typeof window === "undefined") {
    return undefined;
  }

  const scopedWindow = window as Window & {
    __aetherPreviewBackend?: BackendApi;
    __aetherPreviewScenario?: PreviewScenario;
  };

  if (scopedWindow.__aetherPreviewBackend && scopedWindow.__aetherPreviewScenario === scenario) {
    return scopedWindow.__aetherPreviewBackend;
  }

  const state = buildScenarioState(scenario);

  const backend: BackendApi = {
    async GetStatus() {
      return { ...state.status };
    },
    async StartCore() {
      state.status = {
        phase: "running",
        proxyEndpoint: `socks5://${state.settings.host}:${state.settings.port}`,
        tunAdapterName: "Aether-TUN"
      };
      state.logs = [
        {
          time: new Date().toISOString(),
          level: "info",
          source: "preview",
          message: "预览模式：已模拟启动后台核心。"
        },
        ...state.logs
      ];
    },
    async StopCore() {
      state.status = {
        phase: "stopped",
        proxyEndpoint: "",
        tunAdapterName: "Aether-TUN"
      };
      state.logs = [
        {
          time: new Date().toISOString(),
          level: "info",
          source: "preview",
          message: "预览模式：已模拟停止后台核心。"
        },
        ...state.logs
      ];
    },
    async GetRecentLogs(limit: number) {
      return state.logs.slice(0, limit).map((entry) => ({ ...entry }));
    },
    async GetBasicProxySettings() {
      return { ...state.settings };
    },
    async SaveBasicProxySettings(input: BasicProxySettings) {
      const nextSettings = {
        host: input.host,
        port: input.port,
        type: input.type
      };
      const requiresRestart =
        state.status.phase === "running" &&
        (state.settings.host !== nextSettings.host ||
          state.settings.port !== nextSettings.port ||
          state.settings.type !== nextSettings.type);

      state.settings = nextSettings;
      state.onboarding = createOnboardingState(nextSettings);
      if (state.status.phase === "running") {
        state.status = {
          ...state.status,
          proxyEndpoint: `socks5://${nextSettings.host}:${nextSettings.port}`
        };
      }

      return {
        settings: { ...state.settings },
        requiresRestart
      };
    },
    async GetOnboardingState() {
      return { ...state.onboarding };
    },
    async OpenConfigFile() {},
    async OpenLogDirectory() {},
    async ToggleAutoStart() {}
  };

  scopedWindow.__aetherPreviewBackend = backend;
  scopedWindow.__aetherPreviewScenario = scenario;
  return backend;
}
