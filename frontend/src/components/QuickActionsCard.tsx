type QuickActionsCardProps = {
  onOpenConfigFile: () => Promise<void>;
  onOpenLogDirectory: () => Promise<void>;
  onToggleAutoStart: () => Promise<void>;
};

type QuickActionItemProps = {
  label: string;
  description: string;
  onClick: () => Promise<void>;
};

function QuickActionItem(props: QuickActionItemProps) {
  return (
    <div className="quick-action">
      <div className="quick-action__body">
        <strong className="quick-action__title">{props.label}</strong>
        <p className="quick-action__description">{props.description}</p>
      </div>
      <button className="secondary-button secondary-button--compact" type="button" onClick={() => void props.onClick()}>
        {props.label}
      </button>
    </div>
  );
}

export function QuickActionsCard(props: QuickActionsCardProps) {
  return (
    <article className="panel quick-actions-card">
      <div className="panel-header">
        <div>
          <p className="eyebrow eyebrow--subtle">常用入口</p>
          <h2>辅助操作</h2>
          <p className="panel-caption">把排障、配置和后续扩展入口收拢到一个固定区域。</p>
        </div>
      </div>

      <div className="quick-actions">
        <QuickActionItem
          label="打开配置文件"
          description="快速打开真实配置文件进行高级编辑。"
          onClick={props.onOpenConfigFile}
        />
        <QuickActionItem
          label="查看日志"
          description="直接跳到日志目录，便于排查问题。"
          onClick={props.onOpenLogDirectory}
        />
        <QuickActionItem
          label="开机自启"
          description="预留开机自启入口，后续会接入系统注册。"
          onClick={props.onToggleAutoStart}
        />
      </div>
    </article>
  );
}
