import { getRuntimePhaseCopy } from "./runtimePhaseCopy";

type PrimaryActionCardProps = {
  phase: string;
  busy: boolean;
  errorText?: string;
  onStart: () => Promise<void>;
  onStop: () => Promise<void>;
};

function helperText(phase: string, errorText?: string): string {
  if (errorText) {
    return errorText;
  }

  return getRuntimePhaseCopy(phase).actionText;
}

function actionHints(phase: string): string[] {
  if (phase === "running") {
    return ["如需切换代理地址，先保存配置再重启。", "关闭窗口后仍可从托盘继续停止或打开主界面。"];
  }

  if (phase === "starting") {
    return ["启动后即可在右侧日志区查看实时输出。", "如长时间无响应，请稍后重试或检查 UAC 提示。"];
  }

  if (phase === "stopping") {
    return ["正在等待后台核心退出。", "停止完成前，日志区仍会继续显示最后一批输出。"];
  }

  return ["启动后即可在右侧日志区查看实时输出。", "首次使用建议先确认基础代理地址、端口和类型。"];
}

function actionCapabilities(phase: string): string[] {
  if (phase === "running") {
    return ["支持托盘停止", "日志已同步", "可直接查看最近输出"];
  }

  if (phase === "starting") {
    return ["需要 UAC 授权", "正在等待核心上线", "日志即将开始同步"];
  }

  if (phase === "stopping") {
    return ["等待进程退出", "托盘状态会同步更新", "日志仍会保留最后输出"];
  }

  return ["需要 UAC 授权", "支持托盘驻留", "启动后日志会自动显示"];
}

export function PrimaryActionCard({
  phase,
  busy,
  errorText,
  onStart,
  onStop
}: PrimaryActionCardProps) {
  const isDisabled = busy || phase === "starting" || phase === "stopping";
  const phaseCopy = getRuntimePhaseCopy(phase);

  const handleClick = async () => {
    if (isDisabled) {
      return;
    }

    if (phase === "running") {
      await onStop();
      return;
    }

    await onStart();
  };

  return (
    <section className="panel primary-action-card">
      <div className="panel-header">
        <div>
          <p className="eyebrow eyebrow--subtle">主操作</p>
          <h2>{phase === "running" ? "代理已启动" : "准备启动代理"}</h2>
          <p className="panel-caption">{helperText(phase, errorText)}</p>
        </div>
      </div>

      <button
        className={`primary-button primary-button--wide${isDisabled ? " primary-button--disabled" : ""}`}
        type="button"
        onClick={() => void handleClick()}
        disabled={isDisabled}
        aria-busy={phase === "starting" || phase === "stopping"}
      >
        {phaseCopy.buttonLabel}
      </button>

      <div className="action-capabilities" aria-label="操作能力">
        {actionCapabilities(phase).map((item) => (
          <span key={item} className="action-capability">
            {item}
          </span>
        ))}
      </div>

      <ul className="action-hints" aria-label="操作提示">
        {actionHints(phase).map((hint) => (
          <li key={hint}>{hint}</li>
        ))}
      </ul>
    </section>
  );
}
