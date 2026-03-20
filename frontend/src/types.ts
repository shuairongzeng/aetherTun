export type RuntimePhase = "stopped" | "starting" | "running" | "stopping" | "error";

export type RuntimeStatus = {
  phase: RuntimePhase | string;
  proxyEndpoint?: string;
  tunAdapterName?: string;
  lastErrorCode?: string;
  lastErrorText?: string;
};

export type BasicProxySettings = {
  host: string;
  port: number;
  type: string;
};

export type SaveBasicProxySettingsResult = {
  settings: BasicProxySettings;
  requiresRestart: boolean;
};

export type OnboardingState = {
  configExists: boolean;
  isDefaultProxyConfig: boolean;
  shouldShowOnboarding: boolean;
};

export type BasicProxyStatusTone = "success" | "error" | "info";

export type BasicProxyStatus = {
  tone: BasicProxyStatusTone;
  text: string;
};

export type LogEntry = {
  time?: string;
  level?: string;
  source?: string;
  message: string;
};

export type BackendApi = {
  GetStatus?: () => Promise<Record<string, unknown>>;
  StartCore?: () => Promise<void>;
  StopCore?: () => Promise<void>;
  GetRecentLogs?: (limit: number) => Promise<LogEntry[]>;
  GetBasicProxySettings?: () => Promise<BasicProxySettings | Record<string, unknown>>;
  SaveBasicProxySettings?: (
    input: BasicProxySettings
  ) => Promise<SaveBasicProxySettingsResult | Record<string, unknown>>;
  GetOnboardingState?: () => Promise<OnboardingState | Record<string, unknown>>;
  OpenConfigFile?: () => Promise<void>;
  OpenLogDirectory?: () => Promise<void>;
  ToggleAutoStart?: () => Promise<boolean>;
  GetAutoStartEnabled?: () => Promise<boolean>;
};
