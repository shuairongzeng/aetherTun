import type { RuntimeStatus, BasicProxySettings } from "../types";
import { StatusCard } from "../components/StatusCard";
import { PrimaryActionCard } from "../components/PrimaryActionCard";
import { QuickActionsCard } from "../components/QuickActionsCard";

type OverviewPageProps = {
  status: RuntimeStatus;
  busy: boolean;
  autoStartEnabled: boolean;
  proxySettings: BasicProxySettings;
  onStart: () => Promise<void>;
  onStop: () => Promise<void>;
  onOpenConfigFile: () => Promise<void>;
  onOpenLogDirectory: () => Promise<void>;
  onToggleAutoStart: () => Promise<void>;
};

export function OverviewPage({
  status,
  busy,
  autoStartEnabled,
  proxySettings,
  onStart,
  onStop,
  onOpenConfigFile,
  onOpenLogDirectory,
  onToggleAutoStart
}: OverviewPageProps) {
  const phase = status.phase ?? "stopped";
  const isRunning = phase === "running";
  const endpoint = status.proxyEndpoint || `${proxySettings.host}:${proxySettings.port}`;
  const tunName = status.tunAdapterName || "Aether-TUN";

  return (
    <div className="page">
      <div className="overview-layout">
        <div className="card status-card">
          <StatusCard status={status} />
          <PrimaryActionCard
            phase={status.phase}
            busy={busy}
            errorText={status.lastErrorText}
            onStart={onStart}
            onStop={onStop}
          />
        </div>

        <div style={{ display: "flex", flexDirection: "column", gap: "20px" }}>
          <div className="card info-card">
            <h3 className="card__title">代理信息</h3>
            <div className="info-list" style={{ marginTop: "16px" }}>
              <div className="info-item">
                <span className="info-item__label">协议类型</span>
                <span className="info-item__value">{proxySettings.type.toUpperCase()}</span>
              </div>
              <div className="info-item">
                <span className="info-item__label">代理地址</span>
                <span className="info-item__value">{endpoint}</span>
              </div>
              <div className="info-item">
                <span className="info-item__label">TUN 网卡</span>
                <span className="info-item__value">{tunName}</span>
              </div>
              {isRunning ? (
                <div className="info-item">
                  <span className="info-item__label">运行状态</span>
                  <span className="info-item__value" style={{ color: "var(--color-primary)" }}>● 活跃</span>
                </div>
              ) : null}
            </div>
          </div>

          <QuickActionsCard
            autoStartEnabled={autoStartEnabled}
            onOpenConfigFile={onOpenConfigFile}
            onOpenLogDirectory={onOpenLogDirectory}
            onToggleAutoStart={onToggleAutoStart}
          />
        </div>
      </div>
    </div>
  );
}
