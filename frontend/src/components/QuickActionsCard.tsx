type QuickActionsCardProps = {
  autoStartEnabled: boolean;
  onOpenConfigFile: () => Promise<void>;
  onOpenLogDirectory: () => Promise<void>;
  onToggleAutoStart: () => Promise<void>;
};

type ActionItem = {
  icon: string;
  label: string;
  hint: string;
  onClick: () => Promise<void>;
};

export function QuickActionsCard(props: QuickActionsCardProps) {
  const actions: ActionItem[] = [
    {
      icon: "📂",
      label: "打开配置文件",
      hint: "快速编辑高级配置",
      onClick: props.onOpenConfigFile
    },
    {
      icon: "📋",
      label: "查看日志目录",
      hint: "定位日志文件排查问题",
      onClick: props.onOpenLogDirectory
    },
    {
      icon: props.autoStartEnabled ? "✅" : "🔄",
      label: "开机自启",
      hint: props.autoStartEnabled ? "已启用，点击关闭" : "未启用，点击开启",
      onClick: props.onToggleAutoStart
    }
  ];

  return (
    <div className="card quick-actions-card">
      <h3 className="quick-actions-card__title">快捷操作</h3>
      <div className="quick-actions">
        {actions.map((action) => (
          <button
            key={action.label}
            className="quick-action"
            type="button"
            onClick={() => void action.onClick()}
          >
            <span className="quick-action__icon">{action.icon}</span>
            <span className="quick-action__text">
              <span className="quick-action__label">{action.label}</span>
              <span className="quick-action__hint">{action.hint}</span>
            </span>
            <span className="quick-action__chevron">→</span>
          </button>
        ))}
      </div>
    </div>
  );
}
