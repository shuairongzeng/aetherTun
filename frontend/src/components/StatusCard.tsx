import type { RuntimeStatus } from "../types";
import { getRuntimePhaseCopy } from "./runtimePhaseCopy";

type StatusCardProps = {
  status: RuntimeStatus;
};

function indicatorClass(phase: string): string {
  switch (phase) {
    case "running":
      return "status-indicator--running";
    case "starting":
    case "stopping":
      return "status-indicator--starting";
    default:
      return "status-indicator--stopped";
  }
}

function indicatorIcon(phase: string): string {
  switch (phase) {
    case "running":
      return "✓";
    case "starting":
    case "stopping":
      return "⟳";
    default:
      return "○";
  }
}

function coreStateText(phase: string): string {
  switch (phase) {
    case "running":
      return "已连接";
    case "starting":
      return "连接中";
    case "stopping":
      return "停止中";
    default:
      return "未连接";
  }
}

function logStateText(phase: string): string {
  return phase === "running" || phase === "starting" ? "实时同步" : "待命";
}

function trayStateText(phase: string): string {
  return phase === "running" ? "关闭窗口后继续驻留" : "可随时最小化到托盘";
}

export function StatusCard({ status }: StatusCardProps) {
  const phase = status.phase ?? "stopped";
  const phaseCopy = getRuntimePhaseCopy(phase);

  return (
    <div className="status-indicator-wrapper" style={{ textAlign: "center" }}>
      <div className={`status-indicator ${indicatorClass(phase)}`}>
        <span className="status-indicator__icon">{indicatorIcon(phase)}</span>
      </div>
      <h2 className="status-card__phase">
        {phase === "running" ? "代理运行中" : phase === "starting" ? "正在启动…" : phase === "stopping" ? "正在停止…" : "代理已停止"}
      </h2>
      <p className="status-card__description">{phaseCopy.summaryText}</p>

      <div className="status-highlights">
        <div className="status-highlight">
          <span className="status-highlight__label">后台核心</span>
          <strong className="status-highlight__value">{coreStateText(phase)}</strong>
        </div>
        <div className="status-highlight">
          <span className="status-highlight__label">日志同步</span>
          <strong className="status-highlight__value">{logStateText(phase)}</strong>
        </div>
        <div className="status-highlight">
          <span className="status-highlight__label">托盘待命</span>
          <strong className="status-highlight__value">{trayStateText(phase)}</strong>
        </div>
      </div>
    </div>
  );
}
