export type RuntimePhaseCopy = {
  badgeLabel: string;
  badgeToneClass: string;
  buttonLabel: string;
  summaryText: string;
  actionText: string;
  proxyPlaceholder: string;
  logsEmptyText: string;
};

export function getRuntimePhaseCopy(phase: string): RuntimePhaseCopy {
  switch (phase) {
    case "starting":
      return {
        badgeLabel: "启动中",
        badgeToneClass: "status-pill--starting",
        buttonLabel: "正在启动",
        summaryText: "正在等待后台核心响应，请稍候。",
        actionText: "正在等待后台核心响应，请稍候。",
        proxyPlaceholder: "正在同步代理地址…",
        logsEmptyText: "正在连接后台核心，日志准备就绪后会显示在这里。"
      };
    case "running":
      return {
        badgeLabel: "运行中",
        badgeToneClass: "status-pill--running",
        buttonLabel: "停止代理",
        summaryText: "后台核心已连接，代理正在运行。",
        actionText: "后台核心已连接，代理正在运行。",
        proxyPlaceholder: "代理地址同步中…",
        logsEmptyText: "代理已运行，等待新的运行日志…"
      };
    case "stopping":
      return {
        badgeLabel: "停止中",
        badgeToneClass: "status-pill--stopping",
        buttonLabel: "正在停止",
        summaryText: "正在等待后台核心停止，请稍候。",
        actionText: "正在等待后台核心停止，请稍候。",
        proxyPlaceholder: "正在停止代理…",
        logsEmptyText: "正在停止后台核心，停止完成前的日志会显示在这里。"
      };
    case "error":
      return {
        badgeLabel: "启动失败",
        badgeToneClass: "status-pill--error",
        buttonLabel: "启动代理",
        summaryText: "后台核心状态异常，请查看最近日志或错误提示。",
        actionText: "后台核心状态异常，请查看最近日志或错误提示。",
        proxyPlaceholder: "等待后台核心连接…",
        logsEmptyText: "后台核心状态异常，请先查看错误提示或重新启动。"
      };
    default:
      return {
        badgeLabel: "未运行",
        badgeToneClass: "status-pill--stopped",
        buttonLabel: "启动代理",
        summaryText: "当前未连接到后台核心。",
        actionText: "当前未连接到后台核心。",
        proxyPlaceholder: "等待后台核心连接…",
        logsEmptyText: "后台核心未启动，启动后会在这里显示运行日志。"
      };
  }
}
