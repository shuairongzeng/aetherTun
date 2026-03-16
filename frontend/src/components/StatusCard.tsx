import type { RuntimeStatus } from "../types";
import { getRuntimePhaseCopy } from "./runtimePhaseCopy";

type StatusCardProps = {
  status: RuntimeStatus;
};

export function StatusCard({ status }: StatusCardProps) {
  const phase = status.phase ?? "stopped";
  const phaseCopy = getRuntimePhaseCopy(phase);
  const coreState =
    phase === "running" ? "已连接" : phase === "starting" ? "连接中" : phase === "stopping" ? "停止中" : "未连接";
  const logState = phase === "running" || phase === "starting" ? "实时同步" : "待命";
  const trayState = phase === "running" ? "关闭窗口后继续驻留" : "可随时最小化到托盘";

  return (
    <section className="hero-card">
      <div className="hero-header">
        <div>
          <p className="eyebrow">运行概览</p>
          <h1>一键启动透明代理</h1>
          <p className="hero-copy">{phaseCopy.summaryText}</p>
        </div>
        <span className={`status-pill ${phaseCopy.badgeToneClass}`}>{phaseCopy.badgeLabel}</span>
      </div>

      <p className="secondary-text">核心输出会直接进入日志区，避免再弹出命令行窗口。</p>

      <div className="hero-highlights">
        <article className="hero-highlight">
          <span className="hero-highlight__label">后台核心</span>
          <strong className="hero-highlight__value">{coreState}</strong>
          <span className="hero-highlight__hint">当前控制面板与运行进程的连接状态</span>
        </article>
        <article className="hero-highlight">
          <span className="hero-highlight__label">日志同步</span>
          <strong className="hero-highlight__value">{logState}</strong>
          <span className="hero-highlight__hint">启动后日志会自动滚动并同步到界面</span>
        </article>
        <article className="hero-highlight">
          <span className="hero-highlight__label">托盘待命</span>
          <strong className="hero-highlight__value">{trayState}</strong>
          <span className="hero-highlight__hint">适合长时间挂后台时收起窗口</span>
        </article>
      </div>

      <dl className="meta-list">
        <div>
          <dt>代理地址</dt>
          <dd>{status.proxyEndpoint || phaseCopy.proxyPlaceholder}</dd>
        </div>
        <div>
          <dt>TUN 网卡</dt>
          <dd>{status.tunAdapterName || "Aether-TUN"}</dd>
        </div>
      </dl>
    </section>
  );
}
